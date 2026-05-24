// Package fnctspec describes public metadata for pure function helpers that
// can be registered by trusted UWS runtimes.
package fnctspec

// InvocationMode names how a runtime should pass inputs to a helper.
type InvocationMode string

const (
	// InvocationRequestBodyObject means the operation request body is passed as
	// one object argument to the registered function.
	InvocationRequestBodyObject InvocationMode = "request_body_object"
)

// FieldSpec describes one helper input or output field.
type FieldSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Semantic    string `json:"semantic,omitempty"`
	Description string `json:"description,omitempty"`
}

// FunctionSpec describes a portable pure function helper. It is metadata only:
// runtimes still own registration and execution.
type FunctionSpec struct {
	Name           string         `json:"name"`
	RuntimeType    string         `json:"runtime_type"`
	InvocationMode InvocationMode `json:"invocation_mode"`
	Summary        string         `json:"summary,omitempty"`
	Inputs         []FieldSpec    `json:"inputs,omitempty"`
	Output         FieldSpec      `json:"output"`
	Pure           bool           `json:"pure"`
	SideEffects    string         `json:"side_effects"`
}
