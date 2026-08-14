package apitools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/graphql"
	"github.com/OpenUdon/apitools/grpcproto"
	"github.com/OpenUdon/apitools/odata"
	"github.com/OpenUdon/apitools/openrpc"
	"github.com/OpenUdon/asyncapi"
	upstreamsmithy "github.com/OpenUdon/awssmithy"
	upstreamdiscovery "github.com/OpenUdon/googlediscovery"
	"gopkg.in/yaml.v3"
)

const (
	// LocalSourceDiscoveryVersion identifies the durable discovery report shape.
	LocalSourceDiscoveryVersion = "apitools.local-source-discovery.v1"

	DefaultLocalSourceMaxVisitedEntries = 10_000
	DefaultLocalSourceMaxCandidates     = 100

	APISourceKindAsyncAPI     = "asyncapi"
	APISourceKindGraphQL      = "graphql"
	APISourceKindOpenRPC      = "openrpc"
	APISourceKindGRPCProtobuf = "grpc-protobuf"
	APISourceKindOData        = "odata"
)

// LocalSource identifies a local document whose family is already known by
// the caller. Explicit kinds are required for otherwise ambiguous JSON or XML
// documents. ID is downstream authoring identity and is not inferred from a
// filename when supplied.
type LocalSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path"`
}

// LocalSourceDiscoveryOptions configures bounded, local-only API source
// discovery. Roots may name regular files or directories. Sources are explicit
// documents with caller-provided family information. No path is inferred from
// the process working directory.
type LocalSourceDiscoveryOptions struct {
	Roots             []string      `json:"roots,omitempty"`
	Sources           []LocalSource `json:"sources,omitempty"`
	Query             string        `json:"query,omitempty"`
	MaxVisitedEntries int           `json:"max_visited_entries,omitempty"`
	MaxCandidates     int           `json:"max_candidates,omitempty"`
	MaxBytes          int64         `json:"max_bytes,omitempty"`
}

// LocalSourceCandidate is one validated, prompt-safe local API source.
type LocalSourceCandidate struct {
	ID             string   `json:"id,omitempty"`
	Kind           string   `json:"kind"`
	SourceFamily   string   `json:"source_family"`
	Title          string   `json:"title,omitempty"`
	OperationCount int      `json:"operation_count"`
	Score          int      `json:"score"`
	Path           string   `json:"path"`
	SHA256         string   `json:"sha256"`
	Bytes          int64    `json:"bytes"`
	Provenance     string   `json:"provenance"`
	DuplicatePaths []string `json:"duplicate_paths,omitempty"`
}

// LocalSourceRejection records a candidate-shaped local file that could not be
// safely accepted.
type LocalSourceRejection struct {
	Path        string `json:"path"`
	Kind        string `json:"kind,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

// LocalSourceAmbiguity records a document for which content evidence does not
// select exactly one source family.
type LocalSourceAmbiguity struct {
	Path          string   `json:"path"`
	PossibleKinds []string `json:"possible_kinds,omitempty"`
	Message       string   `json:"message"`
	Remediation   string   `json:"remediation"`
}

// LocalSourceDiscoveryReport is a deterministic report of bounded local
// discovery. Limit diagnostics are blockers: callers should narrow roots or
// increase a bound explicitly before treating the candidate list as complete.
type LocalSourceDiscoveryReport struct {
	Version        string                 `json:"version"`
	Candidates     []LocalSourceCandidate `json:"candidates,omitempty"`
	Rejected       []LocalSourceRejection `json:"rejected,omitempty"`
	Ambiguous      []LocalSourceAmbiguity `json:"ambiguous,omitempty"`
	Diagnostics    []Diagnostic           `json:"diagnostics,omitempty"`
	VisitedEntries int                    `json:"visited_entries"`
	Truncated      bool                   `json:"truncated"`
}

type localSourceMetadata struct {
	kind           string
	title          string
	operationCount int
}

type localDiscoveryState struct {
	ctx           context.Context
	opts          LocalSourceDiscoveryOptions
	report        LocalSourceDiscoveryReport
	byDigest      map[string]int
	explicitPaths map[string]LocalSource
	seenPaths     map[string]bool
	stop          bool
}

var errStopLocalDiscovery = errors.New("stop local source discovery")

// DiscoverLocalSources scans only caller-provided local file or directory
// roots. It validates recognized source families, rejects symlinks and special
// files, deduplicates identical content by SHA-256 digest, and never performs a
// network request.
func DiscoverLocalSources(ctx context.Context, opts LocalSourceDiscoveryOptions) (LocalSourceDiscoveryReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	opts.MaxVisitedEntries = positiveOrDefault(opts.MaxVisitedEntries, DefaultLocalSourceMaxVisitedEntries)
	opts.MaxCandidates = positiveOrDefault(opts.MaxCandidates, DefaultLocalSourceMaxCandidates)
	opts.MaxBytes = resolvedLocalMaxBytes(opts.MaxBytes)
	state := &localDiscoveryState{
		ctx:           ctx,
		opts:          opts,
		report:        LocalSourceDiscoveryReport{Version: LocalSourceDiscoveryVersion},
		byDigest:      map[string]int{},
		explicitPaths: map[string]LocalSource{},
		seenPaths:     map[string]bool{},
	}

	if len(opts.Roots) == 0 && len(opts.Sources) == 0 {
		return state.report, fmt.Errorf("at least one explicit local source root is required")
	}
	for _, source := range opts.Sources {
		if strings.TrimSpace(source.Path) == "" {
			return state.report, fmt.Errorf("explicit local source path is required")
		}
		kind := normalizeAPISourceKind(source.Kind)
		if kind == "" {
			return state.report, fmt.Errorf("unsupported explicit API source kind %q", source.Kind)
		}
		abs, err := filepath.Abs(source.Path)
		if err != nil {
			return state.report, err
		}
		source.Path = filepath.Clean(abs)
		source.Kind = kind
		if prior, ok := state.explicitPaths[source.Path]; ok && (prior.Kind != source.Kind || prior.ID != source.ID) {
			return state.report, fmt.Errorf("local source path %q has conflicting explicit declarations", source.Path)
		}
		state.explicitPaths[source.Path] = source
	}

	rootSet := map[string]bool{}
	for _, root := range opts.Roots {
		if strings.TrimSpace(root) == "" {
			return state.report, fmt.Errorf("local source root is required")
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return state.report, err
		}
		rootSet[filepath.Clean(abs)] = true
	}
	for path := range state.explicitPaths {
		rootSet[path] = true
	}
	roots := make([]string, 0, len(rootSet))
	for root := range rootSet {
		roots = append(roots, root)
	}
	sort.Strings(roots)

	for _, root := range roots {
		if err := state.ctx.Err(); err != nil {
			return state.report, err
		}
		info, err := lstatLocalPathNoSymlinks(root)
		if err != nil {
			return state.report, fmt.Errorf("local source root %q: %w", root, err)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return state.report, fmt.Errorf("local source root %q is not a regular file or directory", root)
		}
		if _, explicit := state.explicitPaths[root]; explicit && info.IsDir() {
			return state.report, fmt.Errorf("explicit local source %q is a directory", root)
		}
		if info.IsDir() {
			err = filepath.WalkDir(root, state.visit)
		} else {
			if state.report.VisitedEntries >= state.opts.MaxVisitedEntries {
				state.limit("local.limit.entries", fmt.Sprintf("local discovery reached the %d-entry visit limit", state.opts.MaxVisitedEntries), "Narrow --source-root or explicitly increase MaxVisitedEntries.")
				break
			}
			state.report.VisitedEntries++
			err = state.inspect(root, info)
		}
		if err != nil && !errors.Is(err, errStopLocalDiscovery) {
			return state.report, err
		}
		if state.stop {
			break
		}
	}
	state.finish()
	return state.report, nil
}

func (state *localDiscoveryState) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := state.ctx.Err(); err != nil {
		return err
	}
	if state.report.VisitedEntries >= state.opts.MaxVisitedEntries {
		state.limit("local.limit.entries", fmt.Sprintf("local discovery reached the %d-entry visit limit", state.opts.MaxVisitedEntries), "Narrow --source-root or explicitly increase MaxVisitedEntries.")
		return errStopLocalDiscovery
	}
	state.report.VisitedEntries++
	if entry.Type()&fs.ModeSymlink != 0 {
		state.reject(path, "", "path.symlink", "local source path is a symlink", "Use a regular file inside the explicit root.")
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if entry.IsDir() {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	return state.inspect(path, info)
}

func (state *localDiscoveryState) inspect(path string, info fs.FileInfo) error {
	if err := state.ctx.Err(); err != nil {
		return err
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	path = filepath.Clean(path)
	if state.seenPaths[path] {
		return nil
	}
	state.seenPaths[path] = true
	explicit, hasExplicit := state.explicitPaths[path]
	if !info.Mode().IsRegular() {
		state.reject(path, explicit.Kind, "path.not_regular", "local source path is not a regular file", "Use a regular file.")
		return nil
	}
	if !hasExplicit && !hasLocalSourceExtension(path) {
		return nil
	}
	if !hasExplicit && isLocalSourceSidecar(path) {
		return nil
	}
	if info.Size() > state.opts.MaxBytes {
		state.reject(path, explicit.Kind, "file.too_large", fmt.Sprintf("local source is larger than %d bytes", state.opts.MaxBytes), "Choose a smaller source document or explicitly increase MaxBytes up to a reviewed bound.")
		return nil
	}
	content, err := readLocalSpecFile(path, state.opts.MaxBytes)
	if err != nil {
		state.reject(path, explicit.Kind, "file.read", err.Error(), "Use a stable, regular local file within the configured size bound.")
		return nil
	}
	metadata, matches, err := detectLocalSource(state.ctx, content, path, explicit.Kind)
	if err != nil {
		state.reject(path, explicit.Kind, "document.invalid", err.Error(), "Provide a valid document of the declared source family.")
		return nil
	}
	if metadata.kind == "" {
		if len(matches) > 1 || ambiguousStructuredDocument(content, path) {
			if len(matches) == 0 {
				matches = plausibleKindsForPath(path)
			}
			state.report.Ambiguous = append(state.report.Ambiguous, LocalSourceAmbiguity{
				Path:          path,
				PossibleKinds: matches,
				Message:       "local JSON or XML document does not contain unambiguous API source-family evidence",
				Remediation:   "Declare the document with an explicit source kind.",
			})
			return nil
		}
		if likelySourceFile(path) {
			state.reject(path, "", "document.unrecognized", "local file is not a valid supported API source document", "Provide a valid OpenAPI/Swagger, Google Discovery, AWS Smithy, AsyncAPI, GraphQL, OpenRPC, gRPC/protobuf, or OData document.")
		}
		return nil
	}

	digest := sha256.Sum256(content)
	digestText := hex.EncodeToString(digest[:])
	if index, ok := state.byDigest[digestText]; ok {
		state.report.Candidates[index].DuplicatePaths = append(state.report.Candidates[index].DuplicatePaths, path)
		return nil
	}
	candidate := LocalSourceCandidate{
		ID:             strings.TrimSpace(explicit.ID),
		Kind:           metadata.kind,
		SourceFamily:   metadata.kind,
		Title:          metadata.title,
		OperationCount: metadata.operationCount,
		Score:          ScoreText(state.opts.Query, metadata.title+" "+path),
		Path:           path,
		SHA256:         digestText,
		Bytes:          int64(len(content)),
		Provenance:     "local:" + filepath.ToSlash(path),
	}
	state.byDigest[digestText] = len(state.report.Candidates)
	state.report.Candidates = append(state.report.Candidates, candidate)
	if len(state.report.Candidates) >= state.opts.MaxCandidates {
		state.limit("local.limit.candidates", fmt.Sprintf("local discovery reached the %d-candidate acceptance limit", state.opts.MaxCandidates), "Narrow --source-root or explicitly increase MaxCandidates.")
		return errStopLocalDiscovery
	}
	return nil
}

func (state *localDiscoveryState) reject(path, kind, code, message, remediation string) {
	state.report.Rejected = append(state.report.Rejected, LocalSourceRejection{
		Path: path, Kind: kind, Code: code, Message: message, Remediation: remediation,
	})
}

func (state *localDiscoveryState) limit(code, message, remediation string) {
	if state.stop {
		return
	}
	state.stop = true
	state.report.Truncated = true
	state.report.Diagnostics = append(state.report.Diagnostics, Diagnostic{
		Severity: "error", Code: code, Message: message, Remediation: remediation,
	})
}

func (state *localDiscoveryState) finish() {
	sort.SliceStable(state.report.Candidates, func(i, j int) bool {
		left, right := state.report.Candidates[i], state.report.Candidates[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Path < right.Path
	})
	for i := range state.report.Candidates {
		sort.Strings(state.report.Candidates[i].DuplicatePaths)
	}
	sort.SliceStable(state.report.Rejected, func(i, j int) bool { return state.report.Rejected[i].Path < state.report.Rejected[j].Path })
	sort.SliceStable(state.report.Ambiguous, func(i, j int) bool { return state.report.Ambiguous[i].Path < state.report.Ambiguous[j].Path })
}

func detectLocalSource(ctx context.Context, content []byte, path, explicitKind string) (localSourceMetadata, []string, error) {
	if explicitKind != "" {
		metadata, err := validateLocalSourceKind(ctx, content, path, explicitKind)
		return metadata, []string{explicitKind}, err
	}
	candidates := detectedLocalSourceKinds(content, path)
	if len(candidates) != 1 {
		return localSourceMetadata{}, candidates, nil
	}
	metadata, err := validateLocalSourceKind(ctx, content, path, candidates[0])
	if err != nil {
		return localSourceMetadata{}, candidates, err
	}
	return metadata, candidates, nil
}

func detectedLocalSourceKinds(content []byte, path string) []string {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil
	}
	set := map[string]bool{}
	var root map[string]any
	if trimmed[0] == '{' && json.Unmarshal(trimmed, &root) == nil {
		if nonEmptyString(root["openapi"]) || nonEmptyString(root["swagger"]) {
			set[APISourceKindOpenAPI] = true
		}
		if nonEmptyString(root["asyncapi"]) {
			set[APISourceKindAsyncAPI] = true
		}
		if nonEmptyString(root["openrpc"]) {
			set[APISourceKindOpenRPC] = true
		}
		if looksLikeGoogleDiscovery(root) {
			set[APISourceKindGoogleDiscovery] = true
		}
		if looksLikeAWSSmithy(root) {
			set[APISourceKindAWSSmithy] = true
		}
		if looksLikeGraphQLJSON(root) {
			set[APISourceKindGraphQL] = true
		}
		if looksLikeGRPCDescriptorJSON(root) {
			set[APISourceKindGRPCProtobuf] = true
		}
		if looksLikeODataJSON(root) {
			set[APISourceKindOData] = true
		}
	} else if trimmed[0] == '<' {
		lower := strings.ToLower(string(trimmed[:min(len(trimmed), 4096)]))
		if strings.Contains(lower, "edmx") || strings.Contains(lower, "docs.oasis-open.org/odata") || strings.Contains(lower, "schemas.microsoft.com/ado/2009/11/edm") {
			set[APISourceKindOData] = true
		}
	} else {
		var yamlRoot map[string]any
		if yaml.Unmarshal(trimmed, &yamlRoot) == nil {
			if nonEmptyString(yamlRoot["openapi"]) || nonEmptyString(yamlRoot["swagger"]) {
				set[APISourceKindOpenAPI] = true
			}
			if nonEmptyString(yamlRoot["asyncapi"]) {
				set[APISourceKindAsyncAPI] = true
			}
		}
		text := string(trimmed[:min(len(trimmed), 64*1024)])
		lowerExt := strings.ToLower(filepath.Ext(path))
		if (lowerExt == ".graphql" || lowerExt == ".gql" || lowerExt == ".graphqls") && looksLikeGraphQLText(text) {
			set[APISourceKindGraphQL] = true
		}
		if lowerExt == ".proto" && looksLikeProtoText(text) {
			set[APISourceKindGRPCProtobuf] = true
		}
	}
	if isProtobufBinaryExtension(path) && !bytes.HasPrefix(trimmed, []byte("{")) {
		set[APISourceKindGRPCProtobuf] = true
	}
	kinds := make([]string, 0, len(set))
	for kind := range set {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func validateLocalSourceKind(ctx context.Context, content []byte, path, kind string) (localSourceMetadata, error) {
	if err := ctx.Err(); err != nil {
		return localSourceMetadata{}, err
	}
	switch kind {
	case APISourceKindOpenAPI:
		metadata, ok := specMetadata(ctx, content)
		if !ok {
			metadata, ok = localAuthoringOpenAPIMetadata(content)
		}
		if !ok {
			return localSourceMetadata{}, fmt.Errorf("document is not a structurally valid local OpenAPI or Swagger authoring source")
		}
		return localSourceMetadata{kind: kind, title: metadata.Title, operationCount: metadata.OperationCount}, nil
	case APISourceKindGoogleDiscovery:
		model, err := upstreamdiscovery.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid Google Discovery document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: firstNonEmpty(model.Title, model.Name), operationCount: len(model.Operations)}, nil
	case APISourceKindAWSSmithy:
		model, err := upstreamsmithy.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid AWS Smithy document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: firstNonEmpty(model.Title, model.ServiceID, model.AWSServiceID), operationCount: len(model.Operations)}, nil
	case APISourceKindAsyncAPI:
		model, err := asyncapi.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid AsyncAPI document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: model.Title, operationCount: len(model.Operations)}, nil
	case APISourceKindGraphQL:
		model, err := graphql.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid GraphQL document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: graphqlTitle(model, path), operationCount: len(model.OperationSummaries())}, nil
	case APISourceKindOpenRPC:
		model, err := openrpc.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid OpenRPC document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: model.Info.Title, operationCount: len(model.MethodSummaries())}, nil
	case APISourceKindGRPCProtobuf:
		model, err := grpcproto.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid gRPC/protobuf document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: grpcTitle(model, path), operationCount: len(model.MethodSummaries())}, nil
	case APISourceKindOData:
		model, err := odata.Parse(content)
		if err != nil {
			return localSourceMetadata{}, fmt.Errorf("invalid OData document: %w", err)
		}
		return localSourceMetadata{kind: kind, title: odataTitle(model, path), operationCount: len(model.OperationSummaries())}, nil
	default:
		return localSourceMetadata{}, fmt.Errorf("unsupported API source kind %q", kind)
	}
}

func localAuthoringOpenAPIMetadata(content []byte) (SpecMetadata, bool) {
	trimmed := bytes.TrimSpace(content)
	var root map[string]any
	if len(trimmed) == 0 {
		return SpecMetadata{}, false
	}
	if trimmed[0] == '{' {
		if json.Unmarshal(trimmed, &root) != nil {
			return SpecMetadata{}, false
		}
	} else if yaml.Unmarshal(trimmed, &root) != nil {
		return SpecMetadata{}, false
	}
	openapi, _ := root["openapi"].(string)
	swagger, _ := root["swagger"].(string)
	if !(strings.HasPrefix(strings.TrimSpace(openapi), "3.0") || strings.HasPrefix(strings.TrimSpace(openapi), "3.1") || strings.TrimSpace(swagger) == "2.0") || containsExternalRef(root, 0) {
		return SpecMetadata{}, false
	}
	info, ok := root["info"].(map[string]any)
	if !ok || !nonEmptyString(info["title"]) || !nonEmptyString(info["version"]) {
		return SpecMetadata{}, false
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		return SpecMetadata{}, false
	}
	operations := 0
	for path, raw := range paths {
		if !strings.HasPrefix(strings.TrimSpace(path), "/") {
			return SpecMetadata{}, false
		}
		item, ok := raw.(map[string]any)
		if !ok {
			return SpecMetadata{}, false
		}
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
			rawOperation, exists := item[method]
			if !exists {
				continue
			}
			operation, ok := rawOperation.(map[string]any)
			if !ok || !nonEmptyString(operation["operationId"]) {
				return SpecMetadata{}, false
			}
			operations++
		}
	}
	if operations == 0 {
		return SpecMetadata{}, false
	}
	title, _ := info["title"].(string)
	description, _ := info["description"].(string)
	return SpecMetadata{Title: strings.TrimSpace(title), Description: strings.TrimSpace(description), OpenAPI: strings.TrimSpace(openapi), Swagger: strings.TrimSpace(swagger), OperationCount: operations}, true
}

func looksLikeGoogleDiscovery(root map[string]any) bool {
	kind, _ := root["kind"].(string)
	return strings.HasPrefix(strings.TrimSpace(kind), "discovery#") && nonEmptyString(root["name"]) && nonEmptyString(root["version"])
}

func looksLikeAWSSmithy(root map[string]any) bool {
	_, hasSmithy := root["smithy"]
	shapes, hasShapes := root["shapes"].(map[string]any)
	if !hasSmithy || !hasShapes {
		return false
	}
	for _, raw := range shapes {
		shape, _ := raw.(map[string]any)
		if shape["type"] == "service" {
			return true
		}
	}
	return false
}

func looksLikeGraphQLJSON(root map[string]any) bool {
	if _, ok := root["__schema"].(map[string]any); ok {
		return true
	}
	data, _ := root["data"].(map[string]any)
	_, ok := data["__schema"].(map[string]any)
	return ok
}

func looksLikeGRPCDescriptorJSON(root map[string]any) bool {
	_, camel := root["file"].([]any)
	_, snake := root["file_descriptor_proto"].([]any)
	return camel || snake
}

func looksLikeODataJSON(root map[string]any) bool {
	return nonEmptyString(root["$Version"]) && (root["$EntityContainer"] != nil || len(root) > 1)
}

func looksLikeGraphQLText(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{"schema {", "type query", "type mutation", "query ", "mutation ", "subscription ", "fragment "} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func looksLikeProtoText(text string) bool {
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "syntax = \"proto") || strings.Contains(lower, "edition = \"")) && (strings.Contains(lower, "service ") || strings.Contains(lower, "message "))
}

func nonEmptyString(value any) bool {
	text, _ := value.(string)
	return strings.TrimSpace(text) != ""
}

func graphqlTitle(model *graphql.Model, path string) string {
	if model != nil {
		for _, name := range []string{model.Schema.QueryType, model.Schema.MutationType, model.Schema.SubscriptionType} {
			if strings.TrimSpace(name) != "" {
				return name
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func grpcTitle(model *grpcproto.Model, path string) string {
	if model != nil {
		for _, file := range model.Files {
			if file != nil && len(file.Services) > 0 && file.Services[0] != nil {
				return firstNonEmpty(file.Services[0].FullName, file.Services[0].Name)
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func odataTitle(model *odata.Model, path string) string {
	if model != nil && len(model.Schemas) > 0 && model.Schemas[0] != nil {
		return model.Schemas[0].Namespace
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func hasLocalSourceExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml", ".graphql", ".gql", ".graphqls", ".proto", ".pb", ".bin", ".desc", ".xml", ".edmx":
		return true
	default:
		return false
	}
}

func likelySourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".xml" {
		return true
	}
	base := strings.ToLower(filepath.Base(path))
	if isLocalSourceSidecar(path) {
		return false
	}
	for _, hint := range []string{"openapi", "swagger", "asyncapi", "discovery", "smithy", "graphql", "openrpc", "descriptor", "odata", "metadata"} {
		if strings.Contains(base, hint) {
			return true
		}
	}
	return conventionalSourceDirectory(path)
}

func isLocalSourceSidecar(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, suffix := range []string{
		".security.json",
		".security.yaml",
		".security.yml",
		".security-overlay.json",
		".security-overlay.yaml",
		".security-overlay.yml",
	} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

func conventionalSourceDirectory(path string) bool {
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(filepath.Dir(path))), "/") {
		switch part {
		case "openapi", "swagger", "google-discovery", "discovery", "aws-smithy", "smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "protobuf", "proto", "odata":
			return true
		}
	}
	return false
}

func ambiguousStructuredDocument(content []byte, path string) bool {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return (trimmed[0] == '{' && (ext == ".json" || ext == "")) || (trimmed[0] == '<' && (ext == ".xml" || ext == ".edmx" || ext == ""))
}

func plausibleKindsForPath(path string) []string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xml", ".edmx":
		return []string{APISourceKindOData}
	case ".json":
		return []string{APISourceKindAWSSmithy, APISourceKindGoogleDiscovery, APISourceKindGraphQL, APISourceKindGRPCProtobuf, APISourceKindOData, APISourceKindOpenAPI, APISourceKindOpenRPC, APISourceKindAsyncAPI}
	default:
		return nil
	}
}

func isProtobufBinaryExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pb", ".bin", ".desc":
		return true
	default:
		return false
	}
}

func positiveOrDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
