package apitools

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
	"gopkg.in/yaml.v3"
)

const highLevelCommonSchemasJSON = `{
  "BadRequestDTO": {
    "type": "object",
    "properties": {
      "statusCode": {"type": "number", "example": 400},
      "message": {"type": "string", "example": "Bad Request"}
    }
  },
  "UnauthorizedDTO": {
    "type": "object",
    "properties": {
      "statusCode": {"type": "number", "example": 401},
      "message": {"type": "string", "example": "Invalid token: access token is invalid"},
      "error": {"type": "string", "example": "Unauthorized"}
    }
  },
  "UnprocessableDTO": {
    "type": "object",
    "properties": {
      "statusCode": {"type": "number", "example": 422},
      "message": {"example": ["Unprocessable Entity"], "type": "array", "items": {"type": "string"}},
      "error": {"type": "string", "example": "Unprocessable Entity"}
    }
  }
}`

func correctedCatalogRefreshContent(ref catalog.RefreshableSpecReference, content []byte) ([]byte, []string) {
	root, ok := catalogRefreshContentRoot(content)
	if !ok {
		return content, nil
	}
	var notes []string
	switch {
	case isHighLevelRefreshCorrection(ref):
		if applyHighLevelRefreshCorrection(root) {
			notes = append(notes, "Applied reviewed HighLevel refresh correction: bundled official common error schemas, rewrote relative common-schema refs, filled empty response descriptions, and added placeholder items for incomplete array schemas.")
		}
	case ref.ProviderID == "strava" && ref.SpecRefID == "strava-api-v3-swagger":
		if applyStravaRefreshCorrection(root) {
			notes = append(notes, "Applied reviewed Strava refresh correction: replaced official external Swagger model refs with permissive local object placeholders and removed the undeclared root OAuth scope.")
		}
	case ref.ProviderID == "spotify" && ref.SpecRefID == "spotify-web-api-openapi":
		if applySpotifyRefreshCorrection(root) {
			notes = append(notes, "Applied reviewed Spotify refresh correction: removed an external policy extension ref, dropped schema-level boolean required flags, and pruned required entries that are not declared properties.")
		}
	default:
		return content, nil
	}
	if len(notes) == 0 {
		return content, nil
	}
	normalized, err := json.Marshal(root)
	if err != nil {
		return content, nil
	}
	return normalized, notes
}

func catalogRefreshContentRoot(content []byte) (map[string]any, bool) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return nil, false
	}
	var root map[string]any
	var err error
	if trimmed[0] == '{' {
		err = json.Unmarshal(trimmed, &root)
	} else {
		err = yaml.Unmarshal(trimmed, &root)
	}
	return root, err == nil && len(root) > 0
}

func isHighLevelRefreshCorrection(ref catalog.RefreshableSpecReference) bool {
	if ref.ProviderID != "highlevel" {
		return false
	}
	switch ref.SpecRefID {
	case "highlevel-calendars-openapi",
		"highlevel-contacts-openapi",
		"highlevel-oauth-openapi",
		"highlevel-opportunities-openapi",
		"highlevel-users-openapi":
		return true
	default:
		return false
	}
}

func applyHighLevelRefreshCorrection(root map[string]any) bool {
	changed := false
	schemas := ensureCatalogRefreshObject(root, "components", "schemas")
	var common map[string]any
	if err := json.Unmarshal([]byte(highLevelCommonSchemasJSON), &common); err == nil {
		for name, schema := range common {
			if _, exists := schemas[name]; !exists {
				schemas[name] = schema
				changed = true
			}
		}
	}
	walkCatalogRefreshObjects(root, func(obj map[string]any) {
		if ref := catalogRefreshString(obj["$ref"]); strings.HasPrefix(ref, "../common/common-schemas.json#/components/schemas/") {
			obj["$ref"] = "#/components/schemas/" + strings.TrimPrefix(ref, "../common/common-schemas.json#/components/schemas/")
			changed = true
		}
		if description, ok := obj["description"].(string); ok && strings.TrimSpace(description) == "" {
			obj["description"] = "Successful response"
			changed = true
		}
		if catalogRefreshString(obj["type"]) == "array" && obj["items"] == nil {
			obj["items"] = map[string]any{}
			changed = true
		}
	})
	return changed
}

func applyStravaRefreshCorrection(root map[string]any) bool {
	changed := false
	walkCatalogRefreshObjects(root, func(obj map[string]any) {
		if ref := catalogRefreshString(obj["$ref"]); ref != "" && isExternalRef(ref) {
			delete(obj, "$ref")
			obj["type"] = "object"
			obj["additionalProperties"] = true
			changed = true
		}
	})
	if security, ok := root["security"].([]any); ok {
		for _, requirement := range security {
			req, ok := requirement.(map[string]any)
			if !ok {
				continue
			}
			for scheme, rawScopes := range req {
				scopes, ok := rawScopes.([]any)
				if !ok {
					continue
				}
				filtered := make([]any, 0, len(scopes))
				for _, scope := range scopes {
					if catalogRefreshString(scope) == "public" {
						changed = true
						continue
					}
					filtered = append(filtered, scope)
				}
				req[scheme] = filtered
			}
		}
	}
	return changed
}

func applySpotifyRefreshCorrection(root map[string]any) bool {
	changed := false
	walkCatalogRefreshObjects(root, func(obj map[string]any) {
		if ref := catalogRefreshString(obj["$ref"]); ref == "../policies.yaml" {
			delete(obj, "$ref")
			changed = true
		}
		if _, isBoolRequired := obj["required"].(bool); isBoolRequired {
			if _, isParameter := obj["in"].(string); !isParameter {
				delete(obj, "required")
				changed = true
			}
		}
		properties, ok := obj["properties"].(map[string]any)
		if !ok {
			return
		}
		required, ok := obj["required"].([]any)
		if !ok {
			return
		}
		filtered := make([]any, 0, len(required))
		for _, item := range required {
			name := catalogRefreshString(item)
			if name == "" {
				continue
			}
			if _, exists := properties[name]; exists {
				filtered = append(filtered, item)
				continue
			}
			changed = true
		}
		switch {
		case len(filtered) == 0 && len(required) > 0:
			delete(obj, "required")
		case len(filtered) != len(required):
			obj["required"] = filtered
		}
	})
	return changed
}

func ensureCatalogRefreshObject(root map[string]any, keys ...string) map[string]any {
	current := root
	for _, key := range keys {
		child, ok := current[key].(map[string]any)
		if !ok {
			child = map[string]any{}
			current[key] = child
		}
		current = child
	}
	return current
}

func walkCatalogRefreshObjects(value any, visit func(map[string]any)) {
	switch typed := value.(type) {
	case map[string]any:
		visit(typed)
		for _, child := range typed {
			walkCatalogRefreshObjects(child, visit)
		}
	case []any:
		for _, child := range typed {
			walkCatalogRefreshObjects(child, visit)
		}
	}
}

func catalogRefreshString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
