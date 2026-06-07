package apitools

// InventoryDocument is one local OpenAPI or Swagger document to inspect.
type InventoryDocument struct {
	Name         string `json:"name,omitempty"`
	Path         string `json:"path,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	URL          string `json:"url,omitempty"`
	Content      []byte `json:"-"`
}

// InventoryOptions configures operation inventory extraction.
type InventoryOptions struct {
	Documents []InventoryDocument `json:"documents,omitempty"`
	Query     string              `json:"query,omitempty"`
	Limit     int                 `json:"limit,omitempty"`
	MaxBytes  int64               `json:"max_bytes,omitempty"`
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
	RelativePath   string `json:"relative_path,omitempty"`
	URL            string `json:"url,omitempty"`
	Title          string `json:"title,omitempty"`
	Description    string `json:"description,omitempty"`
	OpenAPI        string `json:"openapi,omitempty"`
	Swagger        string `json:"swagger,omitempty"`
	OperationCount int    `json:"operation_count,omitempty"`
}

// OperationSummary describes one prompt-safe OpenAPI operation.
type OperationSummary struct {
	ID                   string               `json:"id"`
	DocumentName         string               `json:"document_name,omitempty"`
	DocumentPath         string               `json:"document_path,omitempty"`
	DocumentRelativePath string               `json:"document_relative_path,omitempty"`
	DocumentURL          string               `json:"document_url,omitempty"`
	OperationID          string               `json:"operation_id,omitempty"`
	Method               string               `json:"method"`
	Path                 string               `json:"path"`
	Summary              string               `json:"summary,omitempty"`
	Description          string               `json:"description,omitempty"`
	Tags                 []string             `json:"tags,omitempty"`
	Extensions           map[string]string    `json:"extensions,omitempty"`
	Parameters           []ParameterSummary   `json:"parameters,omitempty"`
	RequestBody          *RequestBodySummary  `json:"request_body,omitempty"`
	ResponseBody         *ResponseBodySummary `json:"response_body,omitempty"`
	Security             []SecuritySummary    `json:"security,omitempty"`
	Score                int                  `json:"score,omitempty"`
	Provenance           string               `json:"provenance,omitempty"`
	ReadinessIssues      []ReadinessIssue     `json:"readiness_issues,omitempty"`
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

// ResponseBodySummary describes the first successful response schema without
// examples or defaults.
type ResponseBodySummary struct {
	StatusCode   string                `json:"status_code,omitempty"`
	Description  string                `json:"description,omitempty"`
	ContentTypes []string              `json:"content_types,omitempty"`
	Schema       *SchemaSummary        `json:"schema,omitempty"`
	Ref          string                `json:"ref,omitempty"`
	Fields       []RequestFieldSummary `json:"fields,omitempty"`
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
