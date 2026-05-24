package helper

import (
	"testing"

	"github.com/OpenUdon/apitools/helper/fnctspec"
	"github.com/OpenUdon/apitools/helper/gmailmsg"
)

type testRegistrar struct {
	values map[string]any
}

func (r *testRegistrar) AddStubMap(name string, value any) {
	if r.values == nil {
		r.values = map[string]any{}
	}
	r.values[name] = value
}

func TestFunctionCatalogIncludesGmailRenderRaw(t *testing.T) {
	spec, ok := LookupFunctionSpec(gmailmsg.FunctionNameRenderRaw)
	if !ok {
		t.Fatalf("missing function spec %q", gmailmsg.FunctionNameRenderRaw)
	}
	if spec.InvocationMode != fnctspec.InvocationRequestBodyObject {
		t.Fatalf("invocation mode = %q", spec.InvocationMode)
	}

	reg := &testRegistrar{}
	RegisterFnctHelpers(reg)
	if reg.values[gmailmsg.FunctionNameRenderRaw] == nil {
		t.Fatalf("missing registered helper %q", gmailmsg.FunctionNameRenderRaw)
	}
}
