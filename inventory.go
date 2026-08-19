package apitools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const DefaultMaxInventoryOperations = 10_000

// BuildOperationInventory extracts prompt-safe operation summaries from local
// OpenAPI or Swagger documents. It never fetches remote references or executes
// discovered operations.
func BuildOperationInventory(ctx context.Context, opts InventoryOptions) (OperationInventory, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(opts.Documents) == 0 {
		return OperationInventory{}, fmt.Errorf("at least one OpenAPI document is required")
	}
	maxOperations := resolvedInventoryMaxOperations(opts.MaxOperations)
	var inventory OperationInventory
	for i, doc := range opts.Documents {
		if err := ctx.Err(); err != nil {
			return inventory, err
		}
		content, err := inventoryDocumentContent(doc, opts.MaxBytes)
		if err != nil {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity: "error",
				Code:     "document.read",
				Message:  err.Error(),
				Path:     doc.Path,
			})
			continue
		}
		parsed, err := parseInventoryDocument(content)
		if err != nil {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity:    "error",
				Code:        "document.parse",
				Message:     err.Error(),
				Path:        firstNonEmpty(doc.Path, doc.URL, doc.Name),
				Remediation: "Provide a JSON or YAML OpenAPI/Swagger document.",
			})
			continue
		}
		limitReached, err := addDocumentInventory(ctx, &inventory, doc, i, parsed, opts.Query, maxOperations)
		if err != nil {
			return inventory, err
		}
		if limitReached {
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
		return left.Method < right.Method
	})
	if opts.Limit > 0 && len(inventory.Operations) > opts.Limit {
		inventory.Operations = inventory.Operations[:opts.Limit]
	}
	return inventory, nil
}

func inventoryDocumentContent(doc InventoryDocument, maxBytes int64) ([]byte, error) {
	if len(doc.Content) > 0 {
		if err := validateInlineSpecContent(doc.Content, maxBytes, firstNonEmpty(doc.Path, doc.URL, doc.Name)); err != nil {
			return nil, err
		}
		return doc.Content, nil
	}
	if strings.TrimSpace(doc.Path) == "" {
		return nil, fmt.Errorf("document content or path is required")
	}
	return readLocalSpecFile(doc.Path, maxBytes)
}

func addDocumentInventory(ctx context.Context, inventory *OperationInventory, doc InventoryDocument, index int, root map[string]any, query string, maxOperations int) (bool, error) {
	info := mapValue(root["info"])
	name := firstNonEmpty(doc.Name, stringValue(info["title"]), filepath.Base(doc.Path), doc.URL, fmt.Sprintf("document-%d", index+1))
	summary := DocumentSummary{
		Name:         name,
		Path:         doc.Path,
		RelativePath: doc.RelativePath,
		URL:          doc.URL,
		Title:        stringValue(info["title"]),
		Description:  stringValue(info["description"]),
		OpenAPI:      stringValue(root["openapi"]),
		Swagger:      stringValue(root["swagger"]),
	}
	securitySchemes := securitySchemes(root)
	defaultSecurity := securityRequirements(root["security"], securitySchemes)
	paths := mapValue(root["paths"])
	pathKeys := sortedMapKeys(paths)
	for _, path := range pathKeys {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		pathItem := mapValue(paths[path])
		if ref := stringValue(pathItem["$ref"]); strings.HasPrefix(ref, "#/") {
			if resolved := localObjectRef(root, ref); len(resolved) > 0 {
				pathItem = mergeObjectRef(resolved, pathItem)
			} else {
				inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{Severity: "error", Code: "document.ref_unresolved", Message: "path item reference was not resolved", Path: ref, Remediation: "Provide a valid local path item reference; remote references are not fetched."})
				continue
			}
		}
		pathParameterValues := sliceValue(pathItem["parameters"])
		for _, method := range operationMethods(pathItem) {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			inventory.VisitedOperations++
			if len(inventory.Operations) >= maxOperations {
				inventory.Documents = append(inventory.Documents, summary)
				markInventoryTruncated(inventory, maxOperations)
				return true, nil
			}
			operation := mapValue(pathItem[method])
			op := operationSummary(doc, name, path, method, operation)
			op.Parameters = append(op.Parameters, parameterSummaries(root, pathParameterValues, &op)...)
			op.Parameters = append(op.Parameters, parameterSummaries(root, sliceValue(operation["parameters"]), &op)...)
			var err error
			op.RequestBody, err = requestBodySummary(ctx, root, operation, &op)
			if err != nil {
				return false, err
			}
			op.ResponseBody, err = responseBodySummary(ctx, root, operation, &op)
			if err != nil {
				return false, err
			}
			if value, ok := operation["security"]; ok {
				op.Security = securityRequirements(value, securitySchemes)
			} else {
				op.Security = defaultSecurity
			}
			op.Score = ScoreText(query, operationSearchText(op))
			if op.OperationID == "" {
				issue := ReadinessIssue{
					Severity:    "warning",
					Code:        "operation.missing_id",
					Message:     "operation is missing operationId",
					Path:        op.Provenance,
					Remediation: "Add operationId to the OpenAPI document or select this operation by inventory id.",
				}
				op.ReadinessIssues = append(op.ReadinessIssues, issue)
			}
			inventory.ReadinessIssues = append(inventory.ReadinessIssues, op.ReadinessIssues...)
			inventory.Operations = append(inventory.Operations, op)
			summary.OperationCount++
		}
	}
	inventory.Documents = append(inventory.Documents, summary)
	return false, nil
}

func resolvedInventoryMaxOperations(limit int) int {
	if limit <= 0 {
		return DefaultMaxInventoryOperations
	}
	return limit
}

func markInventoryTruncated(inventory *OperationInventory, maxOperations int) {
	if inventory == nil || inventory.Truncated {
		return
	}
	inventory.Truncated = true
	inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
		Severity:    "error",
		Code:        "inventory.limit.operations",
		Message:     fmt.Sprintf("operation inventory reached the %d-operation limit", maxOperations),
		Remediation: "Narrow the source documents or explicitly increase MaxOperations for a reviewed workload.",
	})
}

func enforceInventoryOperationLimit(inventory *OperationInventory, beforeOperations, beforeDocuments, maxOperations int) bool {
	added := len(inventory.Operations) - beforeOperations
	if added < 0 {
		added = 0
	}
	inventory.VisitedOperations += added
	if len(inventory.Operations) <= maxOperations {
		return false
	}
	accepted := maxOperations - beforeOperations
	if accepted < 0 {
		accepted = 0
	}
	inventory.Operations = inventory.Operations[:maxOperations]
	if len(inventory.Documents) > beforeDocuments {
		inventory.Documents[len(inventory.Documents)-1].OperationCount = accepted
	}
	markInventoryTruncated(inventory, maxOperations)
	return true
}

func operationSummary(doc InventoryDocument, documentName, path, method string, operation map[string]any) OperationSummary {
	operationID := stringValue(operation["operationId"])
	id := operationID
	if id == "" {
		id = sanitizeName(firstNonEmpty(documentName, doc.Path, doc.URL)) + "_" + method + "_" + sanitizeName(path)
	}
	provenance := firstNonEmpty(doc.Path, doc.URL, documentName) + "#" + method + " " + path
	return OperationSummary{
		ID:                   id,
		DocumentName:         documentName,
		DocumentPath:         doc.Path,
		DocumentRelativePath: doc.RelativePath,
		DocumentURL:          doc.URL,
		OperationID:          operationID,
		Method:               strings.ToUpper(method),
		Path:                 path,
		Summary:              stringValue(operation["summary"]),
		Description:          stringValue(operation["description"]),
		Tags:                 stringSlice(operation["tags"]),
		Extensions:           operationExtensions(operation),
		Provenance:           provenance,
	}
}

func operationExtensions(operation map[string]any) map[string]string {
	allow := map[string]struct{}{
		"x-aws-operation-name":        {},
		"x-ms-long-running-operation": {},
	}
	out := map[string]string{}
	for key := range allow {
		value := operationExtensionValue(operation[key])
		if value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func operationExtensionValue(value any) string {
	if text := strings.TrimSpace(stringValue(value)); text != "" {
		return text
	}
	if typed, ok := value.(bool); ok {
		if typed {
			return "true"
		}
		return "false"
	}
	return ""
}

func operationMethods(pathItem map[string]any) []string {
	methods := make([]string, 0, 8)
	for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
		if _, ok := pathItem[method].(map[string]any); ok {
			methods = append(methods, method)
		}
	}
	return methods
}

func parameterSummaries(root map[string]any, parameters []any, op *OperationSummary) []ParameterSummary {
	out := make([]ParameterSummary, 0, len(parameters))
	for _, value := range parameters {
		parameter := mapValue(value)
		if ref := stringValue(parameter["$ref"]); strings.HasPrefix(ref, "#/") {
			if resolved := localObjectRef(root, ref); len(resolved) > 0 {
				parameter = mergeObjectRef(resolved, parameter)
			} else {
				addOperationIssue(op, "schema.ref_unresolved", "parameter reference was not resolved", ref)
			}
		}
		if len(parameter) == 0 {
			continue
		}
		schema := mapValue(parameter["schema"])
		summary := ParameterSummary{
			Name:        stringValue(parameter["name"]),
			In:          stringValue(parameter["in"]),
			Description: stringValue(parameter["description"]),
			Required:    boolValue(parameter["required"]),
			Type:        firstNonEmpty(stringValue(schema["type"]), stringValue(parameter["type"])),
			Format:      firstNonEmpty(stringValue(schema["format"]), stringValue(parameter["format"])),
			Ref:         firstNonEmpty(stringValue(parameter["$ref"]), stringValue(schema["$ref"])),
		}
		if summary.Ref != "" && op != nil {
			addOperationIssue(op, "schema.ref_unresolved", "parameter references a schema that was not resolved", summary.Ref)
		}
		out = append(out, summary)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].In != out[j].In {
			return out[i].In < out[j].In
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func operationSearchText(op OperationSummary) string {
	var parts []string
	parts = append(parts, op.OperationID, op.Summary, op.Description, op.Path, op.Method)
	parts = append(parts, op.Tags...)
	for _, parameter := range op.Parameters {
		parts = append(parts, parameter.Name, parameter.In, parameter.Description, parameter.Type, parameter.Format)
	}
	if op.RequestBody != nil && op.RequestBody.Schema != nil {
		parts = append(parts, op.RequestBody.Schema.Type, op.RequestBody.Schema.Ref)
		for _, property := range op.RequestBody.Schema.Properties {
			parts = append(parts, property.Name, property.Type, property.Description)
		}
	}
	return strings.Join(nonEmptyStrings(parts...), " ")
}

func addOperationIssue(op *OperationSummary, code, message, ref string) {
	if op == nil {
		return
	}
	issue := ReadinessIssue{
		Severity:    "warning",
		Code:        code,
		Message:     message,
		OperationID: op.OperationID,
		Path:        firstNonEmpty(ref, op.Provenance),
		Remediation: "Resolve the reference in the caller before rendering or execution.",
	}
	op.ReadinessIssues = append(op.ReadinessIssues, issue)
}
