package apitools

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const DefaultMaxSchemaResolutionWork = 4_096

func requestBodySummary(ctx context.Context, root map[string]any, operation map[string]any, op *OperationSummary) (*RequestBodySummary, error) {
	if body := mapValue(operation["requestBody"]); len(body) > 0 {
		if ref := stringValue(body["$ref"]); strings.HasPrefix(ref, "#/") {
			if resolved := localObjectRef(root, ref); len(resolved) > 0 {
				body = mergeObjectRef(resolved, body)
			} else {
				addOperationIssue(op, "schema.ref_unresolved", "requestBody reference was not resolved", ref)
			}
		}
		summary := &RequestBodySummary{
			Description: stringValue(body["description"]),
			Required:    boolValue(body["required"]),
			Ref:         stringValue(body["$ref"]),
		}
		if summary.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "requestBody references a schema that was not resolved", summary.Ref)
		}
		content := mapValue(body["content"])
		summary.ContentTypes = sortedMapKeys(content)
		if len(summary.ContentTypes) > 0 {
			media := mapValue(content[summary.ContentTypes[0]])
			rawSchema, err := resolveLocalSchemaRefs(ctx, root, mapValue(media["schema"]), op)
			if err != nil {
				return nil, err
			}
			schema := schemaSummary(rawSchema)
			summary.Schema = &schema
			summary.Fields = requestFieldSummaries(rawSchema, "", summary.Required, 0)
			if len(summary.Fields) == 0 && len(rawSchema) > 0 && !looksLikeCredentialName("body") {
				summary.Fields = []RequestFieldSummary{requestFieldSummary("body", summary.Required, rawSchema)}
			}
			summary.RequiredFieldPaths = requiredRequestFieldPaths(summary.Fields)
			if schema.Ref != "" {
				addOperationIssue(op, "schema.ref_unresolved", "request body schema reference was not resolved", schema.Ref)
			}
		}
		return summary, nil
	}
	for _, parameterValue := range sliceValue(operation["parameters"]) {
		parameter := mapValue(parameterValue)
		if ref := stringValue(parameter["$ref"]); strings.HasPrefix(ref, "#/") {
			if resolved := localObjectRef(root, ref); len(resolved) > 0 {
				parameter = mergeObjectRef(resolved, parameter)
			} else {
				addOperationIssue(op, "schema.ref_unresolved", "body parameter reference was not resolved", ref)
			}
		}
		if stringValue(parameter["in"]) != "body" {
			continue
		}
		rawSchema, err := resolveLocalSchemaRefs(ctx, root, mapValue(parameter["schema"]), op)
		if err != nil {
			return nil, err
		}
		schema := schemaSummary(rawSchema)
		if schema.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "body parameter schema reference was not resolved", schema.Ref)
		}
		fields := requestFieldSummaries(rawSchema, "", boolValue(parameter["required"]), 0)
		if len(fields) == 0 && len(rawSchema) > 0 && !looksLikeCredentialName("body") {
			fields = []RequestFieldSummary{requestFieldSummary("body", boolValue(parameter["required"]), rawSchema)}
		}
		return &RequestBodySummary{
			Description:        stringValue(parameter["description"]),
			Required:           boolValue(parameter["required"]),
			Schema:             &schema,
			Ref:                stringValue(parameter["$ref"]),
			Fields:             fields,
			RequiredFieldPaths: requiredRequestFieldPaths(fields),
		}, nil
	}
	return nil, nil
}

func responseBodySummary(ctx context.Context, root map[string]any, operation map[string]any, op *OperationSummary) (*ResponseBodySummary, error) {
	responses := mapValue(operation["responses"])
	if len(responses) == 0 {
		return nil, nil
	}
	for _, status := range successfulResponseStatuses(responses) {
		response := mapValue(responses[status])
		if len(response) == 0 {
			continue
		}
		if ref := stringValue(response["$ref"]); strings.HasPrefix(ref, "#/") {
			if resolved := localObjectRef(root, ref); len(resolved) > 0 {
				response = mergeObjectRef(resolved, response)
			} else {
				addOperationIssue(op, "schema.ref_unresolved", "response reference was not resolved", ref)
			}
		}
		summary := &ResponseBodySummary{
			StatusCode:  status,
			Description: stringValue(response["description"]),
			Ref:         stringValue(response["$ref"]),
		}
		if summary.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "response references a schema that was not resolved", summary.Ref)
		}
		if content := mapValue(response["content"]); len(content) > 0 {
			summary.ContentTypes = sortedMapKeys(content)
			for _, contentType := range preferredResponseContentTypes(summary.ContentTypes) {
				media := mapValue(content[contentType])
				rawSchema, err := resolveLocalSchemaRefs(ctx, root, mapValue(media["schema"]), op)
				if err != nil {
					return nil, err
				}
				if len(rawSchema) == 0 {
					continue
				}
				schema := schemaSummary(rawSchema)
				summary.Schema = &schema
				summary.Fields = requestFieldSummaries(rawSchema, "", false, 0)
				summary.Fields = appendMissingTopLevelResponseFields(summary.Fields, rawSchema, "name", "id")
				if len(summary.Fields) == 0 && !looksLikeCredentialName("body") {
					summary.Fields = []RequestFieldSummary{requestFieldSummary("body", false, rawSchema)}
				}
				if schema.Ref != "" {
					addOperationIssue(op, "schema.ref_unresolved", "response schema reference was not resolved", schema.Ref)
				}
				return summary, nil
			}
			if len(summary.ContentTypes) > 0 {
				return summary, nil
			}
			continue
		}
		rawSchema, err := resolveLocalSchemaRefs(ctx, root, mapValue(response["schema"]), op)
		if err != nil {
			return nil, err
		}
		if len(rawSchema) == 0 {
			continue
		}
		schema := schemaSummary(rawSchema)
		summary.Schema = &schema
		summary.Fields = requestFieldSummaries(rawSchema, "", false, 0)
		summary.Fields = appendMissingTopLevelResponseFields(summary.Fields, rawSchema, "name", "id")
		if len(summary.Fields) == 0 && !looksLikeCredentialName("body") {
			summary.Fields = []RequestFieldSummary{requestFieldSummary("body", false, rawSchema)}
		}
		if schema.Ref != "" {
			addOperationIssue(op, "schema.ref_unresolved", "response schema reference was not resolved", schema.Ref)
		}
		return summary, nil
	}
	return nil, nil
}

func appendMissingTopLevelResponseFields(fields []RequestFieldSummary, schema map[string]any, names ...string) []RequestFieldSummary {
	properties := mapValue(schema["properties"])
	if len(properties) == 0 {
		return fields
	}
	seen := map[string]bool{}
	for _, field := range fields {
		seen[field.Path] = true
	}
	for _, name := range names {
		if seen[name] {
			continue
		}
		property := mapValue(properties[name])
		if len(property) == 0 {
			continue
		}
		fields = append(fields, requestFieldSummary(name, false, property))
		seen[name] = true
	}
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})
	return fields
}

func successfulResponseStatuses(responses map[string]any) []string {
	var statuses []string
	for _, status := range sortedMapKeys(responses) {
		trimmed := strings.TrimSpace(status)
		if len(trimmed) == 3 && trimmed[0] == '2' {
			statuses = append(statuses, trimmed)
		}
	}
	return statuses
}

func preferredResponseContentTypes(contentTypes []string) []string {
	out := append([]string(nil), contentTypes...)
	sort.SliceStable(out, func(i, j int) bool {
		return responseContentTypeScore(out[i]) > responseContentTypeScore(out[j])
	})
	return out
}

func responseContentTypeScore(contentType string) int {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(contentType, "json"):
		return 3
	case strings.Contains(contentType, "yaml") || strings.Contains(contentType, "xml"):
		return 2
	default:
		return 1
	}
}

type schemaResolutionState struct {
	ctx            context.Context
	root           map[string]any
	op             *OperationSummary
	activeRefs     map[string]bool
	work           int
	limitReported  bool
	branchReported bool
	depthReported  bool
}

func resolveLocalSchemaRefs(ctx context.Context, root map[string]any, schema map[string]any, op *OperationSummary) (map[string]any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state := &schemaResolutionState{ctx: ctx, root: root, op: op, activeRefs: map[string]bool{}}
	return state.resolve(schema, 0)
}

func (state *schemaResolutionState) resolve(schema map[string]any, depth int) (map[string]any, error) {
	if err := state.ctx.Err(); err != nil {
		return nil, err
	}
	if len(schema) == 0 {
		return schema, nil
	}
	if depth > maxRequestFieldDepth {
		if !state.depthReported {
			state.depthReported = true
			addOperationIssue(state.op, "schema.resolution_depth", fmt.Sprintf("schema resolution exceeded depth %d", maxRequestFieldDepth), "")
		}
		return copyStringAnyMap(schema), nil
	}
	state.work++
	if state.work > DefaultMaxSchemaResolutionWork {
		if !state.limitReported {
			state.limitReported = true
			addOperationIssue(state.op, "schema.resolution_limit", fmt.Sprintf("schema resolution exceeded %d visited nodes", DefaultMaxSchemaResolutionWork), "")
		}
		return copyStringAnyMap(schema), nil
	}

	out := copyStringAnyMap(schema)
	if ref := stringValue(out["$ref"]); strings.HasPrefix(ref, "#/") {
		if state.activeRefs[ref] {
			addOperationIssue(state.op, "schema.ref_cycle", "schema reference cycle was not expanded", ref)
			return out, nil
		}
		if resolved := localObjectRef(state.root, ref); len(resolved) > 0 {
			state.activeRefs[ref] = true
			base, err := state.resolve(resolved, depth+1)
			delete(state.activeRefs, ref)
			if err != nil {
				return nil, err
			}
			out = mergeObjectRef(base, out)
		} else {
			addOperationIssue(state.op, "schema.ref_unresolved", "schema reference was not resolved", ref)
		}
	}

	if branches := sliceValue(out["allOf"]); len(branches) > 0 {
		delete(out, "allOf")
		for _, branchValue := range branches {
			branch, err := state.resolve(mapValue(branchValue), depth+1)
			if err != nil {
				return nil, err
			}
			out = mergeSchema(out, branch, true)
		}
	}
	for _, keyword := range []string{"oneOf", "anyOf"} {
		branches := sliceValue(out[keyword])
		if len(branches) == 0 {
			continue
		}
		if !state.branchReported {
			state.branchReported = true
			addOperationIssue(state.op, "schema.branch_selection_required", keyword+" requires an explicit schema branch selection", "")
		}
		delete(out, keyword)
		for _, branchValue := range branches {
			branch, err := state.resolve(mapValue(branchValue), depth+1)
			if err != nil {
				return nil, err
			}
			out = mergeSchema(out, branch, false)
		}
	}

	if properties := mapValue(out["properties"]); len(properties) > 0 {
		resolvedProperties := make(map[string]any, len(properties))
		for _, name := range sortedMapKeys(properties) {
			resolved, err := state.resolve(mapValue(properties[name]), depth+1)
			if err != nil {
				return nil, err
			}
			resolvedProperties[name] = resolved
		}
		out["properties"] = resolvedProperties
	}
	if items := mapValue(out["items"]); len(items) > 0 {
		resolved, err := state.resolve(items, depth+1)
		if err != nil {
			return nil, err
		}
		out["items"] = resolved
	}
	return out, nil
}

func localObjectRef(root map[string]any, ref string) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var current any = root
	for _, encoded := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		segment, err := url.PathUnescape(encoded)
		if err != nil {
			return nil
		}
		key := strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = object[key]
		if !ok {
			return nil
		}
	}
	return mapValue(current)
}

func mergeObjectRef(base, overlay map[string]any) map[string]any {
	out := copyStringAnyMap(base)
	for key, value := range overlay {
		if key != "$ref" {
			out[key] = value
		}
	}
	return out
}

func mergeSchema(base, addition map[string]any, includeRequired bool) map[string]any {
	out := copyStringAnyMap(base)
	for key, value := range addition {
		switch key {
		case "properties":
			properties := copyStringAnyMap(mapValue(out[key]))
			for name, property := range mapValue(value) {
				properties[name] = property
			}
			out[key] = properties
		case "required":
			if includeRequired {
				out[key] = sortedUniqueStrings(append(stringSlice(out[key]), stringSlice(value)...))
			}
		default:
			if _, exists := out[key]; !exists {
				out[key] = value
			}
		}
	}
	return out
}

func copyStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func schemaSummary(schema map[string]any) SchemaSummary {
	if len(schema) == 0 {
		return SchemaSummary{}
	}
	required := stringSlice(schema["required"])
	requiredSet := make(map[string]bool, len(required))
	for _, name := range required {
		requiredSet[name] = true
	}
	summary := SchemaSummary{
		Type:        schemaType(schema["type"]),
		Format:      stringValue(schema["format"]),
		Ref:         stringValue(schema["$ref"]),
		Description: stringValue(schema["description"]),
		Required:    required,
	}
	properties := mapValue(schema["properties"])
	for _, name := range sortedMapKeys(properties) {
		property := schemaSummary(mapValue(properties[name]))
		summary.Properties = append(summary.Properties, PropertySummary{
			Name:        name,
			Type:        property.Type,
			Format:      property.Format,
			Ref:         property.Ref,
			Description: property.Description,
			Required:    requiredSet[name],
		})
	}
	return summary
}

func schemaType(value any) string {
	if text := stringValue(value); text != "" {
		return text
	}
	values := stringSlice(value)
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "|")
}

const (
	maxRequestFieldDepth = 6
	maxRequestFields     = 60
)

func requestFieldSummaries(schema map[string]any, path string, required bool, depth int) []RequestFieldSummary {
	var out []RequestFieldSummary
	collectRequestFields(schema, path, required, depth, &out)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	if len(out) > maxRequestFields {
		out = out[:maxRequestFields]
	}
	return out
}

func collectRequestFields(schema map[string]any, path string, required bool, depth int, out *[]RequestFieldSummary) {
	if len(*out) >= maxRequestFields || depth > maxRequestFieldDepth {
		return
	}
	if len(schema) == 0 {
		if path != "" && !looksLikeCredentialName(path) {
			*out = append(*out, RequestFieldSummary{Path: path, Required: required})
		}
		return
	}
	if path != "" && !looksLikeCredentialName(path) {
		*out = append(*out, requestFieldSummary(path, required, schema))
		if len(*out) >= maxRequestFields || depth == maxRequestFieldDepth {
			return
		}
	}
	properties := mapValue(schema["properties"])
	if len(properties) > 0 {
		requiredSet := make(map[string]bool)
		for _, name := range stringSlice(schema["required"]) {
			requiredSet[name] = true
		}
		for _, name := range sortedMapKeys(properties) {
			childPath := name
			if path != "" {
				childPath = path + "." + name
			}
			collectRequestFields(mapValue(properties[name]), childPath, required && requiredSet[name], depth+1, out)
			if len(*out) >= maxRequestFields {
				return
			}
		}
		return
	}
	if items := mapValue(schema["items"]); len(items) > 0 {
		itemPath := "body[]"
		if path != "" {
			itemPath = path + "[]"
		}
		collectRequestFields(items, itemPath, required, depth+1, out)
	}
}

func requestFieldSummary(path string, required bool, schema map[string]any) RequestFieldSummary {
	return RequestFieldSummary{
		Path:        path,
		Required:    required,
		Type:        schemaType(schema["type"]),
		Format:      stringValue(schema["format"]),
		Ref:         stringValue(schema["$ref"]),
		Description: stringValue(schema["description"]),
	}
}

func requiredRequestFieldPaths(fields []RequestFieldSummary) []string {
	var out []string
	for _, field := range fields {
		if field.Required {
			out = append(out, field.Path)
		}
	}
	sort.Strings(out)
	return out
}
