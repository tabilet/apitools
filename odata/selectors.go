package odata

import (
	"fmt"
	"sort"
	"strings"
)

// EntitySetSelector returns the local selector for an OData entity set.
func EntitySetSelector(name string) string {
	return "#/entitySets/" + escapeJSONPointer(strings.TrimSpace(name))
}

// SingletonSelector returns the local selector for an OData singleton.
func SingletonSelector(name string) string {
	return "#/singletons/" + escapeJSONPointer(strings.TrimSpace(name))
}

// OperationSelector returns the local selector for an OData action or function.
func OperationSelector(kind, fullName string) string {
	return "#/" + strings.ToLower(strings.TrimSpace(kind)) + "s/" + escapeJSONPointer(strings.TrimSpace(fullName))
}

// NavigationSelector returns the local selector for an OData navigation property.
func NavigationSelector(typeName, propertyName string) string {
	return "#/entityTypes/" + escapeJSONPointer(strings.TrimSpace(typeName)) + "/navigation/" + escapeJSONPointer(strings.TrimSpace(propertyName))
}

// OperationSummaries returns deterministic summaries for OData resources,
// actions, functions, imports, and navigation properties.
func (m *Model) OperationSummaries() []OperationSummary {
	if m == nil {
		return nil
	}
	var out []OperationSummary
	for _, schema := range m.Schemas {
		if schema == nil {
			continue
		}
		if schema.EntityContainer != nil {
			for _, entitySet := range schema.EntityContainer.EntitySets {
				if entitySet == nil || entitySet.Name == "" {
					continue
				}
				id := "entityset." + entitySet.Name
				out = append(out, OperationSummary{
					ID:                 id,
					Name:               entitySet.Name,
					Kind:               "entity-set",
					Selector:           entitySet.Selector,
					SourceOperationID:  id,
					SourceOperationRef: entitySet.Selector,
					EntityType:         entitySet.EntityType,
					Collection:         true,
					QueryOptions:       []string{"$select", "$filter", "$orderby", "$top", "$skip", "$expand", "$count"},
				})
			}
			for _, singleton := range schema.EntityContainer.Singletons {
				if singleton == nil || singleton.Name == "" {
					continue
				}
				id := "singleton." + singleton.Name
				out = append(out, OperationSummary{
					ID:                 id,
					Name:               singleton.Name,
					Kind:               "singleton",
					Selector:           singleton.Selector,
					SourceOperationID:  id,
					SourceOperationRef: singleton.Selector,
					EntityType:         singleton.EntityType,
					QueryOptions:       []string{"$select", "$expand"},
				})
			}
			for _, opImport := range schema.EntityContainer.ActionImports {
				out = append(out, operationImportSummary(opImport))
			}
			for _, opImport := range schema.EntityContainer.FunctionImports {
				out = append(out, operationImportSummary(opImport))
			}
		}
		for _, operation := range schema.Actions {
			out = append(out, operationSummary(operation))
		}
		for _, operation := range schema.Functions {
			out = append(out, operationSummary(operation))
		}
		for _, entityType := range schema.EntityTypes {
			if entityType == nil {
				continue
			}
			for _, nav := range entityType.NavigationProperties {
				if nav == nil || nav.Name == "" {
					continue
				}
				id := "navigation." + entityType.FullName + "." + nav.Name
				out = append(out, OperationSummary{
					ID:                 id,
					Name:               nav.Name,
					Kind:               "navigation",
					Selector:           nav.Selector,
					SourceOperationID:  id,
					SourceOperationRef: nav.Selector,
					EntityType:         nav.Type,
					Collection:         nav.Collection,
					QueryOptions:       []string{"$select", "$filter", "$orderby", "$top", "$skip", "$expand", "$count"},
				})
			}
		}
	}
	out = compactSummaries(out)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ResolveSelector resolves OData selector aliases, including source operation
// IDs and local JSON-pointer selectors.
func (m *Model) ResolveSelector(selector string) (SelectorTarget, error) {
	if m == nil {
		return SelectorTarget{}, fmt.Errorf("odata: nil model")
	}
	selector = strings.TrimSpace(selector)
	for _, summary := range m.OperationSummaries() {
		for _, alias := range operationAliases(summary) {
			if selector == alias {
				return SelectorTarget{Kind: SelectorKindOperation, Summary: summary}, nil
			}
		}
	}
	return SelectorTarget{}, fmt.Errorf("odata: unknown selector %q", selector)
}

func operationSummary(operation *Operation) OperationSummary {
	if operation == nil {
		return OperationSummary{}
	}
	summary := OperationSummary{
		ID:                 operation.SourceOperationID,
		Name:               operation.Name,
		Kind:               operation.Kind,
		Selector:           operation.Selector,
		SourceOperationID:  operation.SourceOperationID,
		SourceOperationRef: operation.SourceOperationRef,
		ReturnType:         operation.ReturnType,
		Bound:              operation.Bound,
		QueryOptions:       operationQueryOptions(operation.Kind),
	}
	for _, parameter := range operation.Parameters {
		if parameter == nil {
			continue
		}
		summary.Parameters = append(summary.Parameters, ParameterSummary{
			Name:       parameter.Name,
			Type:       parameter.Type,
			Nullable:   parameter.Nullable,
			Collection: parameter.Collection,
		})
	}
	return summary
}

func operationImportSummary(opImport *OperationImport) OperationSummary {
	if opImport == nil {
		return OperationSummary{}
	}
	return OperationSummary{
		ID:                 opImport.SourceOperationID,
		Name:               opImport.Name,
		Kind:               opImport.Kind + "-import",
		Selector:           opImport.Selector,
		SourceOperationID:  opImport.SourceOperationID,
		SourceOperationRef: opImport.SourceOperationRef,
		ReturnType:         opImport.EntitySet,
		QueryOptions:       operationQueryOptions(opImport.Kind),
	}
}

func operationQueryOptions(kind string) []string {
	if strings.EqualFold(kind, "function") {
		return []string{"$select", "$filter", "$orderby", "$top", "$skip", "$expand", "$count"}
	}
	return nil
}

func compactSummaries(in []OperationSummary) []OperationSummary {
	seen := map[string]struct{}{}
	var out []OperationSummary
	for _, summary := range in {
		if strings.TrimSpace(summary.ID) == "" {
			continue
		}
		if _, exists := seen[summary.ID]; exists {
			continue
		}
		seen[summary.ID] = struct{}{}
		out = append(out, summary)
	}
	return out
}

func operationAliases(summary OperationSummary) []string {
	var aliases []string
	for _, alias := range []string{summary.ID, summary.Name, summary.Selector, summary.SourceOperationRef, "odata:" + summary.ID} {
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
