package apitools

import (
	"strings"

	"github.com/OpenUdon/apitools/odata"
)

// ParseODataOperationSummaries parses local OData source bytes and returns
// prompt-safe summaries for entity sets, singletons, navigation properties,
// actions, functions, and imports.
func ParseODataOperationSummaries(data []byte, relativePath string) ([]OperationSummary, error) {
	model, err := odata.Parse(data)
	if err != nil {
		return nil, err
	}
	return ODataOperationSummaries(relativePath, model), nil
}

// ODataOperationSummaries converts OData source metadata into OperationSummary
// values. It preserves entity/action/function/query semantics; it does not call
// tenant metadata endpoints, execute OData operations, execute $batch, resolve
// credentials, log in to tenants, or choose accounts.
func ODataOperationSummaries(relativePath string, model *odata.Model) []OperationSummary {
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
		parameters := make([]ParameterSummary, 0, len(source.Parameters)+len(source.QueryOptions))
		for _, param := range source.Parameters {
			parameters = append(parameters, ParameterSummary{
				Name:     param.Name,
				In:       "odata-parameter",
				Required: !param.Nullable,
				Type:     param.Type,
			})
		}
		for _, option := range source.QueryOptions {
			parameters = append(parameters, ParameterSummary{Name: option, In: "odata-query-option"})
		}
		extensions := map[string]string{
			"odata_kind":           source.Kind,
			"odata_selector":       source.Selector,
			"source_operation_id":  id,
			"source_operation_ref": source.SourceOperationRef,
		}
		if source.EntityType != "" {
			extensions["odata_entity_type"] = source.EntityType
		}
		if source.ReturnType != "" {
			extensions["odata_return_type"] = source.ReturnType
		}
		if source.Bound {
			extensions["odata_bound"] = "true"
		}
		if source.Collection {
			extensions["odata_collection"] = "true"
		}
		operations = append(operations, OperationSummary{
			ID:                   id,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          id,
			Method:               "odata",
			Path:                 source.Selector,
			Summary:              source.Name,
			Extensions:           extensions,
			Parameters:           parameters,
			Provenance:           "odata",
		})
	}
	return operations
}
