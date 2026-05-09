package apitools

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func parseInventoryDocument(content []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("document is empty")
	}
	var value any
	if trimmed[0] == '{' {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
	} else if err := yaml.Unmarshal(trimmed, &value); err != nil {
		return nil, err
	}
	root, ok := normalizeInventoryValue(value).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("document root must be an object")
	}
	if stringValue(root["openapi"]) == "" && stringValue(root["swagger"]) == "" {
		return nil, fmt.Errorf("document does not declare openapi or swagger")
	}
	return root, nil
}

func normalizeInventoryValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = normalizeInventoryValue(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[fmt.Sprint(key)] = normalizeInventoryValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = normalizeInventoryValue(child)
		}
		return out
	default:
		return value
	}
}
