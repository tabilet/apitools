package apitools

import (
	"strings"

	"github.com/OpenUdon/apitools/grpcproto"
)

// ParseGRPCProtobufOperationSummaries parses local gRPC/protobuf source bytes
// and returns prompt-safe summaries for service methods.
func ParseGRPCProtobufOperationSummaries(data []byte, relativePath string) ([]OperationSummary, error) {
	model, err := grpcproto.Parse(data)
	if err != nil {
		return nil, err
	}
	return GRPCProtobufOperationSummaries(relativePath, model), nil
}

// GRPCProtobufOperationSummaries converts gRPC/protobuf methods into
// OperationSummary values. It preserves service, method, request/response, and
// streaming metadata; it does not execute RPCs, use server reflection, open
// channels, negotiate TLS, or resolve credentials.
func GRPCProtobufOperationSummaries(relativePath string, model *grpcproto.Model) []OperationSummary {
	if model == nil {
		return nil
	}
	sourceSummaries := model.MethodSummaries()
	operations := make([]OperationSummary, 0, len(sourceSummaries))
	for _, source := range sourceSummaries {
		id := strings.TrimSpace(source.SourceOperationID)
		if id == "" {
			continue
		}
		extensions := map[string]string{
			"grpc_full_method":     source.FullMethod,
			"grpc_selector":        source.Selector,
			"grpc_service":         source.ServiceName,
			"grpc_request_type":    source.RequestType,
			"grpc_response_type":   source.ResponseType,
			"source_operation_id":  id,
			"source_operation_ref": source.SourceOperationRef,
		}
		if source.ClientStreaming {
			extensions["grpc_client_streaming"] = "true"
		}
		if source.ServerStreaming {
			extensions["grpc_server_streaming"] = "true"
		}
		requestBody := &RequestBodySummary{
			ContentTypes: []string{"application/grpc+proto"},
			Schema:       &SchemaSummary{Ref: source.RequestType},
		}
		for _, field := range source.RequestFields {
			requestBody.Fields = append(requestBody.Fields, RequestFieldSummary{
				Path:     firstNonEmpty(field.JSONName, field.Name),
				Required: field.Required,
				Type:     grpcFieldType(field),
				Ref:      field.TypeName,
			})
		}
		operations = append(operations, OperationSummary{
			ID:                   id,
			DocumentPath:         relativePath,
			DocumentRelativePath: relativePath,
			OperationID:          id,
			Method:               "grpc",
			Path:                 source.Selector,
			Summary:              source.Name,
			Extensions:           extensions,
			RequestBody:          requestBody,
			Provenance:           "grpc-protobuf",
		})
	}
	return operations
}

func grpcFieldType(field grpcproto.FieldSummary) string {
	if field.TypeName != "" {
		return field.TypeName
	}
	return field.Type
}
