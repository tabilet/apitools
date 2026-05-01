package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/tabilet/oas/openapi20"
	"github.com/tabilet/oas/openapi30"
	"github.com/tabilet/oas/openapi31"
	"gopkg.in/yaml.v3"
)

func specMetadata(ctx context.Context, content []byte) (SpecMetadata, bool) {
	_ = ctx
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return SpecMetadata{}, false
	}
	var root map[string]any
	if trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &root); err != nil {
			return SpecMetadata{}, false
		}
	} else if err := yaml.Unmarshal(trimmed, &root); err != nil {
		return SpecMetadata{}, false
	}
	openapi, _ := root["openapi"].(string)
	swagger, _ := root["swagger"].(string)
	if strings.TrimSpace(openapi) == "" && strings.TrimSpace(swagger) == "" {
		return SpecMetadata{}, false
	}
	if containsExternalRef(root) {
		return SpecMetadata{}, false
	}
	normalized, err := json.Marshal(root)
	if err != nil {
		return SpecMetadata{}, false
	}

	openapi = strings.TrimSpace(openapi)
	if openapi != "" {
		switch {
		case strings.HasPrefix(openapi, "3.0"):
			return specMetadataV30(normalized)
		case strings.HasPrefix(openapi, "3.1"):
			return specMetadataV31(normalized)
		default:
			return SpecMetadata{}, false
		}
	}

	if !validSwagger2Root(root) {
		return SpecMetadata{}, false
	}
	var doc openapi20.Swagger
	if err := json.Unmarshal(normalized, &doc); err != nil {
		return SpecMetadata{}, false
	}
	if strings.TrimSpace(doc.Swagger) != "2.0" || strings.TrimSpace(doc.Info.Title) == "" || strings.TrimSpace(doc.Info.Version) == "" || doc.Paths == nil {
		return SpecMetadata{}, false
	}
	return SpecMetadata{
		Title:          strings.TrimSpace(doc.Info.Title),
		Description:    strings.TrimSpace(doc.Info.Description),
		Swagger:        strings.TrimSpace(doc.Swagger),
		OperationCount: operationCountV2(doc.Paths),
	}, true
}

func specMetadataV30(content []byte) (SpecMetadata, bool) {
	var doc openapi30.OpenAPI
	if err := json.Unmarshal(content, &doc); err != nil {
		return SpecMetadata{}, false
	}
	if result := doc.Validate(); !result.Valid() {
		return SpecMetadata{}, false
	}
	return SpecMetadata{
		Title:          strings.TrimSpace(doc.Info.Title),
		Description:    strings.TrimSpace(doc.Info.Description),
		OpenAPI:        strings.TrimSpace(doc.OpenAPI),
		OperationCount: operationCountV30(doc.Paths),
	}, true
}

func specMetadataV31(content []byte) (SpecMetadata, bool) {
	var doc openapi31.OpenAPI
	if err := json.Unmarshal(content, &doc); err != nil {
		return SpecMetadata{}, false
	}
	if result := doc.Validate(); !result.Valid() {
		return SpecMetadata{}, false
	}
	return SpecMetadata{
		Title:          strings.TrimSpace(doc.Info.Title),
		Description:    strings.TrimSpace(doc.Info.Description),
		OpenAPI:        strings.TrimSpace(doc.OpenAPI),
		OperationCount: operationCountV31(doc.Paths),
	}, true
}

func containsExternalRef(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" {
				ref, _ := child.(string)
				if isExternalRef(ref) {
					return true
				}
			}
			if containsExternalRef(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsExternalRef(child) {
				return true
			}
		}
	}
	return false
}

func isExternalRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return ref != "" && !strings.HasPrefix(ref, "#")
}

func validSwagger2Root(root map[string]any) bool {
	swagger, _ := root["swagger"].(string)
	if strings.TrimSpace(swagger) != "2.0" {
		return false
	}
	info, ok := root["info"].(map[string]any)
	if !ok {
		return false
	}
	title, _ := info["title"].(string)
	version, _ := info["version"].(string)
	if strings.TrimSpace(title) == "" || strings.TrimSpace(version) == "" {
		return false
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return false
	}
	for path, value := range paths {
		if !strings.HasPrefix(path, "/") {
			return false
		}
		pathItem, ok := value.(map[string]any)
		if !ok {
			return false
		}
		if !validSwagger2PathItem(pathItem) {
			return false
		}
	}
	return true
}

func validSwagger2PathItem(pathItem map[string]any) bool {
	for key, value := range pathItem {
		if strings.HasPrefix(key, "x-") || key == "$ref" {
			continue
		}
		if key == "parameters" {
			if _, ok := value.([]any); !ok {
				return false
			}
			continue
		}
		if !isHTTPMethod(key) {
			return false
		}
		operation, ok := value.(map[string]any)
		if !ok {
			return false
		}
		responses, ok := operation["responses"].(map[string]any)
		if !ok || len(responses) == 0 {
			return false
		}
		for _, responseValue := range responses {
			response, ok := responseValue.(map[string]any)
			if !ok {
				return false
			}
			if ref, _ := response["$ref"].(string); strings.TrimSpace(ref) != "" {
				continue
			}
			description, _ := response["description"].(string)
			if strings.TrimSpace(description) == "" {
				return false
			}
		}
	}
	return true
}

func isHTTPMethod(method string) bool {
	switch method {
	case "delete", "get", "head", "options", "patch", "post", "put":
		return true
	default:
		return false
	}
}

func operationCountV30(paths *openapi30.Paths) int {
	count := 0
	if paths == nil {
		return count
	}
	for _, path := range paths.Paths {
		if path == nil {
			continue
		}
		if path.Get != nil {
			count++
		}
		if path.Put != nil {
			count++
		}
		if path.Post != nil {
			count++
		}
		if path.Delete != nil {
			count++
		}
		if path.Options != nil {
			count++
		}
		if path.Head != nil {
			count++
		}
		if path.Patch != nil {
			count++
		}
		if path.Trace != nil {
			count++
		}
	}
	return count
}

func operationCountV31(paths *openapi31.Paths) int {
	count := 0
	if paths == nil {
		return count
	}
	for _, path := range paths.Paths {
		if path == nil {
			continue
		}
		if path.Get != nil {
			count++
		}
		if path.Put != nil {
			count++
		}
		if path.Post != nil {
			count++
		}
		if path.Delete != nil {
			count++
		}
		if path.Options != nil {
			count++
		}
		if path.Head != nil {
			count++
		}
		if path.Patch != nil {
			count++
		}
		if path.Trace != nil {
			count++
		}
	}
	return count
}

func operationCountV2(paths *openapi20.Paths) int {
	count := 0
	if paths == nil {
		return count
	}
	for _, path := range paths.Paths {
		if path == nil {
			continue
		}
		if path.Get != nil {
			count++
		}
		if path.Put != nil {
			count++
		}
		if path.Post != nil {
			count++
		}
		if path.Delete != nil {
			count++
		}
		if path.Options != nil {
			count++
		}
		if path.Head != nil {
			count++
		}
		if path.Patch != nil {
			count++
		}
	}
	return count
}
