package apitools

import (
	"encoding/json"
	"sort"
	"strings"
)

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func sliceValue(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func boolValue(value any) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return false
}

func stringSlice(value any) []string {
	if typed, ok := value.([]string); ok {
		return sortedUniqueStrings(append([]string(nil), typed...))
	}
	values := sliceValue(value)
	if len(values) == 0 {
		if text := stringValue(value); text != "" {
			return []string{text}
		}
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text := stringValue(value); text != "" {
			out = append(out, text)
		}
	}
	sort.Strings(out)
	return out
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
