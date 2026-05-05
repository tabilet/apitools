package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// InventoryDocument is one local OpenAPI or Swagger document to inspect.
type InventoryDocument struct {
	Name    string `json:"name,omitempty"`
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Content []byte `json:"-"`
}

// InventoryOptions configures operation inventory extraction.
type InventoryOptions struct {
	Documents []InventoryDocument `json:"documents,omitempty"`
	Query     string              `json:"query,omitempty"`
	Limit     int                 `json:"limit,omitempty"`
}

// OperationInventory is a prompt-safe summary of OpenAPI operations.
type OperationInventory struct {
	Documents       []DocumentSummary  `json:"documents,omitempty"`
	Operations      []OperationSummary `json:"operations,omitempty"`
	Diagnostics     []Diagnostic       `json:"diagnostics,omitempty"`
	ReadinessIssues []ReadinessIssue   `json:"readiness_issues,omitempty"`
}

// DocumentSummary describes one source document in an inventory.
type DocumentSummary struct {
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	URL            string `json:"url,omitempty"`
	Title          string `json:"title,omitempty"`
	Description    string `json:"description,omitempty"`
	OpenAPI        string `json:"openapi,omitempty"`
	Swagger        string `json:"swagger,omitempty"`
	OperationCount int    `json:"operation_count,omitempty"`
}

// OperationSummary describes one prompt-safe OpenAPI operation.
type OperationSummary struct {
	ID              string              `json:"id"`
	DocumentName    string              `json:"document_name,omitempty"`
	DocumentPath    string              `json:"document_path,omitempty"`
	DocumentURL     string              `json:"document_url,omitempty"`
	OperationID     string              `json:"operation_id,omitempty"`
	Method          string              `json:"method"`
	Path            string              `json:"path"`
	Summary         string              `json:"summary,omitempty"`
	Description     string              `json:"description,omitempty"`
	Tags            []string            `json:"tags,omitempty"`
	Extensions      map[string]string   `json:"extensions,omitempty"`
	Parameters      []ParameterSummary  `json:"parameters,omitempty"`
	RequestBody     *RequestBodySummary `json:"request_body,omitempty"`
	Security        []SecuritySummary   `json:"security,omitempty"`
	Score           int                 `json:"score,omitempty"`
	Provenance      string              `json:"provenance,omitempty"`
	ReadinessIssues []ReadinessIssue    `json:"readiness_issues,omitempty"`
}

// ParameterSummary describes an operation parameter without examples or values.
type ParameterSummary struct {
	Name        string `json:"name,omitempty"`
	In          string `json:"in,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Type        string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Ref         string `json:"ref,omitempty"`
}

// RequestBodySummary describes a request body without examples or defaults.
type RequestBodySummary struct {
	Description        string                `json:"description,omitempty"`
	Required           bool                  `json:"required,omitempty"`
	ContentTypes       []string              `json:"content_types,omitempty"`
	Schema             *SchemaSummary        `json:"schema,omitempty"`
	Ref                string                `json:"ref,omitempty"`
	Fields             []RequestFieldSummary `json:"fields,omitempty"`
	RequiredFieldPaths []string              `json:"required_field_paths,omitempty"`
}

// SchemaSummary is a shallow, prompt-safe schema description.
type SchemaSummary struct {
	Type        string            `json:"type,omitempty"`
	Format      string            `json:"format,omitempty"`
	Ref         string            `json:"ref,omitempty"`
	Description string            `json:"description,omitempty"`
	Required    []string          `json:"required,omitempty"`
	Properties  []PropertySummary `json:"properties,omitempty"`
}

// PropertySummary is a prompt-safe summary of one object property.
type PropertySummary struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Ref         string `json:"ref,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// RequestFieldSummary is a recursive prompt-safe request body field summary. It
// intentionally omits defaults, examples, and secret-like field names.
type RequestFieldSummary struct {
	Path        string `json:"path"`
	Required    bool   `json:"required,omitempty"`
	Type        string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Ref         string `json:"ref,omitempty"`
	Description string `json:"description,omitempty"`
}

// SecuritySummary describes an operation security requirement by symbolic name.
type SecuritySummary struct {
	Name             string             `json:"name"`
	Type             string             `json:"type,omitempty"`
	Scheme           string             `json:"scheme,omitempty"`
	In               string             `json:"in,omitempty"`
	ParameterName    string             `json:"parameter_name,omitempty"`
	Flows            []string           `json:"flows,omitempty"`
	OAuthFlows       []OAuthFlowSummary `json:"oauth_flows,omitempty"`
	AuthorizationURL string             `json:"authorization_url,omitempty"`
	TokenURL         string             `json:"token_url,omitempty"`
	RefreshURL       string             `json:"refresh_url,omitempty"`
	Scopes           []string           `json:"scopes,omitempty"`
	Description      string             `json:"description,omitempty"`
	Extensions       map[string]string  `json:"extensions,omitempty"`
}

// OAuthFlowSummary describes one OpenAPI OAuth2 flow without credential values.
type OAuthFlowSummary struct {
	Name             string   `json:"name"`
	AuthorizationURL string   `json:"authorization_url,omitempty"`
	TokenURL         string   `json:"token_url,omitempty"`
	RefreshURL       string   `json:"refresh_url,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
}

// BuildOperationInventory extracts prompt-safe operation summaries from local
// OpenAPI or Swagger documents. It never fetches remote references or executes
// discovered operations.
func BuildOperationInventory(ctx context.Context, opts InventoryOptions) (OperationInventory, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(opts.Documents) == 0 {
		return OperationInventory{}, fmt.Errorf("at least one OpenAPI document is required")
	}
	var inventory OperationInventory
	for i, doc := range opts.Documents {
		if err := ctx.Err(); err != nil {
			return inventory, err
		}
		content, err := inventoryDocumentContent(doc)
		if err != nil {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "document.read",
				Message:  err.Error(),
				Path:     doc.Path,
			})
			continue
		}
		parsed, err := parseInventoryDocument(content)
		if err != nil {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity:    "error",
				Code:        "document.parse",
				Message:     err.Error(),
				Path:        firstNonEmpty(doc.Path, doc.URL, doc.Name),
				Remediation: "Provide a JSON or YAML OpenAPI/Swagger document.",
			})
			continue
		}
		addDocumentInventory(&inventory, doc, i, parsed, opts.Query)
	}
	sort.SliceStable(inventory.Operations, func(i, j int) bool {
		left, right := inventory.Operations[i], inventory.Operations[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.DocumentPath != right.DocumentPath {
			return left.DocumentPath < right.DocumentPath
		}
		if left.DocumentName != right.DocumentName {
			return left.DocumentName < right.DocumentName
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		return left.Method < right.Method
	})
	if opts.Limit > 0 && len(inventory.Operations) > opts.Limit {
		inventory.Operations = inventory.Operations[:opts.Limit]
	}
	return inventory, nil
}

func inventoryDocumentContent(doc InventoryDocument) ([]byte, error) {
	if len(doc.Content) > 0 {
		return doc.Content, nil
	}
	if strings.TrimSpace(doc.Path) == "" {
		return nil, fmt.Errorf("document content or path is required")
	}
	return os.ReadFile(doc.Path)
}

func parseInventoryDocument(content []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("document is empty")
	}
	var value any
	if trimmed[0] == '{' {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
	} else if err := yaml.Unmarshal(trimmed, &value); err != nil {
		return nil, err
	}
	root, ok := normalizeInventoryValue(value).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("document root must be an object")
	}
	if stringValue(root["openapi"]) == "" && stringValue(root["swagger"]) == "" {
		return nil, fmt.Errorf("document does not declare openapi or swagger")
	}
	return root, nil
}

func normalizeInventoryValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = normalizeInventoryValue(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[fmt.Sprint(key)] = normalizeInventoryValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = normalizeInventoryValue(child)
		}
		return out
	default:
		return value
	}
}

func addDocumentInventory(inventory *OperationInventory, doc InventoryDocument, index int, root map[string]any, query string) {
	info := mapValue(root["info"])
	name := firstNonEmpty(doc.Name, stringValue(info["title"]), filepath.Base(doc.Path), doc.URL, fmt.Sprintf("document-%d", index+1))
	summary := DocumentSummary{
		Name:        name,
		Path:        doc.Path,
		URL:         doc.URL,
		Title:       stringValue(info["title"]),
		Description: stringValue(info["description"]),
		OpenAPI:     stringValue(root["openapi"]),
		Swagger:     stringValue(root["swagger"]),
	}
	securitySchemes := securitySchemes(root)
	defaultSecurity := securityRequirements(root["security"], securitySchemes)
	paths := mapValue(root["paths"])
	pathKeys := sortedMapKeys(paths)
	for _, path := range pathKeys {
		pathItem := mapValue(paths[path])
		pathParameterValues := sliceValue(pathItem["parameters"])
		for _, method := range operationMethods(pathItem) {
			operation := mapValue(pathItem[method])
			op := operationSummary(doc, name, path, method, operation)
			op.Parameters = append(op.Parameters, parameterSummaries(pathParameterValues, &op)...)
			op.Parameters = append(op.Parameters, parameterSummaries(sliceValue(operation["parameters"]), &op)...)
			op.RequestBody = requestBodySummary(operation, &op)
			if value, ok := operation["security"]; ok {
				op.Security = securityRequirements(value, securitySchemes)
			} else {
				op.Security = defaultSecurity
			}
			op.Score = ScoreText(query, operationSearchText(op))
			if op.OperationID == "" {
				issue := ReadinessIssue{
					Severity:    "warning",
					Code:        "operation.missing_id",
					Message:     "operation is missing operationId",
					Path:        op.Provenance,
					Remediation: "Add operationId to the OpenAPI document or select this operation by inventory id.",
				}
				op.ReadinessIssues = append(op.ReadinessIssues, issue)
			}
			inventory.ReadinessIssues = append(inventory.ReadinessIssues, op.ReadinessIssues...)
			inventory.Operations = append(inventory.Operations, op)
			summary.OperationCount++
		}
	}
	inventory.Documents = append(inventory.Documents, summary)
}

func operationSummary(doc InventoryDocument, documentName, path, method string, operation map[string]any) OperationSummary {
	operationID := stringValue(operation["operationId"])
	id := operationID
	if id == "" {
		id = sanitizeName(firstNonEmpty(documentName, doc.Path, doc.URL)) + "_" + method + "_" + sanitizeName(path)
	}
	provenance := firstNonEmpty(doc.Path, doc.URL, documentName) + "#" + method + " " + path
	return OperationSummary{
		ID:           id,
		DocumentName: documentName,
		DocumentPath: doc.Path,
		DocumentURL:  doc.URL,
		OperationID:  operationID,
		Method:       strings.ToUpper(method),
		Path:         path,
		Summary:      stringValue(operation["summary"]),
		Description:  stringValue(operation["description"]),
		Tags:         stringSlice(operation["tags"]),
		Extensions:   operationExtensions(operation),
		Provenance:   provenance,
	}
}

func operationExtensions(operation map[string]any) map[string]string {
	allow := map[string]struct{}{
		"x-aws-operation-name": {},
	}
	out := map[string]string{}
	for key := range allow {
		value := strings.TrimSpace(stringValue(operation[key]))
		if value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func operationMethods(pathItem map[string]any) []string {
	methods := make([]string, 0, 8)
	for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
		if _, ok := pathItem[method].(map[string]any); ok {
			methods = append(methods, method)
		}
	}
	return methods
}

func parameterSummaries(parameters []any, op *OperationSummary) []ParameterSummary {
	out := make([]ParameterSummary, 0, len(parameters))
	for _, value := range parameters {
		parameter := mapValue(value)
		if len(parameter) == 0 {
			continue
		}
		schema := mapValue(parameter["schema"])
		summary := ParameterSummary{
			Name:        stringValue(parameter["name"]),
			In:          stringValue(parameter["in"]),
			Description: stringValue(parameter["description"]),
			Required:    boolValue(parameter["required"]),
			Type:        firstNonEmpty(stringValue(schema["type"]), stringValue(parameter["type"])),
			Format:      firstNonEmpty(stringValue(schema["format"]), stringValue(parameter["format"])),
			Ref:         firstNonEmpty(stringValue(parameter["$ref"]), stringValue(schema["$ref"])),
		}
		if summary.Ref != "" && op != nil {
			addOperationIssue(op, "schema.ref_unresolved", "parameter references a schema that was not resolved", summary.Ref)
		}
		out = append(out, summary)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].In != out[j].In {
			return out[i].In < out[j].In
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func requestBodySummary(operation map[string]any, op *OperationSummary) *RequestBodySummary {
	if body := mapValue(operation["requestBody"]); len(body) > 0 {
		summary := &RequestBodySummary{
			Description: stringValue(body["description"]),
			Required:    boolValue(body["required"]),
			Ref:         stringValue(body["$ref"]),
		}
		if summary.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "requestBody references a schema that was not resolved", summary.Ref)
		}
		content := mapValue(body["content"])
		summary.ContentTypes = sortedMapKeys(content)
		if len(summary.ContentTypes) > 0 {
			media := mapValue(content[summary.ContentTypes[0]])
			rawSchema := mapValue(media["schema"])
			schema := schemaSummary(rawSchema)
			summary.Schema = &schema
			summary.Fields = requestFieldSummaries(rawSchema, "", summary.Required, 0)
			if len(summary.Fields) == 0 && len(rawSchema) > 0 && !looksLikeCredentialName("body") {
				summary.Fields = []RequestFieldSummary{requestFieldSummary("body", summary.Required, rawSchema)}
			}
			summary.RequiredFieldPaths = requiredRequestFieldPaths(summary.Fields)
			if schema.Ref != "" {
				addOperationIssue(op, "schema.ref_unresolved", "request body schema reference was not resolved", schema.Ref)
			}
		}
		return summary
	}
	for _, parameterValue := range sliceValue(operation["parameters"]) {
		parameter := mapValue(parameterValue)
		if stringValue(parameter["in"]) != "body" {
			continue
		}
		rawSchema := mapValue(parameter["schema"])
		schema := schemaSummary(rawSchema)
		if schema.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "body parameter schema reference was not resolved", schema.Ref)
		}
		fields := requestFieldSummaries(rawSchema, "", boolValue(parameter["required"]), 0)
		if len(fields) == 0 && len(rawSchema) > 0 && !looksLikeCredentialName("body") {
			fields = []RequestFieldSummary{requestFieldSummary("body", boolValue(parameter["required"]), rawSchema)}
		}
		return &RequestBodySummary{
			Description:        stringValue(parameter["description"]),
			Required:           boolValue(parameter["required"]),
			Schema:             &schema,
			Ref:                stringValue(parameter["$ref"]),
			Fields:             fields,
			RequiredFieldPaths: requiredRequestFieldPaths(fields),
		}
	}
	return nil
}

func schemaSummary(schema map[string]any) SchemaSummary {
	if len(schema) == 0 {
		return SchemaSummary{}
	}
	required := stringSlice(schema["required"])
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	summary := SchemaSummary{
		Type:        schemaType(schema["type"]),
		Format:      stringValue(schema["format"]),
		Ref:         stringValue(schema["$ref"]),
		Description: stringValue(schema["description"]),
		Required:    required,
	}
	properties := mapValue(schema["properties"])
	for _, name := range sortedMapKeys(properties) {
		property := schemaSummary(mapValue(properties[name]))
		summary.Properties = append(summary.Properties, PropertySummary{
			Name:        name,
			Type:        property.Type,
			Format:      property.Format,
			Ref:         property.Ref,
			Description: property.Description,
			Required:    requiredSet[name],
		})
	}
	return summary
}

func securitySchemes(root map[string]any) map[string]SecuritySummary {
	out := make(map[string]SecuritySummary)
	components := mapValue(root["components"])
	for name, value := range mapValue(components["securitySchemes"]) {
		scheme := mapValue(value)
		out[name] = SecuritySummary{
			Name:             name,
			Type:             stringValue(scheme["type"]),
			Scheme:           stringValue(scheme["scheme"]),
			In:               stringValue(scheme["in"]),
			ParameterName:    stringValue(scheme["name"]),
			Flows:            securitySchemeFlows(scheme),
			OAuthFlows:       securitySchemeOAuthFlows(scheme),
			AuthorizationURL: securitySchemeOAuthURL(scheme, "authorizationUrl"),
			TokenURL:         securitySchemeOAuthURL(scheme, "tokenUrl"),
			RefreshURL:       securitySchemeOAuthURL(scheme, "refreshUrl"),
			Description:      stringValue(scheme["description"]),
			Extensions:       securitySchemeExtensions(scheme),
		}
	}
	for name, value := range mapValue(root["securityDefinitions"]) {
		scheme := mapValue(value)
		out[name] = SecuritySummary{
			Name:             name,
			Type:             stringValue(scheme["type"]),
			Scheme:           stringValue(scheme["scheme"]),
			In:               stringValue(scheme["in"]),
			ParameterName:    stringValue(scheme["name"]),
			Flows:            securitySchemeFlows(scheme),
			OAuthFlows:       securitySchemeOAuthFlows(scheme),
			AuthorizationURL: securitySchemeOAuthURL(scheme, "authorizationUrl"),
			TokenURL:         securitySchemeOAuthURL(scheme, "tokenUrl"),
			RefreshURL:       securitySchemeOAuthURL(scheme, "refreshUrl"),
			Description:      stringValue(scheme["description"]),
			Extensions:       securitySchemeExtensions(scheme),
		}
	}
	return out
}

func securitySchemeFlows(scheme map[string]any) []string {
	flows := sortedMapKeys(mapValue(scheme["flows"]))
	if len(flows) > 0 {
		return flows
	}
	flow := strings.TrimSpace(stringValue(scheme["flow"]))
	if flow == "" {
		return nil
	}
	return []string{flow}
}

func securitySchemeOAuthFlows(scheme map[string]any) []OAuthFlowSummary {
	flows := mapValue(scheme["flows"])
	if len(flows) > 0 {
		out := make([]OAuthFlowSummary, 0, len(flows))
		for _, name := range sortedMapKeys(flows) {
			flow := mapValue(flows[name])
			out = append(out, OAuthFlowSummary{
				Name:             name,
				AuthorizationURL: strings.TrimSpace(stringValue(flow["authorizationUrl"])),
				TokenURL:         strings.TrimSpace(stringValue(flow["tokenUrl"])),
				RefreshURL:       strings.TrimSpace(stringValue(flow["refreshUrl"])),
				Scopes:           sortedMapKeys(mapValue(flow["scopes"])),
			})
		}
		return out
	}
	name := strings.TrimSpace(stringValue(scheme["flow"]))
	if name == "" {
		return nil
	}
	return []OAuthFlowSummary{{
		Name:             name,
		AuthorizationURL: strings.TrimSpace(stringValue(scheme["authorizationUrl"])),
		TokenURL:         strings.TrimSpace(stringValue(scheme["tokenUrl"])),
		RefreshURL:       strings.TrimSpace(stringValue(scheme["refreshUrl"])),
		Scopes:           sortedMapKeys(mapValue(scheme["scopes"])),
	}}
}

func securitySchemeOAuthURL(scheme map[string]any, key string) string {
	if value := strings.TrimSpace(stringValue(scheme[key])); value != "" {
		return value
	}
	var values []string
	for _, flowName := range sortedMapKeys(mapValue(scheme["flows"])) {
		value := strings.TrimSpace(stringValue(mapValue(mapValue(scheme["flows"])[flowName])[key]))
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return ""
	}
	sort.Strings(values)
	return values[0]
}

func securitySchemeExtensions(scheme map[string]any) map[string]string {
	allow := map[string]struct{}{
		"x-amazon-apigateway-authtype": {},
		"x-aws-auth-type":              {},
		"x-aws-signature-version":      {},
	}
	out := map[string]string{}
	for key := range allow {
		value := strings.TrimSpace(stringValue(scheme[key]))
		if value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func securityRequirements(value any, schemes map[string]SecuritySummary) []SecuritySummary {
	var out []SecuritySummary
	seen := make(map[string]bool)
	for _, requirementValue := range sliceValue(value) {
		requirement := mapValue(requirementValue)
		for _, name := range sortedMapKeys(requirement) {
			if seen[name] {
				continue
			}
			summary := schemes[name]
			if summary.Name == "" {
				summary.Name = name
			}
			summary.Scopes = stringSlice(requirement[name])
			out = append(out, summary)
			seen[name] = true
		}
	}
	return out
}

func operationSearchText(op OperationSummary) string {
	var parts []string
	parts = append(parts, op.OperationID, op.Summary, op.Description, op.Path, op.Method)
	parts = append(parts, op.Tags...)
	for _, parameter := range op.Parameters {
		parts = append(parts, parameter.Name, parameter.In, parameter.Description, parameter.Type, parameter.Format)
	}
	if op.RequestBody != nil && op.RequestBody.Schema != nil {
		parts = append(parts, op.RequestBody.Schema.Type, op.RequestBody.Schema.Ref)
		for _, property := range op.RequestBody.Schema.Properties {
			parts = append(parts, property.Name, property.Type, property.Description)
		}
	}
	return strings.Join(nonEmptyStrings(parts...), " ")
}

func addOperationIssue(op *OperationSummary, code, message, ref string) {
	if op == nil {
		return
	}
	issue := ReadinessIssue{
		Severity:    "warning",
		Code:        code,
		Message:     message,
		OperationID: op.OperationID,
		Path:        firstNonEmpty(ref, op.Provenance),
		Remediation: "Resolve the reference in the caller before rendering or execution.",
	}
	op.ReadinessIssues = append(op.ReadinessIssues, issue)
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func sliceValue(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func boolValue(value any) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func stringSlice(value any) []string {
	values := sliceValue(value)
	if len(values) == 0 {
		if text := stringValue(value); text != "" {
			return []string{text}
		}
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text := stringValue(value); text != "" {
			out = append(out, text)
		}
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func schemaType(value any) string {
	if text := stringValue(value); text != "" {
		return text
	}
	values := stringSlice(value)
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "|")
}

const (
	maxRequestFieldDepth = 6
	maxRequestFields     = 60
)

func requestFieldSummaries(schema map[string]any, path string, required bool, depth int) []RequestFieldSummary {
	var out []RequestFieldSummary
	collectRequestFields(schema, path, required, depth, &out)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	if len(out) > maxRequestFields {
		out = out[:maxRequestFields]
	}
	return out
}

func collectRequestFields(schema map[string]any, path string, required bool, depth int, out *[]RequestFieldSummary) {
	if len(*out) >= maxRequestFields || depth > maxRequestFieldDepth {
		return
	}
	if len(schema) == 0 {
		if path != "" && !looksLikeCredentialName(path) {
			*out = append(*out, RequestFieldSummary{Path: path, Required: required})
		}
		return
	}
	if path != "" && !looksLikeCredentialName(path) {
		*out = append(*out, requestFieldSummary(path, required, schema))
		if len(*out) >= maxRequestFields || depth == maxRequestFieldDepth {
			return
		}
	}
	properties := mapValue(schema["properties"])
	if len(properties) > 0 {
		requiredSet := make(map[string]bool)
		for _, name := range stringSlice(schema["required"]) {
			requiredSet[name] = true
		}
		for _, name := range sortedMapKeys(properties) {
			childPath := name
			if path != "" {
				childPath = path + "." + name
			}
			collectRequestFields(mapValue(properties[name]), childPath, required && requiredSet[name], depth+1, out)
			if len(*out) >= maxRequestFields {
				return
			}
		}
		return
	}
	if items := mapValue(schema["items"]); len(items) > 0 {
		itemPath := "body[]"
		if path != "" {
			itemPath = path + "[]"
		}
		collectRequestFields(items, itemPath, required, depth+1, out)
	}
}

func requestFieldSummary(path string, required bool, schema map[string]any) RequestFieldSummary {
	return RequestFieldSummary{
		Path:        path,
		Required:    required,
		Type:        schemaType(schema["type"]),
		Format:      stringValue(schema["format"]),
		Ref:         stringValue(schema["$ref"]),
		Description: stringValue(schema["description"]),
	}
}

func requiredRequestFieldPaths(fields []RequestFieldSummary) []string {
	var out []string
	for _, field := range fields {
		if field.Required {
			out = append(out, field.Path)
		}
	}
	sort.Strings(out)
	return out
}
