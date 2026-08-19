package apitools

import (
	"bytes"
	"strings"

	"github.com/OpenUdon/apitools/internal/sourceguard"
	"github.com/OpenUdon/asyncapi"
)

// ParseAsyncAPIOperationSummaries parses AsyncAPI YAML or JSON bytes and
// returns prompt-safe summaries for root AsyncAPI operations.
func ParseAsyncAPIOperationSummaries(data []byte, relativePath string) ([]OperationSummary, error) {
	if err := sourceguard.CheckDocument("asyncapi", data); err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		if err := sourceguard.CheckJSON("asyncapi", trimmed); err != nil {
			return nil, err
		}
	} else if err := sourceguard.CheckYAML("asyncapi", trimmed); err != nil {
		return nil, err
	}
	model, err := asyncapi.Parse(data)
	if err != nil {
		return nil, err
	}
	return AsyncAPIOperationSummaries(relativePath, model), nil
}

// AsyncAPIOperationSummaries converts root AsyncAPI operations into
// OperationSummary values. It preserves AsyncAPI as source metadata; it does not
// lower operations to HTTP, execute protocols, or resolve external references.
func AsyncAPIOperationSummaries(relativePath string, model *asyncapi.Model) []OperationSummary {
	if model == nil {
		return nil
	}
	operations := make([]OperationSummary, 0, len(model.Operations))
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.ID) == "" {
			continue
		}
		path := op.ChannelRef
		if strings.TrimSpace(path) == "" && len(op.MessageRefs) > 0 {
			path = op.MessageRefs[0]
		}
		operations = append(operations, OperationSummary{
			ID:                   op.ID,
			DocumentName:         model.Title,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          op.ID,
			Method:               op.Action,
			Path:                 path,
			Summary:              op.Summary,
			Description:          op.Description,
			Provenance:           "asyncapi",
		})
	}
	return operations
}
