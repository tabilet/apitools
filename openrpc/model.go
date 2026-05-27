package openrpc

// Model is the metadata preserved from an OpenRPC document.
type Model struct {
	OpenRPC    string
	Info       Info
	Servers    []*Server
	Methods    []*Method
	Components Components
	Raw        map[string]any
}

// Info describes the OpenRPC document.
type Info struct {
	Title          string
	Version        string
	Description    string
	TermsOfService string
	Contact        map[string]any
	License        map[string]any
	Raw            map[string]any
}

// Server records a server object without opening a connection to it.
type Server struct {
	Name        string
	URL         string
	Summary     string
	Description string
	Variables   map[string]any
	Raw         map[string]any
}

// Method records one JSON-RPC method contract.
type Method struct {
	Name           string
	Summary        string
	Description    string
	ParamStructure string
	Params         []*ContentDescriptor
	Result         *ContentDescriptor
	Errors         []*Error
	Tags           []string
	ExternalDocs   map[string]any
	Deprecated     bool
	Servers        []*Server
	Raw            map[string]any
}

// ContentDescriptor records OpenRPC content descriptor metadata for params and
// results. Schema is intentionally left as generic structured JSON.
type ContentDescriptor struct {
	Name        string
	Summary     string
	Description string
	Required    bool
	Deprecated  bool
	Ref         string
	Schema      map[string]any
	Raw         map[string]any
}

// Error records OpenRPC error metadata. Code is stored as text so large and
// negative JSON-RPC codes remain stable through decode/encode cycles.
type Error struct {
	Code    string
	Message string
	Data    any
	Ref     string
	Raw     map[string]any
}

// Components preserves OpenRPC reusable metadata maps. Known reusable content
// descriptors and errors are parsed for selector and summary use; all other
// component maps remain raw structured JSON.
type Components struct {
	Schemas            map[string]any
	ContentDescriptors map[string]*ContentDescriptor
	Errors             map[string]*Error
	Examples           map[string]any
	Links              map[string]any
	Tags               map[string]any
	Raw                map[string]any
}

// MethodSummary is a stable, prompt-safe summary of one OpenRPC method.
type MethodSummary struct {
	ID             string
	Name           string
	Selector       string
	Summary        string
	Description    string
	ParamStructure string
	Params         []ParamSummary
	ParamNames     []string
	ParamRefs      []string
	ResultName     string
	ResultRef      string
	ErrorCodes     []string
	ErrorRefs      []string
	Tags           []string
}

// ParamSummary is a stable summary of one OpenRPC method parameter.
type ParamSummary struct {
	Name     string
	Ref      string
	Required bool
}

// SelectorTarget is the result of resolving a local OpenRPC selector alias.
type SelectorTarget struct {
	Kind    string
	Method  *Method
	Summary MethodSummary
}

const (
	// SelectorKindMethod identifies method selector targets.
	SelectorKindMethod = "method"
)
