package graphql

import (
	"fmt"
	"sort"
	"strings"
)

// OperationSelector returns the local selector for a GraphQL operation ID.
func OperationSelector(id string) string {
	return "#/operations/" + escapeJSONPointer(strings.TrimSpace(id))
}

// SchemaFieldSelector returns the local selector for a GraphQL schema root
// field.
func SchemaFieldSelector(rootType, fieldName string) string {
	return "#/schema/" + escapeJSONPointer(strings.TrimSpace(rootType)) + "/fields/" + escapeJSONPointer(strings.TrimSpace(fieldName))
}

// TypeSelector returns the local selector for a GraphQL type.
func TypeSelector(name string) string {
	return "#/schema/types/" + escapeJSONPointer(strings.TrimSpace(name))
}

// OperationByID returns an operation by stable source operation ID.
func (m *Model) OperationByID(id string) (*Operation, bool) {
	if m == nil {
		return nil, false
	}
	id = strings.TrimSpace(id)
	for _, operation := range m.Operations {
		if operation != nil && operation.ID == id {
			return operation, true
		}
	}
	return nil, false
}

// OperationSummaries returns deterministic summaries for all GraphQL operations.
func (m *Model) OperationSummaries() []OperationSummary {
	if m == nil {
		return nil
	}
	out := make([]OperationSummary, 0, len(m.Operations))
	for _, operation := range m.Operations {
		if operation == nil || strings.TrimSpace(operation.ID) == "" {
			continue
		}
		out = append(out, operationSummary(operation))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// SelectorAliases returns all supported selector aliases for every operation.
func (m *Model) SelectorAliases() map[string]string {
	if m == nil {
		return nil
	}
	out := map[string]string{}
	for _, operation := range m.Operations {
		if operation == nil || strings.TrimSpace(operation.ID) == "" {
			continue
		}
		for _, alias := range operationSelectorAliases(operation) {
			out[alias] = operation.Selector
		}
	}
	return out
}

// ResolveSelector resolves supported GraphQL selector aliases, including the
// source operation ID, operation name, operation:<id>, and JSON-pointer selector.
func (m *Model) ResolveSelector(selector string) (SelectorTarget, error) {
	if m == nil {
		return SelectorTarget{}, fmt.Errorf("graphql: nil model")
	}
	selector = strings.TrimSpace(selector)
	for _, operation := range m.Operations {
		if operation == nil {
			continue
		}
		for _, alias := range operationSelectorAliases(operation) {
			if selector == alias {
				return SelectorTarget{
					Kind:      SelectorKindOperation,
					Operation: operation,
					Summary:   operationSummary(operation),
				}, nil
			}
		}
	}
	return SelectorTarget{}, fmt.Errorf("graphql: unknown selector %q", selector)
}

func operationSummary(operation *Operation) OperationSummary {
	summary := OperationSummary{
		ID:                 operation.ID,
		Name:               operation.Name,
		Kind:               operation.Kind,
		Selector:           operation.Selector,
		SourceOperationID:  operation.ID,
		SourceOperationRef: operation.SourceRef,
		Summary:            operation.Summary,
		Description:        operation.Description,
		SelectionNames:     append([]string(nil), operation.SelectionNames...),
		FieldName:          operation.FieldName,
		FieldType:          operation.FieldType.Display,
		RootType:           operation.RootType,
	}
	for _, variable := range operation.Variables {
		if variable == nil {
			continue
		}
		variableSummary := VariableSummary{
			Name:         variable.Name,
			Type:         variable.Type.Display,
			Required:     variable.Required,
			DefaultValue: variable.DefaultValue,
		}
		summary.Variables = append(summary.Variables, variableSummary)
		if variableSummary.Name != "" {
			summary.VariableNames = append(summary.VariableNames, variableSummary.Name)
		}
	}
	sort.Strings(summary.SelectionNames)
	return summary
}

func operationSelectorAliases(operation *Operation) []string {
	if operation == nil {
		return nil
	}
	var aliases []string
	for _, alias := range []string{
		operation.ID,
		operation.Name,
		"operation:" + operation.ID,
		operation.Selector,
		operation.SourceRef,
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
