package apitools

import (
	"sort"
	"strings"
)

func requestBodySummary(operation map[string]any, op *OperationSummary) *RequestBodySummary {
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
			rawSchema := mapValue(media["schema"])
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
		rawSchema := mapValue(parameter["schema"])
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
