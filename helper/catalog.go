// Package helper exposes public metadata and registration hooks for pure
// payload-shaping helpers.
package helper

import (
	"strings"

	"github.com/OpenUdon/apitools/helper/fnctspec"
	"github.com/OpenUdon/apitools/helper/gmailmsg"
)

// FnctRegistrar is the minimal runtime hook needed to register pure fnct
// helpers. Runtime packages own when and where registration happens.
type FnctRegistrar interface {
	AddStubMap(string, any)
}

// FunctionSpecs returns the public fnct helper descriptors known to apitools.
func FunctionSpecs() []fnctspec.FunctionSpec {
	return []fnctspec.FunctionSpec{
		gmailmsg.FunctionSpec(),
	}
}

// LookupFunctionSpec returns the descriptor for a helper name.
func LookupFunctionSpec(name string) (fnctspec.FunctionSpec, bool) {
	name = strings.TrimSpace(name)
	for _, spec := range FunctionSpecs() {
		if spec.Name == name {
			return spec, true
		}
	}
	return fnctspec.FunctionSpec{}, false
}

// RegisterFnctHelpers registers all known pure helper implementations with a
// trusted fnct runtime.
func RegisterFnctHelpers(reg FnctRegistrar) {
	if reg == nil {
		return
	}
	reg.AddStubMap(gmailmsg.FunctionNameRenderRaw, gmailmsg.RenderRawAny)
}
