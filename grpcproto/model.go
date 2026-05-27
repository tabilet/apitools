package grpcproto

// ArtifactKind classifies the local gRPC/protobuf source artifact shape.
type ArtifactKind string

const (
	ArtifactKindProtoFile           ArtifactKind = "proto-file"
	ArtifactKindDescriptorSet       ArtifactKind = "protobuf-descriptor-set"
	ArtifactKindDescriptorJSON      ArtifactKind = "protobuf-descriptor-json"
	ArtifactKindGeneratedDescriptor ArtifactKind = "generated-descriptor-artifact"
)

// ArtifactDetection records local-only gRPC/protobuf source detection evidence.
type ArtifactDetection struct {
	Kind   ArtifactKind
	Format string
	Path   string
	Reason string
}

// Model is the metadata preserved from protobuf source artifacts.
type Model struct {
	SourceKind ArtifactKind
	Files      []*File
}

// File records one protobuf file descriptor or .proto source file.
type File struct {
	Name     string
	Package  string
	Syntax   string
	Imports  []string
	Services []*Service
	Messages []*Message
	Enums    []*Enum
	Selector string
}

// Service records one gRPC service.
type Service struct {
	Name     string
	FullName string
	Package  string
	Methods  []*Method
	Selector string
}

// Method records one gRPC service method.
type Method struct {
	Name               string
	FullName           string
	ServiceName        string
	Package            string
	RequestType        string
	ResponseType       string
	ClientStreaming    bool
	ServerStreaming    bool
	Selector           string
	SourceOperationID  string
	SourceOperationRef string
}

// Message records one protobuf message.
type Message struct {
	Name     string
	FullName string
	Fields   []*Field
	Messages []*Message
	Enums    []*Enum
	Selector string
}

// Field records one protobuf message field.
type Field struct {
	Name     string
	Number   int
	Label    string
	Type     string
	TypeName string
	JSONName string
	Repeated bool
	Required bool
	Selector string
}

// Enum records one protobuf enum.
type Enum struct {
	Name     string
	FullName string
	Values   []*EnumValue
	Selector string
}

// EnumValue records one protobuf enum value.
type EnumValue struct {
	Name   string
	Number int
}

// MethodSummary is a stable, prompt-safe summary of one gRPC method.
type MethodSummary struct {
	ID                 string
	Name               string
	ServiceName        string
	Package            string
	Selector           string
	SourceOperationID  string
	SourceOperationRef string
	FullMethod         string
	RequestType        string
	ResponseType       string
	ClientStreaming    bool
	ServerStreaming    bool
	RequestFields      []FieldSummary
}

// FieldSummary is a stable summary of one protobuf request field.
type FieldSummary struct {
	Name     string
	Number   int
	Label    string
	Type     string
	TypeName string
	JSONName string
	Repeated bool
	Required bool
}

// SelectorTarget is the result of resolving a local gRPC/protobuf selector.
type SelectorTarget struct {
	Kind    string
	Method  *Method
	Summary MethodSummary
}

const (
	// SelectorKindMethod identifies gRPC method selector targets.
	SelectorKindMethod = "method"
)
