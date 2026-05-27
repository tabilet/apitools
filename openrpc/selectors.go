package openrpc

import (
	"fmt"
	"sort"
	"strings"
)

// MethodSelector returns the local selector for an OpenRPC method name.
func MethodSelector(name string) string {
	return "#/methods/" + escapeJSONPointer(strings.TrimSpace(name))
}

// MethodByName returns a method by JSON-RPC method name.
func (m *Model) MethodByName(name string) (*Method, bool) {
	if m == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	for _, method := range m.Methods {
		if method != nil && method.Name == name {
			return method, true
		}
	}
	return nil, false
}

// MethodSummaries returns deterministic summaries for all OpenRPC methods.
func (m *Model) MethodSummaries() []MethodSummary {
	if m == nil {
		return nil
	}
	out := make([]MethodSummary, 0, len(m.Methods))
	for _, method := range m.Methods {
		if method == nil || strings.TrimSpace(method.Name) == "" {
			continue
		}
		out = append(out, m.methodSummary(method))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// SelectorAliases returns all supported selector aliases for every method.
func (m *Model) SelectorAliases() map[string]string {
	if m == nil {
		return nil
	}
	out := map[string]string{}
	for _, method := range m.Methods {
		if method == nil || strings.TrimSpace(method.Name) == "" {
			continue
		}
		selector := MethodSelector(method.Name)
		for _, alias := range methodSelectorAliases(method.Name) {
			out[alias] = selector
		}
	}
	return out
}

// ResolveSelector resolves supported method selector aliases, including the
// canonical method name, method:<name>, and #/methods/<json-pointer-name>.
func (m *Model) ResolveSelector(selector string) (SelectorTarget, error) {
	if m == nil {
		return SelectorTarget{}, fmt.Errorf("openrpc: nil model")
	}
	selector = strings.TrimSpace(selector)
	for _, method := range m.Methods {
		if method == nil {
			continue
		}
		for _, alias := range methodSelectorAliases(method.Name) {
			if selector == alias {
				return SelectorTarget{
					Kind:    SelectorKindMethod,
					Method:  method,
					Summary: m.methodSummary(method),
				}, nil
			}
		}
	}
	return SelectorTarget{}, fmt.Errorf("openrpc: unknown selector %q", selector)
}

func (m *Model) methodSummary(method *Method) MethodSummary {
	summary := MethodSummary{
		ID:             method.Name,
		Name:           method.Name,
		Selector:       MethodSelector(method.Name),
		Summary:        method.Summary,
		Description:    method.Description,
		ParamStructure: method.ParamStructure,
		Tags:           append([]string(nil), method.Tags...),
	}
	for _, param := range method.Params {
		if param == nil {
			continue
		}
		descriptor := m.resolveContentDescriptor(param)
		name := strings.TrimSpace(descriptor.Name)
		if name == "" {
			name = refName(param.Ref)
		}
		if name != "" {
			summary.ParamNames = append(summary.ParamNames, name)
		}
		if param.Ref != "" {
			summary.ParamRefs = append(summary.ParamRefs, param.Ref)
		}
		summary.Params = append(summary.Params, ParamSummary{
			Name:     name,
			Ref:      param.Ref,
			Required: descriptor.Required,
		})
	}
	if method.Result != nil {
		result := m.resolveContentDescriptor(method.Result)
		summary.ResultName = result.Name
		summary.ResultRef = method.Result.Ref
		if summary.ResultName == "" {
			summary.ResultName = refName(method.Result.Ref)
		}
	}
	for _, errDef := range method.Errors {
		if errDef == nil {
			continue
		}
		resolved := m.resolveError(errDef)
		if resolved.Code != "" {
			summary.ErrorCodes = append(summary.ErrorCodes, resolved.Code)
		}
		if errDef.Ref != "" {
			summary.ErrorRefs = append(summary.ErrorRefs, errDef.Ref)
		}
	}
	sort.Strings(summary.ErrorCodes)
	sort.Strings(summary.ErrorRefs)
	return summary
}

func (m *Model) resolveContentDescriptor(descriptor *ContentDescriptor) *ContentDescriptor {
	if descriptor == nil || descriptor.Ref == "" || m == nil {
		return descriptor
	}
	if name, ok := localComponentRefName(descriptor.Ref, "#/components/contentDescriptors/"); ok {
		if resolved := m.Components.ContentDescriptors[name]; resolved != nil {
			return resolved
		}
	}
	return descriptor
}

func (m *Model) resolveError(errDef *Error) *Error {
	if errDef == nil || errDef.Ref == "" || m == nil {
		return errDef
	}
	if name, ok := localComponentRefName(errDef.Ref, "#/components/errors/"); ok {
		if resolved := m.Components.Errors[name]; resolved != nil {
			return resolved
		}
	}
	return errDef
}

func methodSelectorAliases(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return []string{name, "method:" + name, MethodSelector(name)}
}

func localComponentRefName(ref, prefix string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name, err := unescapeJSONPointer(strings.TrimPrefix(ref, prefix))
	if err != nil || name == "" {
		return "", false
	}
	return name, true
}

func refName(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 && i < len(ref)-1 {
		name, err := unescapeJSONPointer(ref[i+1:])
		if err == nil {
			return name
		}
		return ref[i+1:]
	}
	return ref
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}

func unescapeJSONPointer(value string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '~' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) {
			return "", fmt.Errorf("invalid JSON pointer escape")
		}
		switch value[i+1] {
		case '0':
			b.WriteByte('~')
		case '1':
			b.WriteByte('/')
		default:
			return "", fmt.Errorf("invalid JSON pointer escape")
		}
		i++
	}
	return b.String(), nil
}
