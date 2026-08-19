package apitools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/internal/sourceguard"
	upstreamsmithy "github.com/OpenUdon/awssmithy"
	upstreamdiscovery "github.com/OpenUdon/googlediscovery"
)

const (
	APISourceKindOpenAPI         = "openapi"
	APISourceKindAWSSmithy       = "aws-smithy"
	APISourceKindGoogleDiscovery = "google-discovery"
)

// APISourceDocument is one local API source document to inspect.
type APISourceDocument struct {
	Kind         string `json:"kind"`
	Name         string `json:"name,omitempty"`
	Path         string `json:"path,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	Content      []byte `json:"-"`
}

// APISourceInventoryOptions configures native API source operation extraction.
type APISourceInventoryOptions struct {
	Documents     []APISourceDocument `json:"documents,omitempty"`
	Query         string              `json:"query,omitempty"`
	MaxBytes      int64               `json:"max_bytes,omitempty"`
	MaxOperations int                 `json:"max_operations,omitempty"`
}

// BuildAPISourceOperationInventory extracts prompt-safe operation summaries
// from OpenAPI/Swagger, AWS Smithy JSON, Google Discovery, AsyncAPI, GraphQL,
// OpenRPC, gRPC/protobuf, and OData source documents. It preserves native
// source families rather than lowering them into OpenAPI.
func BuildAPISourceOperationInventory(ctx context.Context, opts APISourceInventoryOptions) (OperationInventory, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(opts.Documents) == 0 {
		return OperationInventory{}, fmt.Errorf("at least one API source document is required")
	}
	maxOperations := resolvedInventoryMaxOperations(opts.MaxOperations)
	var inventory OperationInventory
	for i, doc := range opts.Documents {
		if err := ctx.Err(); err != nil {
			return inventory, err
		}
		if len(inventory.Operations) >= maxOperations {
			markInventoryTruncated(&inventory, maxOperations)
			break
		}
		kind := normalizeAPISourceKind(doc.Kind)
		if kind == "" {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "document.kind",
				Message:  fmt.Sprintf("unsupported API source kind %q", doc.Kind),
				Path:     doc.Path,
			})
			continue
		}
		if kind == APISourceKindOpenAPI {
			remaining := maxOperations - len(inventory.Operations)
			if remaining <= 0 {
				markInventoryTruncated(&inventory, maxOperations)
				break
			}
			openapi, err := BuildOperationInventory(ctx, InventoryOptions{
				Documents: []InventoryDocument{{
					Name:         doc.Name,
					Path:         doc.Path,
					RelativePath: doc.RelativePath,
					Content:      doc.Content,
				}},
				Query:         opts.Query,
				MaxBytes:      opts.MaxBytes,
				MaxOperations: remaining,
			})
			if err != nil {
				return inventory, err
			}
			inventory.Documents = append(inventory.Documents, openapi.Documents...)
			inventory.Operations = append(inventory.Operations, openapi.Operations...)
			inventory.Diagnostics = append(inventory.Diagnostics, openapi.Diagnostics...)
			inventory.ReadinessIssues = append(inventory.ReadinessIssues, openapi.ReadinessIssues...)
			inventory.VisitedOperations += openapi.VisitedOperations
			if openapi.Truncated {
				inventory.Truncated = true
				break
			}
			continue
		}
		content := doc.Content
		if len(content) > 0 {
			if err := validateInlineSpecContent(content, opts.MaxBytes, firstNonEmpty(doc.Path, doc.Name)); err != nil {
				inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{Severity: "error", Code: "document.read", Message: err.Error(), Path: doc.Path})
				continue
			}
		}
		if len(content) == 0 {
			data, err := readLocalSpecFile(doc.Path, opts.MaxBytes)
			if err != nil {
				inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
					Severity: "error",
					Code:     "document.read",
					Message:  err.Error(),
					Path:     doc.Path,
				})
				continue
			}
			content = data
		}
		beforeOperations := len(inventory.Operations)
		beforeDocuments := len(inventory.Documents)
		if kind == APISourceKindAWSSmithy || kind == APISourceKindGoogleDiscovery {
			if err := sourceguard.CheckJSON(kind, content); err != nil {
				inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{Severity: "error", Code: "document.parse", Message: err.Error(), Path: doc.Path})
				continue
			}
		}
		switch kind {
		case APISourceKindAWSSmithy:
			addAWSSmithyInventory(&inventory, doc, i, content, opts.Query)
		case APISourceKindGoogleDiscovery:
			addGoogleDiscoveryInventory(&inventory, doc, i, content, opts.Query)
		case APISourceKindAsyncAPI, APISourceKindGraphQL, APISourceKindOpenRPC, APISourceKindGRPCProtobuf, APISourceKindOData:
			addAdditionalSourceInventory(&inventory, doc, i, kind, content, opts.Query)
		}
		if err := ctx.Err(); err != nil {
			return inventory, err
		}
		if enforceInventoryOperationLimit(&inventory, beforeOperations, beforeDocuments, maxOperations) {
			break
		}
	}
	sort.SliceStable(inventory.Operations, func(i, j int) bool {
		left, right := inventory.Operations[i], inventory.Operations[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		if left.DocumentPath != right.DocumentPath {
			return left.DocumentPath < right.DocumentPath
		}
		if left.DocumentName != right.DocumentName {
			return left.DocumentName < right.DocumentName
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		return left.OperationID < right.OperationID
	})
	return inventory, nil
}

func normalizeAPISourceKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case APISourceKindOpenAPI, "swagger":
		return APISourceKindOpenAPI
	case APISourceKindAWSSmithy, "smithy", "smithy-json":
		return APISourceKindAWSSmithy
	case APISourceKindGoogleDiscovery, "discovery", "google":
		return APISourceKindGoogleDiscovery
	case APISourceKindAsyncAPI:
		return APISourceKindAsyncAPI
	case APISourceKindGraphQL:
		return APISourceKindGraphQL
	case APISourceKindOpenRPC:
		return APISourceKindOpenRPC
	case APISourceKindGRPCProtobuf, "grpc", "protobuf", "proto":
		return APISourceKindGRPCProtobuf
	case APISourceKindOData:
		return APISourceKindOData
	default:
		return ""
	}
}

func addAdditionalSourceInventory(inventory *OperationInventory, doc APISourceDocument, index int, kind string, content []byte, query string) {
	path := firstNonEmpty(doc.RelativePath, doc.Path, doc.Name)
	var operations []OperationSummary
	var err error
	switch kind {
	case APISourceKindAsyncAPI:
		operations, err = ParseAsyncAPIOperationSummaries(content, path)
	case APISourceKindGraphQL:
		operations, err = ParseGraphQLOperationSummaries(content, path)
	case APISourceKindOpenRPC:
		operations, err = ParseOpenRPCOperationSummaries(content, path)
	case APISourceKindGRPCProtobuf:
		operations, err = ParseGRPCProtobufOperationSummaries(content, path)
	case APISourceKindOData:
		operations, err = ParseODataOperationSummaries(content, path)
	}
	if err != nil {
		inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
			Severity: "error", Code: "document.parse", Message: err.Error(), Path: path,
			Remediation: "Provide a valid " + kind + " source document.",
		})
		return
	}
	name := firstNonEmpty(doc.Name, func() string {
		if len(operations) > 0 {
			return operations[0].DocumentName
		}
		return ""
	}(), fmt.Sprintf("document-%d", index+1))
	for operationIndex := range operations {
		operation := &operations[operationIndex]
		operation.DocumentName = firstNonEmpty(operation.DocumentName, name)
		operation.DocumentPath = firstNonEmpty(doc.Path, operation.DocumentPath)
		operation.DocumentRelativePath = firstNonEmpty(doc.RelativePath, operation.DocumentRelativePath)
		if operation.Extensions == nil {
			operation.Extensions = map[string]string{}
		}
		operation.Extensions["x-uws-source-kind"] = kind
		operation.Score = ScoreText(query, operationSearchText(*operation))
	}
	inventory.Documents = append(inventory.Documents, DocumentSummary{
		Name: name, Path: doc.Path, RelativePath: doc.RelativePath, Title: name, OperationCount: len(operations),
	})
	inventory.Operations = append(inventory.Operations, operations...)
}

func addAWSSmithyInventory(inventory *OperationInventory, doc APISourceDocument, index int, content []byte, query string) {
	model, err := upstreamsmithy.Parse(content)
	if err != nil {
		inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
			Severity:    "error",
			Code:        "document.parse",
			Message:     err.Error(),
			Path:        firstNonEmpty(doc.Path, doc.Name),
			Remediation: "Provide a JSON AWS Smithy service model.",
		})
		return
	}
	name := firstNonEmpty(doc.Name, model.ServiceID, model.Title, model.AWSServiceID, fmt.Sprintf("document-%d", index+1))
	summary := DocumentSummary{
		Name:           name,
		Path:           doc.Path,
		RelativePath:   doc.RelativePath,
		Title:          firstNonEmpty(model.Title, model.ServiceID),
		Description:    model.Description,
		OperationCount: len(model.Operations),
	}
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.Name) == "" {
			continue
		}
		operation := awsSmithyOperationSummary(name, doc, model, op)
		operation.Score = ScoreText(query, operationSearchText(operation))
		inventory.Operations = append(inventory.Operations, operation)
	}
	inventory.Documents = append(inventory.Documents, summary)
}

func awsSmithyOperationSummary(documentName string, doc APISourceDocument, model *upstreamsmithy.Model, op *upstreamsmithy.Operation) OperationSummary {
	summary := OperationSummary{
		ID:                   op.Name,
		DocumentName:         documentName,
		DocumentPath:         doc.Path,
		DocumentRelativePath: doc.RelativePath,
		OperationID:          op.Name,
		Method:               op.Method,
		Path:                 op.URI,
		Parameters:           awsSmithyRequestParameters(op),
		Extensions: map[string]string{
			"x-uws-source-kind": APISourceKindAWSSmithy,
		},
		Provenance: firstNonEmpty(doc.Path, documentName) + "#" + op.Name,
		Security: []SecuritySummary{{
			Name:          "hmac",
			Type:          "apiKey",
			In:            "header",
			ParameterName: "Authorization",
			Description:   "AWS Signature Version 4 metadata for native Smithy source execution.",
			Extensions: map[string]string{
				"x-aws-signing-name": firstNonEmpty(model.SigningName, model.EndpointPrefix, model.AWSServiceID),
			},
		}},
	}
	if summary.Method == "" {
		summary.Method = "POST"
	}
	if summary.Path == "" {
		summary.Path = "/"
	}
	if body := awsSmithyRequestBodySummary(op); body != nil {
		summary.RequestBody = body
	}
	return summary
}

func awsSmithyRequestParameters(op *upstreamsmithy.Operation) []ParameterSummary {
	var out []ParameterSummary
	for _, binding := range op.InputBindings {
		location, ok := awsSmithyNonBodyLocation(binding)
		if !ok {
			continue
		}
		out = append(out, ParameterSummary{
			Name:     firstNonEmpty(binding.WireName, binding.MemberName),
			In:       location,
			Required: binding.Required,
			Ref:      binding.Target,
		})
	}
	return out
}

func awsSmithyRequestBodySummary(op *upstreamsmithy.Operation) *RequestBodySummary {
	if op == nil || (op.Payload == nil && len(op.UnboundInput) == 0 && len(op.StaticPayload) == 0) {
		return nil
	}
	body := &RequestBodySummary{
		ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
		Ref:          op.Input,
	}
	add := func(binding *upstreamsmithy.MemberBinding) {
		if binding == nil || strings.TrimSpace(binding.MemberName) == "" {
			return
		}
		field := RequestFieldSummary{
			Path:     strings.TrimSpace(binding.MemberName),
			Required: binding.Required,
			Ref:      binding.Target,
		}
		body.Fields = append(body.Fields, field)
		if field.Required {
			body.Required = true
			body.RequiredFieldPaths = append(body.RequiredFieldPaths, field.Path)
		}
	}
	add(op.Payload)
	for _, binding := range op.UnboundInput {
		add(binding)
	}
	sort.Strings(body.RequiredFieldPaths)
	return body
}

func awsSmithyNonBodyLocation(binding *upstreamsmithy.MemberBinding) (string, bool) {
	if binding == nil {
		return "", false
	}
	switch strings.TrimSpace(binding.Location) {
	case "label", "path":
		return "path", true
	case "query", "queryParams":
		return "query", true
	case "header", "prefixHeaders":
		return "header", true
	default:
		return "", false
	}
}

func addGoogleDiscoveryInventory(inventory *OperationInventory, doc APISourceDocument, index int, content []byte, query string) {
	model, err := upstreamdiscovery.Parse(content)
	if err != nil {
		inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
			Severity:    "error",
			Code:        "document.parse",
			Message:     err.Error(),
			Path:        firstNonEmpty(doc.Path, doc.Name),
			Remediation: "Provide a JSON Google Discovery document.",
		})
		return
	}
	name := firstNonEmpty(doc.Name, model.Name, model.Title, fmt.Sprintf("document-%d", index+1))
	summary := DocumentSummary{
		Name:           name,
		Path:           doc.Path,
		RelativePath:   doc.RelativePath,
		Title:          firstNonEmpty(model.Title, model.Name),
		Description:    model.Description,
		OperationCount: len(model.Operations),
	}
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.OperationID) == "" {
			continue
		}
		operation := googleDiscoveryOperationSummary(name, doc, model, op)
		operation.Score = ScoreText(query, operationSearchText(operation))
		inventory.Operations = append(inventory.Operations, operation)
	}
	inventory.Documents = append(inventory.Documents, summary)
}

func googleDiscoveryOperationSummary(documentName string, doc APISourceDocument, model *upstreamdiscovery.Model, op *upstreamdiscovery.Operation) OperationSummary {
	operationID := firstNonEmpty(op.ID, op.OperationID)
	return OperationSummary{
		ID:                   operationID,
		DocumentName:         documentName,
		DocumentPath:         doc.Path,
		DocumentRelativePath: doc.RelativePath,
		OperationID:          operationID,
		Method:               op.HTTPMethod,
		Path:                 op.Path,
		Summary:              op.Summary,
		Description:          op.Description,
		Tags:                 append([]string(nil), op.Tags...),
		Parameters:           googleDiscoveryRequestParameters(op.Parameters),
		RequestBody:          googleDiscoveryRequestBodySummary(model, op),
		ResponseBody:         googleDiscoveryResponseBodySummary(model, op),
		Security: []SecuritySummary{{
			Name:   "googleOAuth2",
			Type:   "oauth2",
			Scopes: append([]string(nil), op.Scopes...),
		}},
		Extensions: map[string]string{
			"x-uws-source-kind": APISourceKindGoogleDiscovery,
		},
		Provenance: firstNonEmpty(doc.Path, documentName) + "#" + operationID,
	}
}

func googleDiscoveryRequestParameters(params []*upstreamdiscovery.Parameter) []ParameterSummary {
	var out []ParameterSummary
	for _, param := range params {
		if param == nil {
			continue
		}
		out = append(out, ParameterSummary{
			Name:        param.Name,
			In:          param.Location,
			Required:    param.Required,
			Description: param.Description,
			Type:        stringFromMap(param.Schema, "type"),
			Format:      stringFromMap(param.Schema, "format"),
		})
	}
	return out
}

func googleDiscoveryRequestBodySummary(model *upstreamdiscovery.Model, op *upstreamdiscovery.Operation) *RequestBodySummary {
	if op == nil || strings.TrimSpace(op.RequestRef) == "" {
		return nil
	}
	body := &RequestBodySummary{
		Required:     true,
		Ref:          op.RequestRef,
		ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
	}
	if model == nil {
		return body
	}
	schemaName := strings.TrimPrefix(op.RequestRef, "#/components/schemas/")
	schema, ok := model.Schemas[schemaName]
	if !ok {
		return body
	}
	body.Description = stringFromMap(schema, "description")
	body.Fields = googleDiscoveryRequestFields(schema, op.OperationID)
	for _, field := range body.Fields {
		if field.Required {
			body.RequiredFieldPaths = append(body.RequiredFieldPaths, field.Path)
		}
	}
	return body
}

func googleDiscoveryResponseBodySummary(model *upstreamdiscovery.Model, op *upstreamdiscovery.Operation) *ResponseBodySummary {
	if op == nil || strings.TrimSpace(op.ResponseRef) == "" {
		return nil
	}
	body := &ResponseBodySummary{
		StatusCode:   "2xx",
		Ref:          op.ResponseRef,
		ContentTypes: []string{firstNonEmpty(op.ResponseMediaType, "application/json")},
	}
	if model == nil {
		return body
	}
	schemaName := strings.TrimPrefix(op.ResponseRef, "#/components/schemas/")
	schema, ok := model.Schemas[schemaName]
	if !ok {
		return body
	}
	body.Description = stringFromMap(schema, "description")
	summary := schemaSummary(schema)
	body.Schema = &summary
	body.Fields = googleDiscoveryResponseFields(schema)
	return body
}

func googleDiscoveryResponseFields(schema map[string]any) []RequestFieldSummary {
	return appendMissingTopLevelResponseFields(requestFieldSummaries(schema, "", false, 0), schema, "name", "id")
}

func googleDiscoveryRequestFields(schema map[string]any, operationID string) []RequestFieldSummary {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return nil
	}
	required := stringBoolSet(stringSliceAny(schema["required"]))
	var out []RequestFieldSummary
	for _, name := range sortedAnyMapKeys(props) {
		prop, _ := props[name].(map[string]any)
		if len(prop) == 0 {
			continue
		}
		out = append(out, RequestFieldSummary{
			Path:        name,
			Required:    required[name] || googleDiscoveryPropertyRequiredForOperation(prop, operationID),
			Type:        stringFromMap(prop, "type"),
			Format:      stringFromMap(prop, "format"),
			Ref:         stringFromMap(prop, "$ref"),
			Description: stringFromMap(prop, "description"),
		})
	}
	return out
}

func googleDiscoveryPropertyRequiredForOperation(prop map[string]any, operationID string) bool {
	annotations, _ := prop["annotations"].(map[string]any)
	for _, required := range stringSliceAny(annotations["required"]) {
		if required == operationID || sanitizeName(required) == sanitizeName(operationID) {
			return true
		}
	}
	return false
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func stringSliceAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func stringBoolSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out[strings.TrimSpace(value)] = true
		}
	}
	return out
}

func sortedAnyMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
