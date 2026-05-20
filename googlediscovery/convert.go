package googlediscovery

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const openAPIVersion = "3.0.1"

// Convert converts a Google Discovery document into a bounded OpenAPI 3.0.1
// document that is sufficient for the HCL generator.
func Convert(data []byte) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse discovery document: %w", err)
	}

	doc, err := ConvertMap(raw)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(doc, "", "  ")
}

// ConvertMap converts a Discovery document already decoded into a generic map.
func ConvertMap(raw map[string]any) (map[string]any, error) {
	c := &converter{
		raw:          raw,
		schemas:      map[string]map[string]any{},
		inflight:     map[string]bool{},
		oauth2Scopes: discoveryOAuthScopes(raw),
	}
	return c.convert()
}

type converter struct {
	raw          map[string]any
	schemas      map[string]map[string]any
	inflight     map[string]bool
	oauth2Scopes map[string]any
}

func (c *converter) convert() (map[string]any, error) {
	title := stringValue(c.raw["title"])
	if title == "" {
		title = stringValue(c.raw["name"])
	}
	version := stringValue(c.raw["version"])
	description := stringValue(c.raw["description"])

	serverURL, pathPrefix := discoveryServerAndPathPrefix(c.raw)

	components, err := c.convertComponents()
	if err != nil {
		return nil, err
	}
	if securitySchemes := c.convertSecuritySchemes(); len(securitySchemes) > 0 {
		if components == nil {
			components = map[string]any{}
		}
		components["securitySchemes"] = securitySchemes
	}
	paths, err := c.convertPaths(pathPrefix)
	if err != nil {
		return nil, err
	}

	info := map[string]any{
		"title":   title,
		"version": version,
	}
	out := map[string]any{
		"openapi": openAPIVersion,
		"info":    info,
		"servers": []any{
			map[string]any{"url": serverURL},
		},
		"paths": paths,
	}
	if description != "" {
		info["description"] = description
	}
	if len(components) > 0 {
		out["components"] = components
	}
	return out, nil
}

func discoveryServerAndPathPrefix(raw map[string]any) (serverURL, pathPrefix string) {
	rootURL := trimTrailingSlash(stringValue(raw["rootUrl"]))
	servicePath := trimSlashes(stringValue(raw["servicePath"]))
	baseURL := trimTrailingSlash(stringValue(raw["baseUrl"]))

	if rootURL != "" {
		if servicePath != "" {
			pathPrefix = "/" + servicePath
		}
		return rootURL, pathPrefix
	}
	if baseURL != "" {
		return baseURL, ""
	}
	return "https://www.googleapis.com", ""
}

func (c *converter) convertComponents() (map[string]any, error) {
	schemas, ok, err := optionalMapField(c.raw, "schemas", "discovery document")
	if err != nil {
		return nil, err
	}
	if !ok || len(schemas) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	converted := make(map[string]any, len(names))
	for _, name := range names {
		schema, ok, err := mapValueRequired(schemas[name], fmt.Sprintf("schemas.%s", name))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		convertedSchema, err := c.convertSchema(name, schema)
		if err != nil {
			return nil, err
		}
		converted[name] = convertedSchema
	}
	return map[string]any{"schemas": converted}, nil
}

func (c *converter) convertSecuritySchemes() map[string]any {
	if !c.hasGoogleOAuth2SecurityScheme() {
		return nil
	}
	return map[string]any{
		"google_oauth2": map[string]any{
			"type":        "oauth2",
			"description": "Google OAuth 2.0",
			"flows": map[string]any{
				"authorizationCode": map[string]any{
					"authorizationUrl": "https://accounts.google.com/o/oauth2/v2/auth",
					"tokenUrl":         "https://oauth2.googleapis.com/token",
					"scopes":           c.oauth2Scopes,
				},
			},
		},
	}
}

func (c *converter) hasGoogleOAuth2SecurityScheme() bool {
	return len(c.oauth2Scopes) > 0
}

func discoveryOAuthScopes(raw map[string]any) map[string]any {
	auth, ok := mapValue(raw["auth"])
	if !ok {
		return nil
	}
	oauth2, ok := mapValue(auth["oauth2"])
	if !ok {
		return nil
	}
	scopes, ok := mapValue(oauth2["scopes"])
	if !ok || len(scopes) == 0 {
		return nil
	}
	out := make(map[string]any, len(scopes))
	for _, name := range sortedKeys(scopes) {
		description := ""
		if scope, ok := mapValue(scopes[name]); ok {
			description = stringValue(scope["description"])
		}
		out[name] = description
	}
	return out
}

func (c *converter) convertPaths(pathPrefix string) (map[string]any, error) {
	paths := map[string]any{}

	if methods, ok, err := optionalMapField(c.raw, "methods", "discovery document"); err != nil {
		return nil, err
	} else if ok {
		if err := c.addMethods(paths, pathPrefix, nil, methods, nil); err != nil {
			return nil, err
		}
	}
	if resources, ok, err := optionalMapField(c.raw, "resources", "discovery document"); err != nil {
		return nil, err
	} else if ok {
		if err := c.addResources(paths, pathPrefix, nil, resources, nil); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

func (c *converter) addResources(paths map[string]any, pathPrefix string, ancestorParams []map[string]any, resources map[string]any, tags []string) error {
	names := sortedKeys(resources)
	for _, name := range names {
		resource, ok, err := mapValueRequired(resources[name], fmt.Sprintf("resources.%s", name))
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		var nextTags []string
		nextTags = append(nextTags, tags...)
		if name != "" {
			nextTags = append(nextTags, name)
		}

		params := append([]map[string]any(nil), ancestorParams...)
		if resParams, ok, err := optionalMapField(resource, "parameters", "resource "+name); err != nil {
			return err
		} else if ok {
			items, err := parameterListFromMap(resParams, "resource "+name+".parameters")
			if err != nil {
				return err
			}
			params = append(params, items...)
		}

		if methods, ok, err := optionalMapField(resource, "methods", "resource "+name); err != nil {
			return err
		} else if ok {
			if err := c.addMethods(paths, pathPrefix, params, methods, nextTags); err != nil {
				return err
			}
		}
		if subresources, ok, err := optionalMapField(resource, "resources", "resource "+name); err != nil {
			return err
		} else if ok {
			if err := c.addResources(paths, pathPrefix, params, subresources, nextTags); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *converter) addMethods(paths map[string]any, pathPrefix string, inheritedParams []map[string]any, methods map[string]any, tags []string) error {
	names := sortedKeys(methods)
	for _, name := range names {
		method, ok, err := mapValueRequired(methods[name], fmt.Sprintf("methods.%s", name))
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		op, path, err := c.convertMethod(pathPrefix, name, method, inheritedParams, tags)
		if err != nil {
			return err
		}
		if path == "" {
			continue
		}

		pathItem, _ := mapValue(paths[path])
		if pathItem == nil {
			pathItem = map[string]any{}
			paths[path] = pathItem
		}
		httpMethod := strings.ToLower(stringValue(method["httpMethod"]))
		if httpMethod == "" {
			return fmt.Errorf("discovery method %q missing httpMethod", stringValue(method["id"]))
		}
		pathItem[httpMethod] = op
	}
	return nil
}

func (c *converter) convertMethod(pathPrefix, methodName string, method map[string]any, inheritedParams []map[string]any, tags []string) (map[string]any, string, error) {
	opID := sanitizeIdentifier(stringValue(method["id"]))
	if opID == "" {
		opID = sanitizeIdentifier(methodName)
	}

	path := methodPath(pathPrefix, method)
	if path == "" {
		return nil, "", fmt.Errorf("discovery method %q missing path", stringValue(method["id"]))
	}

	op := map[string]any{
		"operationId": opID,
		"summary":     summarize(stringValue(method["description"]), methodName),
		"description": stringValue(method["description"]),
	}
	if len(tags) > 0 {
		op["tags"] = tags
	}
	op["x-discovery-id"] = stringValue(method["id"])

	params := append([]map[string]any(nil), inheritedParams...)
	if methodParams, ok, err := optionalMapField(method, "parameters", "method "+methodName); err != nil {
		return nil, "", err
	} else if ok {
		items, err := parameterListFromMap(methodParams, "method "+methodName+".parameters")
		if err != nil {
			return nil, "", err
		}
		params = append(params, items...)
	}
	converted, err := convertParameters(params)
	if err != nil {
		return nil, "", err
	}
	if len(converted) > 0 {
		op["parameters"] = converted
	}

	requestBody, err := c.convertRequestBody(method)
	if err != nil {
		return nil, "", err
	}
	if requestBody != nil {
		op["requestBody"] = requestBody
	}
	if security := c.methodSecurity(method); len(security) > 0 {
		op["security"] = security
	}

	responses, err := c.convertResponses(method)
	if err != nil {
		return nil, "", err
	}
	op["responses"] = responses
	return op, path, nil
}

func (c *converter) methodSecurity(method map[string]any) []any {
	if !c.hasGoogleOAuth2SecurityScheme() {
		return nil
	}
	scopes := stringSliceValue(method["scopes"])
	if len(scopes) == 0 {
		return nil
	}
	return []any{map[string]any{"google_oauth2": scopes}}
}

func methodPath(pathPrefix string, method map[string]any) string {
	if upload := mediaUploadSimplePath(method); upload != "" {
		return upload
	}
	p := strings.TrimSpace(stringValue(method["path"]))
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	p = "/" + strings.TrimPrefix(p, "/")
	if pathPrefix == "" {
		return p
	}
	return "/" + strings.Trim(pathPrefix, "/") + p
}

func mediaUploadSimplePath(method map[string]any) string {
	mediaUpload, ok := mapValue(method["mediaUpload"])
	if !ok {
		return ""
	}
	protocols, ok := mapValue(mediaUpload["protocols"])
	if !ok {
		return ""
	}
	simple, ok := mapValue(protocols["simple"])
	if !ok {
		return ""
	}
	return stringValue(simple["path"])
}

func (c *converter) convertRequestBody(method map[string]any) (map[string]any, error) {
	requestSchema, err := c.requestSchemaRef(method)
	if err != nil {
		return nil, err
	}
	if requestSchema == "" && mediaUploadSimplePath(method) == "" {
		return nil, nil
	}

	content := map[string]any{}
	if uploadPath := mediaUploadSimplePath(method); uploadPath != "" {
		content["multipart/related"] = map[string]any{
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"metadata": func() any {
						if requestSchema != "" {
							return map[string]any{"$ref": requestSchema}
						}
						return map[string]any{"type": "object"}
					}(),
					"content": map[string]any{
						"type":   "string",
						"format": "binary",
					},
				},
				"required":             []string{"content"},
				"additionalProperties": false,
			},
		}
		return map[string]any{
			"required": true,
			"content":  content,
		}, nil
	}

	if requestSchema != "" {
		content["application/json"] = map[string]any{
			"schema": map[string]any{"$ref": requestSchema},
		}
		return map[string]any{
			"required": true,
			"content":  content,
		}, nil
	}

	content["application/octet-stream"] = map[string]any{
		"schema": map[string]any{
			"type":   "string",
			"format": "binary",
		},
	}
	return map[string]any{
		"required": true,
		"content":  content,
	}, nil
}

func (c *converter) requestSchemaRef(method map[string]any) (string, error) {
	request, ok, err := optionalMapField(method, "request", "method "+stringValue(method["id"]))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	ref := stringValue(request["$ref"])
	if ref == "" {
		return "", nil
	}
	return "#/components/schemas/" + ref, nil
}

func (c *converter) convertResponses(method map[string]any) (map[string]any, error) {
	responseRef := ""
	if response, ok, err := optionalMapField(method, "response", "method "+stringValue(method["id"])); err != nil {
		return nil, err
	} else if ok {
		responseRef = stringValue(response["$ref"])
	}
	if responseRef != "" {
		return map[string]any{
			"200": map[string]any{
				"description": "Success",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"$ref": "#/components/schemas/" + responseRef,
						},
					},
				},
			},
		}, nil
	}
	return map[string]any{
		"204": map[string]any{
			"description": "No content",
		},
	}, nil
}

func (c *converter) convertSchema(name string, schema map[string]any) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}
	if cached, ok := c.schemas[name]; ok {
		return cached, nil
	}
	if c.inflight[name] {
		return map[string]any{"$ref": "#/components/schemas/" + name}, nil
	}
	c.inflight[name] = true
	out, err := convertSchemaMap(schema, "schemas."+name)
	if err != nil {
		return nil, err
	}
	c.schemas[name] = out
	c.inflight[name] = false
	return out, nil
}

func convertSchemaMap(schema map[string]any, path string) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}
	if ref := stringValue(schema["$ref"]); ref != "" {
		return map[string]any{"$ref": "#/components/schemas/" + ref}, nil
	}
	out := map[string]any{}
	for _, key := range []string{
		"type", "format", "description", "title", "default", "example",
		"pattern", "nullable", "deprecated", "readOnly", "writeOnly",
		"minimum", "maximum", "minLength", "maxLength", "minItems",
		"maxItems", "uniqueItems",
	} {
		if v, ok := schema[key]; ok {
			if normalized, ok := normalizeNumericJSON(v); ok {
				out[key] = normalized
				continue
			}
			out[key] = v
		}
	}
	if v, ok := schema["enum"]; ok {
		out["enum"] = v
	}
	if v, ok := schema["required"]; ok {
		out["required"] = v
	}
	if v, ok := schema["properties"]; ok {
		props, ok, err := mapValueRequired(v, path+".properties")
		if err != nil {
			return nil, err
		}
		if ok {
			outProps := make(map[string]any, len(props))
			for _, name := range sortedKeys(props) {
				prop, ok, err := mapValueRequired(props[name], path+".properties."+name)
				if err != nil {
					return nil, err
				}
				if ok {
					converted, err := convertSchemaMap(prop, path+".properties."+name)
					if err != nil {
						return nil, err
					}
					outProps[name] = converted
				}
			}
			if len(outProps) > 0 {
				out["properties"] = outProps
			}
		}
	}
	if v, ok := schema["items"]; ok {
		if items, ok := mapValue(v); ok {
			converted, err := convertSchemaMap(items, path+".items")
			if err != nil {
				return nil, err
			}
			out["items"] = converted
		}
	}
	if v, ok := schema["allOf"]; ok {
		if arr, ok := sliceValue(v); ok {
			outArr := make([]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := mapValue(item); ok {
					converted, err := convertSchemaMap(m, path+".allOf")
					if err != nil {
						return nil, err
					}
					outArr = append(outArr, converted)
				}
			}
			if len(outArr) > 0 {
				out["allOf"] = outArr
			}
		}
	}
	if v, ok := schema["oneOf"]; ok {
		if arr, ok := sliceValue(v); ok {
			outArr := make([]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := mapValue(item); ok {
					converted, err := convertSchemaMap(m, path+".oneOf")
					if err != nil {
						return nil, err
					}
					outArr = append(outArr, converted)
				}
			}
			if len(outArr) > 0 {
				out["oneOf"] = outArr
			}
		}
	}
	if v, ok := schema["anyOf"]; ok {
		if arr, ok := sliceValue(v); ok {
			outArr := make([]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := mapValue(item); ok {
					converted, err := convertSchemaMap(m, path+".anyOf")
					if err != nil {
						return nil, err
					}
					outArr = append(outArr, converted)
				}
			}
			if len(outArr) > 0 {
				out["anyOf"] = outArr
			}
		}
	}
	if v, ok := schema["additionalProperties"]; ok {
		switch t := v.(type) {
		case bool:
			out["additionalProperties"] = t
		case map[string]any:
			converted, err := convertSchemaMap(t, path+".additionalProperties")
			if err != nil {
				return nil, err
			}
			out["additionalProperties"] = converted
		}
	}
	return out, nil
}

func convertParameters(params []map[string]any) ([]any, error) {
	if len(params) == 0 {
		return nil, nil
	}
	type key struct {
		name string
		in   string
	}
	seen := map[key]bool{}
	out := make([]any, 0, len(params))
	for _, p := range params {
		name := sanitizeParameterName(stringValue(p["name"]))
		loc := strings.ToLower(stringValue(firstNonEmpty(p["location"], p["in"])))
		if name == "" || loc == "" {
			continue
		}
		k := key{name: name, in: loc}
		if seen[k] {
			continue
		}
		seen[k] = true
		param := map[string]any{
			"name": name,
			"in":   loc,
		}
		if req, ok := boolValue(p["required"]); ok && req {
			param["required"] = true
		}
		if desc := stringValue(p["description"]); desc != "" {
			param["description"] = desc
		}
		schema, err := convertDiscoveryParamSchema(p)
		if err != nil {
			return nil, err
		}
		if schema != nil {
			param["schema"] = schema
		}
		out = append(out, param)
	}
	return out, nil
}

func sanitizeParameterName(name string) string {
	if len(name) == 0 {
		return name
	}
	switch name[0:1] {
	case "{", "[", "<", "(", "$", "#", "@", "%", "!", "~", "`", "&", "^", "*", "+", "=", "|", ";", ":", ",", ".":
		return "x_" + name[1:]
	default:
		return name
	}
}

func convertDiscoveryParamSchema(param map[string]any) (map[string]any, error) {
	schema := map[string]any{}
	if t := stringValue(param["type"]); t != "" {
		if repeated, _ := boolValue(param["repeated"]); repeated {
			schema["type"] = "array"
			schema["items"] = map[string]any{"type": t}
		} else {
			schema["type"] = t
		}
	}
	for _, key := range []string{"format", "description", "default", "pattern", "minimum", "maximum", "minLength", "maxLength"} {
		if v, ok := param[key]; ok {
			schema[key] = v
		}
	}
	if v, ok := param["enum"]; ok {
		schema["enum"] = v
	}
	if v, ok := param["items"]; ok {
		if itemMap, ok := mapValue(v); ok {
			converted, err := convertSchemaMap(itemMap, "parameter.items")
			if err != nil {
				return nil, err
			}
			schema["items"] = converted
		}
	}
	for _, key := range []string{"minimum", "maximum", "minLength", "maxLength"} {
		if v, ok := param[key]; ok {
			if normalized, ok := normalizeNumericJSON(v); ok {
				schema[key] = normalized
				continue
			}
			schema[key] = v
		}
	}
	if len(schema) == 0 {
		return nil, nil
	}
	return schema, nil
}

func normalizeNumericJSON(v any) (any, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case json.Number:
		if s := strings.TrimSpace(t.String()); s != "" {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return n, true
			}
		}
	case string:
		if s := strings.TrimSpace(t); s != "" {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return n, true
			}
		}
	}
	return nil, false
}

func parameterListFromMap(params map[string]any, path string) ([]map[string]any, error) {
	if len(params) == 0 {
		return nil, nil
	}
	names := sortedKeys(params)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		param, ok, err := mapValueRequired(params[name], path+"."+name)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		param = cloneMap(param)
		param["name"] = name
		out = append(out, param)
	}
	return out, nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func stringValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func firstNonEmpty(values ...any) any {
	for _, v := range values {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func boolValue(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func mapValue(v any) (map[string]any, bool) {
	switch t := v.(type) {
	case map[string]any:
		return t, true
	default:
		return nil, false
	}
}

func optionalMapField(parent map[string]any, key, context string) (map[string]any, bool, error) {
	if parent == nil {
		return nil, false, nil
	}
	value, exists := parent[key]
	if !exists || value == nil {
		return nil, false, nil
	}
	m, ok := mapValue(value)
	if !ok {
		return nil, false, fmt.Errorf("%s field %q must be an object", context, key)
	}
	return m, true, nil
}

func mapValueRequired(value any, context string) (map[string]any, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	m, ok := mapValue(value)
	if !ok {
		return nil, false, fmt.Errorf("%s must be an object", context)
	}
	return m, true, nil
}

func sliceValue(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	default:
		return nil, false
	}
}

func stringSliceValue(v any) []string {
	if items, ok := v.([]string); ok {
		return append([]string(nil), items...)
	}
	items, ok := sliceValue(v)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := stringValue(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func trimTrailingSlash(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

func trimSlashes(s string) string {
	return strings.Trim(strings.TrimSpace(s), "/")
}

func summarize(description, fallback string) string {
	if description == "" {
		return fallback
	}
	s := strings.TrimSpace(description)
	for _, sep := range []string{"\n", ". "} {
		if idx := strings.Index(s, sep); idx > 0 {
			return strings.TrimSpace(s[:idx+1])
		}
	}
	return s
}

var nonID = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = nonID.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return ""
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "op_" + s
	}
	return strings.ToLower(s)
}
