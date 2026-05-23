package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProviderArtifactCapability describes the strongest local catalog artifact
// surface available for a provider without fetching remote documents.
type ProviderArtifactCapability string

const (
	ProviderArtifactCapabilityExecutableOpenAPI    ProviderArtifactCapability = "executable-openapi"
	ProviderArtifactCapabilityOpenAPIReferenceOnly ProviderArtifactCapability = "openapi-reference-only"
	ProviderArtifactCapabilityDiscoveryOnly        ProviderArtifactCapability = "discovery-only"
	ProviderArtifactCapabilitySmithyOnly           ProviderArtifactCapability = "smithy-only"
	ProviderArtifactCapabilityDocsOnly             ProviderArtifactCapability = "docs-only"
	ProviderArtifactCapabilityAdvisoryOverlayOnly  ProviderArtifactCapability = "advisory-overlay-only"
	ProviderArtifactCapabilityMixedMetadata        ProviderArtifactCapability = "mixed-metadata"
	ProviderArtifactCapabilityUnknown              ProviderArtifactCapability = "unknown"
)

// ProviderResolutionOptions controls provider-name resolution and local
// artifact availability reporting.
type ProviderResolutionOptions struct {
	Catalog      Catalog
	Artifacts    []CatalogSpecArtifact
	ProviderKeys []string
	Query        string
}

// ProviderResolution records catalog identity, protocol capability, local
// artifact availability, and security overlay metadata for one provider.
type ProviderResolution struct {
	ProviderID              string                     `json:"provider_id"`
	DisplayName             string                     `json:"display_name,omitempty"`
	Aliases                 []string                   `json:"aliases,omitempty"`
	Category                string                     `json:"category,omitempty"`
	Capability              ProviderArtifactCapability `json:"capability"`
	OpenAPIAvailability     SpecAvailability           `json:"openapi_availability"`
	MachineSpecAvailability SpecAvailability           `json:"machine_spec_availability"`
	UserOpenAPINeed         UserOpenAPINeed            `json:"user_openapi_need"`
	SpecReferences          []ResolutionSpecReference  `json:"spec_references,omitempty"`
	Artifacts               []ResolutionArtifact       `json:"artifacts,omitempty"`
	SecurityOverlayIDs      []string                   `json:"security_overlay_ids,omitempty"`
	ManualFollowUps         []string                   `json:"manual_follow_ups,omitempty"`
}

// ResolutionSpecReference records a catalog source reference for resolution
// output.
type ResolutionSpecReference struct {
	ID              string          `json:"id"`
	Kind            SpecKind        `json:"kind"`
	Protocol        SpecProtocol    `json:"protocol"`
	ProtocolVersion string          `json:"protocol_version,omitempty"`
	UWSSourceType   string          `json:"uws_source_type,omitempty"`
	URL             string          `json:"url,omitempty"`
	SourceAuthority SourceAuthority `json:"source_authority,omitempty"`
	VerifiedAt      string          `json:"verified_at,omitempty"`
	SourceNote      string          `json:"source_note,omitempty"`
}

// ResolutionArtifact records one local artifact row joined from an existing
// cache or caller-supplied registry.
type ResolutionArtifact struct {
	ArtifactID     string            `json:"artifact_id,omitempty"`
	SpecRefID      string            `json:"spec_ref_id,omitempty"`
	Kind           string            `json:"kind,omitempty"`
	Protocol       SpecProtocol      `json:"protocol,omitempty"`
	UWSSourceType  string            `json:"uws_source_type,omitempty"`
	Path           string            `json:"path,omitempty"`
	OverlayPath    string            `json:"overlay_path,omitempty"`
	BuilderPath    string            `json:"builder_path,omitempty"`
	SourceURL      string            `json:"source_url,omitempty"`
	SHA256         string            `json:"sha256,omitempty"`
	Bytes          int64             `json:"bytes,omitempty"`
	Materializable bool              `json:"materializable"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ResolveProviders resolves built-in providers by key or whitespace-separated
// query string. It does not fetch documents, read artifact files, execute API
// operations, apply overlays to OpenAPI documents, or resolve credentials.
func ResolveProviders(query string) ([]ProviderResolution, error) {
	return ResolveProvidersWithOptions(ProviderResolutionOptions{Query: query})
}

// ResolveProvidersWithOptions resolves providers from explicit keys or a query
// string and joins caller-supplied local artifact rows.
func ResolveProvidersWithOptions(options ProviderResolutionOptions) ([]ProviderResolution, error) {
	cat := options.Catalog
	if cat.isZero() {
		cat = BuiltInCatalog()
	}
	if err := cat.Validate(); err != nil {
		return nil, err
	}
	keys := append([]string(nil), options.ProviderKeys...)
	if len(keys) == 0 {
		keys = strings.Fields(options.Query)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one provider key is required")
	}
	artifactsByProvider := materializeArtifactsByProvider(options.Artifacts)
	overlaysByProvider := materializeOverlaysByProvider(cat.ListSecurityOverlays())
	var out []ProviderResolution
	seen := map[string]struct{}{}
	for _, key := range keys {
		provider, ok := cat.FindProvider(key)
		if !ok {
			return nil, fmt.Errorf("unknown provider %q", key)
		}
		if _, exists := seen[provider.ID]; exists {
			continue
		}
		seen[provider.ID] = struct{}{}
		out = append(out, buildProviderResolution(provider, artifactsByProvider[provider.ID], overlaysByProvider[provider.ID]))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ProviderID < out[j].ProviderID })
	return out, nil
}

// MaterializeOptions controls offline provider artifact materialization.
type MaterializeOptions struct {
	Catalog                 Catalog
	ProviderKey             string
	TargetDir               string
	CacheDir                string
	Artifacts               []CatalogSpecArtifact
	IncludeSecurityOverlays bool
	WriteManifest           bool
}

// MaterializedArtifact records one copied local artifact.
type MaterializedArtifact struct {
	ArtifactID    string            `json:"artifact_id,omitempty"`
	SpecRefID     string            `json:"spec_ref_id,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	Protocol      SpecProtocol      `json:"protocol,omitempty"`
	UWSSourceType string            `json:"uws_source_type,omitempty"`
	SourcePath    string            `json:"source_path"`
	TargetPath    string            `json:"target_path"`
	SourceURL     string            `json:"source_url,omitempty"`
	SHA256        string            `json:"sha256,omitempty"`
	Bytes         int64             `json:"bytes,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// MaterializedSecurityOverlay records one emitted catalog security overlay.
type MaterializedSecurityOverlay struct {
	OverlayID  string                 `json:"overlay_id"`
	SpecRefID  string                 `json:"spec_ref_id,omitempty"`
	Status     AuthCompletenessStatus `json:"status"`
	TargetPath string                 `json:"target_path"`
}

// MaterializationReport records the result of materializing one provider.
type MaterializationReport struct {
	ProviderID       string                        `json:"provider_id"`
	DisplayName      string                        `json:"display_name,omitempty"`
	Capability       ProviderArtifactCapability    `json:"capability"`
	TargetDir        string                        `json:"target_dir"`
	Artifacts        []MaterializedArtifact        `json:"artifacts,omitempty"`
	SecurityOverlays []MaterializedSecurityOverlay `json:"security_overlays,omitempty"`
	ManualFollowUps  []string                      `json:"manual_follow_ups,omitempty"`
	Resolution       ProviderResolution            `json:"resolution"`
	ManifestPath     string                        `json:"manifest_path,omitempty"`
}

// Materialize copies local artifacts for a built-in provider into targetDir
// when matching artifact rows are supplied through MaterializeProvider. This
// convenience wrapper emits security overlays and a manifest but has no artifact
// registry input, so callers that need file copies should use
// MaterializeProvider with Artifacts and CacheDir.
func Materialize(ctx context.Context, providerID, targetDir string) (MaterializationReport, error) {
	return MaterializeProvider(ctx, MaterializeOptions{
		ProviderKey:             providerID,
		TargetDir:               targetDir,
		IncludeSecurityOverlays: true,
		WriteManifest:           true,
	})
}

// MaterializeProvider copies existing local catalog artifacts and emits
// provider security overlays as JSON metadata. It never downloads documents,
// executes API operations, applies overlays to OpenAPI documents, or resolves
// credentials.
func MaterializeProvider(ctx context.Context, options MaterializeOptions) (MaterializationReport, error) {
	if err := ctx.Err(); err != nil {
		return MaterializationReport{}, err
	}
	cat := options.Catalog
	if cat.isZero() {
		cat = BuiltInCatalog()
	}
	if err := cat.Validate(); err != nil {
		return MaterializationReport{}, err
	}
	provider, ok := cat.FindProvider(options.ProviderKey)
	if !ok {
		return MaterializationReport{}, fmt.Errorf("unknown provider %q", options.ProviderKey)
	}
	targetDir := strings.TrimSpace(options.TargetDir)
	if targetDir == "" {
		return MaterializationReport{}, fmt.Errorf("target dir is required")
	}
	providerDir := filepath.Join(targetDir, provider.ID)
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		return MaterializationReport{}, err
	}
	artifacts := materializeArtifactsByProvider(options.Artifacts)[provider.ID]
	overlays := materializeOverlaysByProvider(cat.ListSecurityOverlays())[provider.ID]
	resolution := buildProviderResolution(provider, artifacts, overlays)
	report := MaterializationReport{
		ProviderID:  provider.ID,
		DisplayName: provider.DisplayName,
		Capability:  resolution.Capability,
		TargetDir:   providerDir,
		Resolution:  resolution,
	}
	for _, artifact := range artifacts {
		if err := ctx.Err(); err != nil {
			return MaterializationReport{}, err
		}
		materialized, err := copyMaterializedArtifact(options.CacheDir, providerDir, provider, artifact)
		if err != nil {
			report.ManualFollowUps = append(report.ManualFollowUps, err.Error())
			continue
		}
		report.Artifacts = append(report.Artifacts, materialized)
	}
	if options.IncludeSecurityOverlays {
		for _, overlay := range overlays {
			if err := ctx.Err(); err != nil {
				return MaterializationReport{}, err
			}
			materialized, err := emitMaterializedSecurityOverlay(providerDir, overlay)
			if err != nil {
				return MaterializationReport{}, err
			}
			report.SecurityOverlays = append(report.SecurityOverlays, materialized)
		}
	}
	if len(report.Artifacts) == 0 {
		report.ManualFollowUps = append(report.ManualFollowUps, "No local catalog artifact was materialized; register or provide artifact rows before expecting importable files.")
	}
	report.ManualFollowUps = sortedUniqueStrings(report.ManualFollowUps)
	if options.WriteManifest {
		manifestPath := filepath.Join(providerDir, "provenance.json")
		if err := writeMaterializationJSON(manifestPath, report); err != nil {
			return MaterializationReport{}, err
		}
		report.ManifestPath = manifestPath
	}
	return report, nil
}

// ExportWorkflowArtifactsOptions controls copying provider artifacts into a
// workflow-owned artifact directory.
type ExportWorkflowArtifactsOptions struct {
	Catalog                 Catalog
	ProviderKeys            []string
	WorkflowDir             string
	ArtifactDir             string
	CacheDir                string
	Artifacts               []CatalogSpecArtifact
	IncludeSecurityOverlays bool
	WriteManifest           bool
}

// ExportReport records a multi-provider workflow artifact export.
type ExportReport struct {
	WorkflowDir  string                  `json:"workflow_dir"`
	ArtifactDir  string                  `json:"artifact_dir"`
	ManifestPath string                  `json:"manifest_path,omitempty"`
	Providers    []MaterializationReport `json:"providers,omitempty"`
}

// ExportWorkflowArtifacts materializes selected providers under a workflow
// directory and writes aggregate provenance. It performs no network access and
// does not execute provider operations.
func ExportWorkflowArtifacts(ctx context.Context, opts ExportWorkflowArtifactsOptions) (ExportReport, error) {
	if len(opts.ProviderKeys) == 0 {
		return ExportReport{}, fmt.Errorf("at least one provider key is required")
	}
	workflowDir := strings.TrimSpace(opts.WorkflowDir)
	if workflowDir == "" {
		return ExportReport{}, fmt.Errorf("workflow dir is required")
	}
	artifactDir := strings.TrimSpace(opts.ArtifactDir)
	if artifactDir == "" {
		artifactDir = "api-artifacts"
	}
	targetDir := filepath.Join(workflowDir, artifactDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return ExportReport{}, err
	}
	includeOverlays := opts.IncludeSecurityOverlays
	report := ExportReport{
		WorkflowDir: workflowDir,
		ArtifactDir: targetDir,
	}
	for _, key := range opts.ProviderKeys {
		materialized, err := MaterializeProvider(ctx, MaterializeOptions{
			Catalog:                 opts.Catalog,
			ProviderKey:             key,
			TargetDir:               targetDir,
			CacheDir:                opts.CacheDir,
			Artifacts:               opts.Artifacts,
			IncludeSecurityOverlays: includeOverlays,
			WriteManifest:           true,
		})
		if err != nil {
			return ExportReport{}, err
		}
		report.Providers = append(report.Providers, materialized)
	}
	sort.SliceStable(report.Providers, func(i, j int) bool {
		return report.Providers[i].ProviderID < report.Providers[j].ProviderID
	})
	if opts.WriteManifest {
		manifestPath := filepath.Join(targetDir, "provenance.json")
		if err := writeMaterializationJSON(manifestPath, report); err != nil {
			return ExportReport{}, err
		}
		report.ManifestPath = manifestPath
	}
	return report, nil
}

func buildProviderResolution(provider Provider, artifacts []CatalogSpecArtifact, overlays []SecurityOverlay) ProviderResolution {
	resolution := ProviderResolution{
		ProviderID:              provider.ID,
		DisplayName:             provider.DisplayName,
		Aliases:                 append([]string(nil), provider.Aliases...),
		Category:                provider.Category,
		OpenAPIAvailability:     provider.OfficialOpenAPIAvailability,
		MachineSpecAvailability: provider.OfficialMachineSpecAvailability,
		UserOpenAPINeed:         provider.UserOpenAPINeed,
		SecurityOverlayIDs:      overlayIDs(overlays),
	}
	refByID := map[string]SpecReference{}
	var protocols []SpecProtocol
	for _, ref := range provider.SpecReferences {
		protocol := ref.ProtocolClassification()
		refByID[ref.ID] = ref
		protocols = append(protocols, protocol.Protocol)
		resolution.SpecReferences = append(resolution.SpecReferences, ResolutionSpecReference{
			ID:              ref.ID,
			Kind:            ref.Kind,
			Protocol:        protocol.Protocol,
			ProtocolVersion: protocol.Version,
			UWSSourceType:   protocol.UWSSourceType(),
			URL:             ref.URL,
			SourceAuthority: ref.SourceAuthority,
			VerifiedAt:      ref.VerifiedAt,
			SourceNote:      ref.SourceNote,
		})
	}
	for _, artifact := range artifacts {
		specRefID := strings.TrimSpace(artifact.SpecRefID)
		protocol := artifactProtocolClassification(artifact, refByID)
		resolution.Artifacts = append(resolution.Artifacts, ResolutionArtifact{
			ArtifactID:     strings.TrimSpace(artifact.ArtifactID),
			SpecRefID:      specRefID,
			Kind:           strings.TrimSpace(artifact.Kind),
			Protocol:       protocol.Protocol,
			UWSSourceType:  protocol.UWSSourceType(),
			Path:           strings.TrimSpace(artifact.Path),
			OverlayPath:    strings.TrimSpace(artifact.OverlayPath),
			BuilderPath:    strings.TrimSpace(artifact.BuilderPath),
			SourceURL:      strings.TrimSpace(artifact.SourceURL),
			SHA256:         strings.TrimSpace(artifact.SHA256),
			Bytes:          artifact.Bytes,
			Materializable: strings.TrimSpace(artifact.Path) != "",
			Metadata:       cloneStringMap(artifact.Metadata),
		})
	}
	sort.SliceStable(resolution.SpecReferences, func(i, j int) bool {
		return resolution.SpecReferences[i].ID < resolution.SpecReferences[j].ID
	})
	sort.SliceStable(resolution.Artifacts, func(i, j int) bool {
		if resolution.Artifacts[i].SpecRefID == resolution.Artifacts[j].SpecRefID {
			return resolution.Artifacts[i].ArtifactID < resolution.Artifacts[j].ArtifactID
		}
		return resolution.Artifacts[i].SpecRefID < resolution.Artifacts[j].SpecRefID
	})
	resolution.Capability = providerArtifactCapability(protocols, resolution.Artifacts)
	if len(resolution.Artifacts) == 0 {
		resolution.ManualFollowUps = append(resolution.ManualFollowUps, "No local artifact row is registered for this provider.")
	}
	return resolution
}

func providerArtifactCapability(protocols []SpecProtocol, artifacts []ResolutionArtifact) ProviderArtifactCapability {
	hasOpenAPIRef := false
	hasDiscoveryRef := false
	hasSmithyRef := false
	hasHumanDocs := false
	for _, protocol := range protocols {
		switch protocol {
		case SpecProtocolOpenAPI, SpecProtocolSwagger:
			hasOpenAPIRef = true
		case SpecProtocolGoogleDiscovery:
			hasDiscoveryRef = true
		case SpecProtocolSmithy:
			hasSmithyRef = true
		case SpecProtocolHumanDocs:
			hasHumanDocs = true
		}
	}
	hasAdvisoryOverlay := false
	hasExecutableOpenAPI := false
	for _, artifact := range artifacts {
		if artifact.Kind == "advisory-overlay" {
			hasAdvisoryOverlay = true
			continue
		}
		if artifact.Protocol == SpecProtocolOpenAPI || artifact.Protocol == SpecProtocolSwagger {
			status := strings.TrimSpace(artifact.Metadata["validation_status"])
			if status == "" || strings.HasPrefix(status, "valid-") {
				hasExecutableOpenAPI = true
			}
		}
	}
	switch {
	case hasExecutableOpenAPI:
		return ProviderArtifactCapabilityExecutableOpenAPI
	case hasOpenAPIRef:
		return ProviderArtifactCapabilityOpenAPIReferenceOnly
	case hasAdvisoryOverlay:
		return ProviderArtifactCapabilityAdvisoryOverlayOnly
	case hasDiscoveryRef && !hasSmithyRef:
		return ProviderArtifactCapabilityDiscoveryOnly
	case hasSmithyRef && !hasDiscoveryRef:
		return ProviderArtifactCapabilitySmithyOnly
	case hasDiscoveryRef || hasSmithyRef:
		return ProviderArtifactCapabilityMixedMetadata
	case hasHumanDocs:
		return ProviderArtifactCapabilityDocsOnly
	default:
		return ProviderArtifactCapabilityUnknown
	}
}

func materializeArtifactsByProvider(artifacts []CatalogSpecArtifact) map[string][]CatalogSpecArtifact {
	out := map[string][]CatalogSpecArtifact{}
	for _, artifact := range artifacts {
		providerID := strings.TrimSpace(artifact.ProviderID)
		if providerID == "" {
			continue
		}
		out[providerID] = append(out[providerID], artifact)
	}
	for providerID := range out {
		sort.SliceStable(out[providerID], func(i, j int) bool {
			if out[providerID][i].SpecRefID == out[providerID][j].SpecRefID {
				return out[providerID][i].ArtifactID < out[providerID][j].ArtifactID
			}
			return out[providerID][i].SpecRefID < out[providerID][j].SpecRefID
		})
	}
	return out
}

func materializeOverlaysByProvider(overlays []SecurityOverlay) map[string][]SecurityOverlay {
	out := map[string][]SecurityOverlay{}
	for _, overlay := range overlays {
		providerID := strings.TrimSpace(overlay.ProviderID)
		if providerID == "" {
			continue
		}
		out[providerID] = append(out[providerID], overlay)
	}
	for providerID := range out {
		sortSecurityOverlays(out[providerID])
	}
	return out
}

func copyMaterializedArtifact(cacheDir, providerDir string, provider Provider, artifact CatalogSpecArtifact) (MaterializedArtifact, error) {
	sourcePath, err := resolveMaterializeSourcePath(cacheDir, artifact.Path)
	if err != nil {
		return MaterializedArtifact{}, fmt.Errorf("%s/%s: %w", provider.ID, artifactIDForMaterialize(artifact), err)
	}
	targetName := materializedFileName(artifact)
	targetPath := filepath.Join(providerDir, materializedSourceDir(artifact), targetName)
	digest, bytes, err := copyFileWithDigest(sourcePath, targetPath)
	if err != nil {
		return MaterializedArtifact{}, fmt.Errorf("%s/%s: %w", provider.ID, artifactIDForMaterialize(artifact), err)
	}
	protocol := artifactProtocolClassification(artifact, specReferencesByID(provider.SpecReferences))
	return MaterializedArtifact{
		ArtifactID:    artifactIDForMaterialize(artifact),
		SpecRefID:     strings.TrimSpace(artifact.SpecRefID),
		Kind:          strings.TrimSpace(artifact.Kind),
		Protocol:      protocol.Protocol,
		UWSSourceType: protocol.UWSSourceType(),
		SourcePath:    sourcePath,
		TargetPath:    targetPath,
		SourceURL:     strings.TrimSpace(artifact.SourceURL),
		SHA256:        digest,
		Bytes:         bytes,
		Metadata:      cloneStringMap(artifact.Metadata),
	}, nil
}

func emitMaterializedSecurityOverlay(providerDir string, overlay SecurityOverlay) (MaterializedSecurityOverlay, error) {
	targetPath := filepath.Join(providerDir, "security-overlays", overlay.ID+".json")
	if err := writeMaterializationJSON(targetPath, overlay); err != nil {
		return MaterializedSecurityOverlay{}, err
	}
	return MaterializedSecurityOverlay{
		OverlayID:  overlay.ID,
		SpecRefID:  strings.TrimSpace(overlay.SpecRefID),
		Status:     overlay.Status,
		TargetPath: targetPath,
	}, nil
}

func resolveMaterializeSourcePath(cacheDir, artifactPath string) (string, error) {
	path := strings.TrimSpace(artifactPath)
	if path == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	base := strings.TrimSpace(cacheDir)
	if base == "" {
		return "", fmt.Errorf("cache dir is required for relative artifact path %q", path)
	}
	cleanSlash := filepath.ToSlash(filepath.Clean(path))
	if cleanSlash == "." || strings.HasPrefix(cleanSlash, "../") || cleanSlash == ".." {
		return "", fmt.Errorf("artifact path %q escapes cache dir", path)
	}
	return filepath.Join(base, filepath.FromSlash(cleanSlash)), nil
}

func copyFileWithDigest(sourcePath, targetPath string) (string, int64, error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", 0, err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", 0, err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(target, hash), source)
	closeErr := target.Close()
	if copyErr != nil {
		return "", written, copyErr
	}
	if closeErr != nil {
		return "", written, closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), written, nil
}

func writeMaterializationJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return os.WriteFile(path, content, 0o600)
}

func artifactIDForMaterialize(artifact CatalogSpecArtifact) string {
	if id := strings.TrimSpace(artifact.ArtifactID); id != "" {
		return id
	}
	if id := strings.TrimSpace(artifact.SpecRefID); id != "" {
		return id
	}
	return "artifact"
}

func materializedSourceDir(artifact CatalogSpecArtifact) string {
	if dir := specProtocolClassificationForArtifactKind(artifact.Kind).SourceAlignedArtifactDir(); dir != "" {
		return dir
	}
	switch strings.TrimSpace(artifact.Kind) {
	case "advisory-overlay":
		return "openapi"
	default:
		return "artifacts"
	}
}

func specReferencesByID(refs []SpecReference) map[string]SpecReference {
	out := make(map[string]SpecReference, len(refs))
	for _, ref := range refs {
		out[ref.ID] = ref
	}
	return out
}

func artifactProtocolClassification(artifact CatalogSpecArtifact, refsByID map[string]SpecReference) SpecProtocolClassification {
	if ref, ok := refsByID[strings.TrimSpace(artifact.SpecRefID)]; ok {
		return ref.ProtocolClassification()
	}
	return specProtocolClassificationForArtifactKind(artifact.Kind)
}

func specProtocolClassificationForArtifactKind(kind string) SpecProtocolClassification {
	switch strings.TrimSpace(kind) {
	case "advisory-overlay":
		return SpecProtocolClassification{Protocol: SpecProtocolOpenAPI}
	case "swagger":
		return SpecProtocolClassification{Protocol: SpecProtocolSwagger, Version: "2.0"}
	default:
		return specProtocolClassificationForKind(SpecKind(strings.TrimSpace(kind)))
	}
}

func materializedFileName(artifact CatalogSpecArtifact) string {
	id := artifactIDForMaterialize(artifact)
	path := strings.TrimSpace(artifact.Path)
	lowerPath := strings.ToLower(path)
	ext := filepath.Ext(path)
	if strings.HasSuffix(lowerPath, ".tar.gz") {
		ext = ".tar.gz"
	}
	if ext == "" {
		switch strings.TrimSpace(artifact.Kind) {
		case "advisory-overlay", "openapi", "swagger", "google-discovery", "smithy-json", "openapi-index":
			ext = ".json"
		default:
			ext = ".artifact"
		}
	}
	return safeFileStem(id) + ext
}

func safeFileStem(value string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		return "artifact"
	}
	return out
}

func overlayIDs(overlays []SecurityOverlay) []string {
	var out []string
	for _, overlay := range overlays {
		out = append(out, overlay.ID)
	}
	return sortedUniqueStrings(out)
}
