package apitools

import (
	"strings"

	"github.com/OpenUdon/apitools/graphql"
)

// ParseGraphQLOperationSummaries parses local GraphQL source bytes and returns
// prompt-safe summaries for schema root fields or operation documents.
func ParseGraphQLOperationSummaries(data []byte, relativePath string) ([]OperationSummary, error) {
	model, err := graphql.Parse(data)
	if err != nil {
		return nil, err
	}
	return GraphQLOperationSummaries(relativePath, model), nil
}

// GraphQLOperationSummaries converts GraphQL source operations into
// OperationSummary values. It preserves GraphQL operation kind, variables, and
// source refs; it does not execute GraphQL operations, introspect live
// endpoints, resolve variables, or resolve credentials.
func GraphQLOperationSummaries(relativePath string, model *graphql.Model) []OperationSummary {
	if model == nil {
		return nil
	}
	sourceSummaries := model.OperationSummaries()
	operations := make([]OperationSummary, 0, len(sourceSummaries))
	for _, source := range sourceSummaries {
		id := strings.TrimSpace(source.SourceOperationID)
		if id == "" {
			id = strings.TrimSpace(source.ID)
		}
		if id == "" {
			continue
		}
		parameters := make([]ParameterSummary, 0, len(source.Variables))
		for _, variable := range source.Variables {
			parameters = append(parameters, ParameterSummary{
				Name:     variable.Name,
				In:       "graphql-variable",
				Required: variable.Required,
				Type:     variable.Type,
			})
		}
		extensions := map[string]string{
			"graphql_operation_kind": source.Kind,
			"graphql_selector":       source.Selector,
			"source_operation_id":    id,
			"source_operation_ref":   source.SourceOperationRef,
		}
		if source.FieldName != "" {
			extensions["graphql_field"] = source.FieldName
		}
		if source.FieldType != "" {
			extensions["graphql_field_type"] = source.FieldType
		}
		if source.RootType != "" {
			extensions["graphql_root_type"] = source.RootType
		}
		operations = append(operations, OperationSummary{
			ID:                   id,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          id,
			Method:               source.Kind,
			Path:                 source.Selector,
			Summary:              source.Summary,
			Description:          source.Description,
			Extensions:           extensions,
			Parameters:           parameters,
			Provenance:           "graphql",
		})
	}
	return operations
}
