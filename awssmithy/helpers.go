package awssmithy

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var identifierRE = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func primitiveSchema(typ string) map[string]any {
	switch typ {
	case "blob":
		return map[string]any{"type": "string", "format": "binary"}
	case "boolean":
		return map[string]any{"type": "boolean"}
	case "byte", "short", "integer", "intEnum":
		return map[string]any{"type": "integer", "format": "int32"}
	case "long", "bigInteger":
		return map[string]any{"type": "integer", "format": "int64"}
	case "float":
		return map[string]any{"type": "number", "format": "float"}
	case "double", "bigDecimal":
		return map[string]any{"type": "number", "format": "double"}
	case "string":
		return map[string]any{"type": "string"}
	case "timestamp":
		return map[string]any{"type": "string", "format": "date-time"}
	case "document":
		return map[string]any{"type": "object", "additionalProperties": true}
	default:
		return nil
	}
}

func preludeSchema(id string) map[string]any {
	if !strings.HasPrefix(id, "smithy.api#") {
		return nil
	}
	switch strings.TrimPrefix(id, "smithy.api#") {
	case "Unit":
		return map[string]any{"type": "object", "additionalProperties": false}
	case "Blob", "PrimitiveBlob":
		return primitiveSchema("blob")
	case "Boolean", "PrimitiveBoolean":
		return primitiveSchema("boolean")
	case "Byte", "PrimitiveByte":
		return primitiveSchema("byte")
	case "Short", "PrimitiveShort":
		return primitiveSchema("short")
	case "Integer", "PrimitiveInteger":
		return primitiveSchema("integer")
	case "Long", "PrimitiveLong":
		return primitiveSchema("long")
	case "Float", "PrimitiveFloat":
		return primitiveSchema("float")
	case "Double", "PrimitiveDouble":
		return primitiveSchema("double")
	case "BigInteger":
		return primitiveSchema("bigInteger")
	case "BigDecimal":
		return primitiveSchema("bigDecimal")
	case "String":
		return primitiveSchema("string")
	case "Timestamp":
		return primitiveSchema("timestamp")
	case "Document":
		return primitiveSchema("document")
	default:
		return nil
	}
}

func splitSmithyURI(uri string) (string, map[string]string, []string) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		uri = "/"
	}
	path := uri
	query := ""
	if idx := strings.Index(uri, "?"); idx >= 0 {
		path = uri[:idx]
		query = uri[idx+1:]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var greedy []string
	path = regexp.MustCompile(`\{([^}/{}+]+)\+\}`).ReplaceAllStringFunc(path, func(label string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(label, "{"), "+}")
		greedy = append(greedy, name)
		return "{" + name + "}"
	})
	literals := map[string]string{}
	for _, part := range strings.Split(query, "&") {
		if part == "" {
			continue
		}
		name, value, _ := strings.Cut(part, "=")
		if name != "" {
			literals[name] = value
		}
	}
	if len(literals) == 0 {
		literals = nil
	}
	return path, literals, greedy
}

func operationPathKey(path string, queryLiterals map[string]string) string {
	if len(queryLiterals) == 0 {
		return path
	}
	parts := make([]string, 0, len(queryLiterals))
	for _, name := range sortedKeysStringMap(queryLiterals) {
		if queryLiterals[name] == "" {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, name+"="+queryLiterals[name])
	}
	return path + "?" + strings.Join(parts, "&")
}

func pathLabels(path string) []string {
	matches := regexp.MustCompile(`\{([^{}]+)\}`).FindAllStringSubmatch(path, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			out = append(out, match[1])
		}
	}
	return out
}

func sortedKeysStringMap(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func enumValues(shape map[string]any) []any {
	if values := sliceValueOrNil(traits(shape)["smithy.api#enum"]); len(values) > 0 {
		out := make([]any, 0, len(values))
		for _, value := range values {
			item := mapValueOrNil(value)
			if enumValue := stringValue(item["value"]); enumValue != "" {
				out = append(out, enumValue)
			}
		}
		return out
	}
	if stringValue(shape["type"]) != "enum" && stringValue(shape["type"]) != "intEnum" {
		return nil
	}
	members := mapValueOrNil(shape["members"])
	out := make([]any, 0, len(members))
	for _, name := range sortedKeys(members) {
		member := mapValueOrNil(members[name])
		if value, ok := traits(member)["smithy.api#enumValue"]; ok {
			out = append(out, value)
			continue
		}
		out = append(out, name)
	}
	return out
}

func traits(shape map[string]any) map[string]any {
	return mapValueOrNil(shape["traits"])
}

func hasTrait(shape map[string]any, name string) bool {
	_, ok := traits(shape)[name]
	return ok
}

func smithyDocumentation(shape map[string]any) string {
	return cleanDocumentation(stringValue(traits(shape)["smithy.api#documentation"]))
}

func cleanDocumentation(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func summaryFromDocumentation(s string) string {
	if s == "" {
		return ""
	}
	for _, sep := range []string{". ", "? ", "! "} {
		if idx := strings.Index(s, sep); idx > 0 {
			return strings.TrimSpace(s[:idx+1])
		}
	}
	return s
}

func requiredMapField(root map[string]any, field, context string) (map[string]any, bool, error) {
	value, ok := root[field]
	if !ok {
		return nil, false, nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("%s field %q must be an object", context, field)
	}
	return m, true, nil
}

func mapValueOrNil(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	default:
		return nil
	}
}

func sliceValueOrNil(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	default:
		return nil
	}
}

func stringSlice(v any) []string {
	items := sliceValueOrNil(v)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if value := stringValue(item); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func stringValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func intValue(v any, fallback int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return fallback
	}
}

func targetValue(v any) string {
	m := mapValueOrNil(v)
	return stringValue(m["target"])
}

func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func localName(id string) string {
	if idx := strings.LastIndex(id, "#"); idx >= 0 {
		id = id[idx+1:]
	}
	if idx := strings.LastIndex(id, "$"); idx >= 0 {
		id = id[idx+1:]
	}
	return id
}

func sanitizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = identifierRE.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return ""
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
