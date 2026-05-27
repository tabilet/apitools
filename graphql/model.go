package graphql

// ArtifactKind classifies the local GraphQL source artifact shape.
type ArtifactKind string

const (
	ArtifactKindIntrospectionJSON ArtifactKind = "graphql-introspection-json"
	ArtifactKindSDL               ArtifactKind = "graphql-sdl"
	ArtifactKindOperationDocument ArtifactKind = "graphql-operation-document"
)

// ArtifactDetection records local-only GraphQL source detection evidence.
type ArtifactDetection struct {
	Kind   ArtifactKind
	Format string
	Path   string
	Reason string
}

// Model is the metadata preserved from a GraphQL source artifact.
type Model struct {
	SourceKind       ArtifactKind
	Schema           Schema
	Types            []*Type
	Operations       []*Operation
	RawIntrospection map[string]any
}

// Schema records the operation root type names.
type Schema struct {
	QueryType        string
	MutationType     string
	SubscriptionType string
	Description      string
}

// Type records GraphQL type metadata.
type Type struct {
	Name          string
	Kind          string
	Description   string
	Fields        []*Field
	InputFields   []*Argument
	EnumValues    []*EnumValue
	Interfaces    []string
	PossibleTypes []string
	UnionMembers  []string
	Selector      string
}

// Field records an output field on an object or interface type.
type Field struct {
	Name              string
	Description       string
	Type              TypeRef
	Args              []*Argument
	Deprecated        bool
	DeprecationReason string
	Selector          string
}

// Argument records field argument, input-object field, or variable metadata.
type Argument struct {
	Name         string
	Description  string
	Type         TypeRef
	DefaultValue string
	Required     bool
}

// EnumValue records GraphQL enum value metadata.
type EnumValue struct {
	Name              string
	Description       string
	Deprecated        bool
	DeprecationReason string
}

// TypeRef is a compact GraphQL type reference.
type TypeRef struct {
	Name     string
	Kind     string
	Display  string
	Required bool
	List     bool
	OfType   *TypeRef
}

// Operation records one GraphQL operation document entry or schema root field.
type Operation struct {
	Name           string
	Kind           string
	ID             string
	Selector       string
	SourceRef      string
	Description    string
	Summary        string
	Variables      []*Variable
	SelectionNames []string
	FieldName      string
	FieldType      TypeRef
	RootType       string
}

// Variable records GraphQL operation variable metadata.
type Variable struct {
	Name         string
	Type         TypeRef
	DefaultValue string
	Required     bool
}

// OperationSummary is a stable, prompt-safe summary of one GraphQL source
// operation or schema root field.
type OperationSummary struct {
	ID                 string
	Name               string
	Kind               string
	Selector           string
	SourceOperationID  string
	SourceOperationRef string
	Summary            string
	Description        string
	Variables          []VariableSummary
	VariableNames      []string
	SelectionNames     []string
	FieldName          string
	FieldType          string
	RootType           string
}

// VariableSummary is a stable summary of one GraphQL operation variable.
type VariableSummary struct {
	Name         string
	Type         string
	Required     bool
	DefaultValue string
}

// SelectorTarget is the result of resolving a local GraphQL selector alias.
type SelectorTarget struct {
	Kind      string
	Operation *Operation
	Summary   OperationSummary
}

const (
	// SelectorKindOperation identifies operation selector targets.
	SelectorKindOperation = "operation"
)
