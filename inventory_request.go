package apitools

import (
	"sort"
	"strings"
)

func requestBodySummary(root map[string]any, operation map[string]any, op *OperationSummary) *RequestBodySummary {
	if body := mapValue(operation["requestBody"]); len(body) > 0 {
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
			rawSchema := resolveLocalSchemaRefs(root, mapValue(media["schema"]), map[string]bool{}, 0)
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
		return summary
	}
	for _, parameterValue := range sliceValue(operation["parameters"]) {
		parameter := mapValue(parameterValue)
		if stringValue(parameter["in"]) != "body" {
			continue
		}
		rawSchema := resolveLocalSchemaRefs(root, mapValue(parameter["schema"]), map[string]bool{}, 0)
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
		}
	}
	return nil
}

func responseBodySummary(root map[string]any, operation map[string]any, op *OperationSummary) *ResponseBodySummary {
	responses := mapValue(operation["responses"])
	if len(responses) == 0 {
		return nil
	}
	for _, status := range successfulResponseStatuses(responses) {
		response := mapValue(responses[status])
		if len(response) == 0 {
			continue
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
				rawSchema := resolveLocalSchemaRefs(root, mapValue(media["schema"]), map[string]bool{}, 0)
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
				return summary
			}
			if len(summary.ContentTypes) > 0 {
				return summary
			}
			continue
		}
		rawSchema := resolveLocalSchemaRefs(root, mapValue(response["schema"]), map[string]bool{}, 0)
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
		return summary
	}
	return nil
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

func resolveLocalSchemaRefs(root map[string]any, schema map[string]any, seen map[string]bool, depth int) map[string]any {
	if len(schema) == 0 || depth > maxRequestFieldDepth {
		return schema
	}
	if ref := stringValue(schema["$ref"]); strings.HasPrefix(ref, "#/definitions/") || strings.HasPrefix(ref, "#/components/schemas/") {
		if !seen[ref] {
			nextSeen := copyStringBoolMap(seen)
			nextSeen[ref] = true
			if resolved := localSchemaRef(root, ref); len(resolved) > 0 {
				base := resolveLocalSchemaRefs(root, resolved, nextSeen, depth+1)
				merged := copyStringAnyMap(base)
				for key, value := range schema {
					if key == "$ref" {
						continue
					}
					merged[key] = value
				}
				schema = merged
			}
		}
	}
	out := copyStringAnyMap(schema)
	if properties := mapValue(out["properties"]); len(properties) > 0 {
		resolvedProperties := make(map[string]any, len(properties))
		for _, name := range sortedMapKeys(properties) {
			resolvedProperties[name] = resolveLocalSchemaRefs(root, mapValue(properties[name]), copyStringBoolMap(seen), depth+1)
		}
		out["properties"] = resolvedProperties
	}
	if items := mapValue(out["items"]); len(items) > 0 {
		out["items"] = resolveLocalSchemaRefs(root, items, copyStringBoolMap(seen), depth+1)
	}
	return out
}

func localSchemaRef(root map[string]any, ref string) map[string]any {
	switch {
	case strings.HasPrefix(ref, "#/definitions/"):
		name := strings.TrimPrefix(ref, "#/definitions/")
		return mapValue(mapValue(root["definitions"])[name])
	case strings.HasPrefix(ref, "#/components/schemas/"):
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		return mapValue(mapValue(mapValue(root["components"])["schemas"])[name])
	default:
		return nil
	}
}

func copyStringAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyStringBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
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
