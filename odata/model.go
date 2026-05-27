package odata

// ArtifactKind classifies the local OData source artifact shape.
type ArtifactKind string

const (
	ArtifactKindCSDLXML  ArtifactKind = "odata-csdl-xml"
	ArtifactKindCSDLJSON ArtifactKind = "odata-csdl-json"
)

// ArtifactDetection records local-only OData source detection evidence.
type ArtifactDetection struct {
	Kind   ArtifactKind
	Format string
	Path   string
	Reason string
}

// Model is the metadata preserved from an OData source artifact.
type Model struct {
	SourceKind ArtifactKind
	Version    string
	Schemas    []*Schema
}

// Schema records one OData schema namespace.
type Schema struct {
	Namespace       string
	Alias           string
	EntityTypes     []*EntityType
	ComplexTypes    []*EntityType
	Actions         []*Operation
	Functions       []*Operation
	EntityContainer *EntityContainer
}

// EntityType records an OData entity or complex type.
type EntityType struct {
	Name                 string
	FullName             string
	BaseType             string
	OpenType             bool
	Properties           []*Property
	NavigationProperties []*NavigationProperty
	Selector             string
}

// Property records a structural property.
type Property struct {
	Name       string
	Type       string
	Nullable   bool
	Collection bool
}

// NavigationProperty records a navigation property.
type NavigationProperty struct {
	Name       string
	Type       string
	Nullable   bool
	Collection bool
	Partner    string
	Contains   bool
	Selector   string
}

// EntityContainer records entity sets, singletons, and operation imports.
type EntityContainer struct {
	Name            string
	FullName        string
	EntitySets      []*EntitySet
	Singletons      []*Singleton
	ActionImports   []*OperationImport
	FunctionImports []*OperationImport
	Selector        string
}

// EntitySet records one OData entity set.
type EntitySet struct {
	Name       string
	EntityType string
	Selector   string
}

// Singleton records one OData singleton.
type Singleton struct {
	Name       string
	EntityType string
	Selector   string
}

// Operation records one OData action or function.
type Operation struct {
	Name               string
	FullName           string
	Kind               string
	Bound              bool
	EntitySetPath      string
	Composable         bool
	Parameters         []*Parameter
	ReturnType         string
	SourceOperationID  string
	SourceOperationRef string
	Selector           string
}

// OperationImport records an action or function import in an entity container.
type OperationImport struct {
	Name               string
	Kind               string
	Operation          string
	EntitySet          string
	SourceOperationID  string
	SourceOperationRef string
	Selector           string
}

// Parameter records operation parameter metadata.
type Parameter struct {
	Name       string
	Type       string
	Nullable   bool
	Collection bool
}

// OperationSummary is a stable, prompt-safe summary of one OData operation or
// addressable resource.
type OperationSummary struct {
	ID                 string
	Name               string
	Kind               string
	Selector           string
	SourceOperationID  string
	SourceOperationRef string
	EntityType         string
	ReturnType         string
	Parameters         []ParameterSummary
	QueryOptions       []string
	Bound              bool
	Collection         bool
}

// ParameterSummary is a stable summary of one OData operation parameter.
type ParameterSummary struct {
	Name       string
	Type       string
	Nullable   bool
	Collection bool
}

// SelectorTarget is the result of resolving a local OData selector alias.
type SelectorTarget struct {
	Kind    string
	Summary OperationSummary
}

const (
	SelectorKindOperation = "operation"
)
