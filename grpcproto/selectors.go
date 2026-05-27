package grpcproto

import (
	"fmt"
	"sort"
	"strings"
)

// ServiceSelector returns the local selector for a protobuf service.
func ServiceSelector(fullName string) string {
	return "#/services/" + escapeJSONPointer(strings.TrimSpace(fullName))
}

// MethodSelector returns the local selector for a gRPC method.
func MethodSelector(serviceFullName, methodName string) string {
	return ServiceSelector(serviceFullName) + "/methods/" + escapeJSONPointer(strings.TrimSpace(methodName))
}

// MessageSelector returns the local selector for a protobuf message.
func MessageSelector(fullName string) string {
	return "#/messages/" + escapeJSONPointer(strings.TrimSpace(fullName))
}

// MethodByID returns a method by stable source operation ID.
func (m *Model) MethodByID(id string) (*Method, bool) {
	if m == nil {
		return nil, false
	}
	id = strings.TrimSpace(id)
	for _, file := range m.Files {
		for _, service := range file.Services {
			for _, method := range service.Methods {
				if method != nil && method.SourceOperationID == id {
					return method, true
				}
			}
		}
	}
	return nil, false
}

// MethodSummaries returns deterministic summaries for all gRPC methods.
func (m *Model) MethodSummaries() []MethodSummary {
	if m == nil {
		return nil
	}
	out := []MethodSummary{}
	for _, file := range m.Files {
		for _, service := range file.Services {
			for _, method := range service.Methods {
				if method == nil || strings.TrimSpace(method.SourceOperationID) == "" {
					continue
				}
				out = append(out, m.methodSummary(method))
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// SelectorAliases returns all supported selector aliases for every method.
func (m *Model) SelectorAliases() map[string]string {
	if m == nil {
		return nil
	}
	out := map[string]string{}
	for _, summary := range m.MethodSummaries() {
		for _, alias := range methodSelectorAliases(summary) {
			out[alias] = summary.Selector
		}
	}
	return out
}

// ResolveSelector resolves method selector aliases, including gRPC full method
// names, source operation IDs, rpc:<id>, and JSON-pointer selectors.
func (m *Model) ResolveSelector(selector string) (SelectorTarget, error) {
	if m == nil {
		return SelectorTarget{}, fmt.Errorf("grpcproto: nil model")
	}
	selector = strings.TrimSpace(selector)
	for _, file := range m.Files {
		for _, service := range file.Services {
			for _, method := range service.Methods {
				if method == nil {
					continue
				}
				summary := m.methodSummary(method)
				for _, alias := range methodSelectorAliases(summary) {
					if selector == alias {
						return SelectorTarget{Kind: SelectorKindMethod, Method: method, Summary: summary}, nil
					}
				}
			}
		}
	}
	return SelectorTarget{}, fmt.Errorf("grpcproto: unknown selector %q", selector)
}

func (m *Model) methodSummary(method *Method) MethodSummary {
	summary := MethodSummary{
		ID:                 method.SourceOperationID,
		Name:               method.Name,
		ServiceName:        method.ServiceName,
		Package:            method.Package,
		Selector:           method.Selector,
		SourceOperationID:  method.SourceOperationID,
		SourceOperationRef: method.SourceOperationRef,
		FullMethod:         "/" + strings.TrimPrefix(method.SourceOperationID, "/"),
		RequestType:        method.RequestType,
		ResponseType:       method.ResponseType,
		ClientStreaming:    method.ClientStreaming,
		ServerStreaming:    method.ServerStreaming,
	}
	if message := m.messageByName(method.RequestType, method.Package); message != nil {
		for _, field := range message.Fields {
			if field == nil {
				continue
			}
			summary.RequestFields = append(summary.RequestFields, FieldSummary{
				Name:     field.Name,
				Number:   field.Number,
				Label:    field.Label,
				Type:     field.Type,
				TypeName: field.TypeName,
				JSONName: field.JSONName,
				Repeated: field.Repeated,
				Required: field.Required,
			})
		}
	}
	return summary
}

func (m *Model) messageByName(name, packageName string) *Message {
	name = normalizeTypeName(name)
	packageName = strings.TrimSpace(packageName)
	candidates := map[string]struct{}{name: {}}
	if packageName != "" && !strings.Contains(name, ".") {
		candidates[packageName+"."+name] = struct{}{}
	}
	for _, file := range m.Files {
		for _, message := range flattenMessages(file.Messages) {
			if message == nil {
				continue
			}
			if _, ok := candidates[normalizeTypeName(message.FullName)]; ok {
				return message
			}
			if _, ok := candidates[message.Name]; ok {
				return message
			}
		}
	}
	return nil
}

func flattenMessages(messages []*Message) []*Message {
	var out []*Message
	var walk func([]*Message)
	walk = func(items []*Message) {
		for _, item := range items {
			if item == nil {
				continue
			}
			out = append(out, item)
			walk(item.Messages)
		}
	}
	walk(messages)
	return out
}

func methodSelectorAliases(summary MethodSummary) []string {
	var aliases []string
	for _, alias := range []string{
		summary.SourceOperationID,
		summary.FullMethod,
		"rpc:" + summary.SourceOperationID,
		summary.Selector,
		summary.SourceOperationRef,
	} {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}
