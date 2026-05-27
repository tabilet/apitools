package apitools

import (
	"strings"

	"github.com/OpenUdon/apitools/openrpc"
)

// ParseOpenRPCOperationSummaries parses OpenRPC JSON bytes and returns
// prompt-safe summaries for JSON-RPC methods.
func ParseOpenRPCOperationSummaries(data []byte, relativePath string) ([]OperationSummary, error) {
	model, err := openrpc.Parse(data)
	if err != nil {
		return nil, err
	}
	return OpenRPCOperationSummaries(relativePath, model), nil
}

// OpenRPCOperationSummaries converts OpenRPC methods into OperationSummary
// values. It preserves JSON-RPC method metadata and selector refs; it does not
// lower methods to HTTP, execute JSON-RPC calls, or resolve credentials.
func OpenRPCOperationSummaries(relativePath string, model *openrpc.Model) []OperationSummary {
	if model == nil {
		return nil
	}
	methods := model.MethodSummaries()
	operations := make([]OperationSummary, 0, len(methods))
	for _, method := range methods {
		id := strings.TrimSpace(method.ID)
		if id == "" {
			continue
		}
		parameters := make([]ParameterSummary, 0, len(method.Params)+len(method.ParamRefs))
		for _, sourceParam := range method.Params {
			parameters = append(parameters, ParameterSummary{
				Name:     sourceParam.Name,
				In:       "json-rpc",
				Required: sourceParam.Required,
				Ref:      sourceParam.Ref,
			})
		}
		if len(method.Params) == 0 {
			for _, ref := range method.ParamRefs {
				parameters = append(parameters, ParameterSummary{In: "json-rpc", Ref: ref})
			}
		}
		extensions := map[string]string{"openrpc_selector": method.Selector}
		if method.ParamStructure != "" {
			extensions["openrpc_param_structure"] = method.ParamStructure
		}
		if method.ResultName != "" {
			extensions["openrpc_result"] = method.ResultName
		}
		if method.ResultRef != "" {
			extensions["openrpc_result_ref"] = method.ResultRef
		}
		operations = append(operations, OperationSummary{
			ID:                   id,
			DocumentName:         model.Info.Title,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          id,
			Method:               "json-rpc",
			Path:                 method.Selector,
			Summary:              method.Summary,
			Description:          method.Description,
			Tags:                 append([]string(nil), method.Tags...),
			Extensions:           extensions,
			Parameters:           parameters,
			Provenance:           "openrpc",
		})
	}
	return operations
}
