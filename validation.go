package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/OpenUdon/oas/openapi20"
	"github.com/OpenUdon/oas/openapi30"
	"github.com/OpenUdon/oas/openapi31"
	"gopkg.in/yaml.v3"
)

func specMetadata(ctx context.Context, content []byte) (SpecMetadata, bool) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return SpecMetadata{}, false
		}
	}
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
	if containsExternalRef(root, 0) {
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

func downloadedSpecMetadata(ctx context.Context, content []byte, sourceURL string) (SpecMetadata, bool) {
	if metadata, ok := specMetadata(ctx, content); ok {
		return metadata, true
	}
	if !awsOpenAPIFallbackSource(sourceURL) {
		return SpecMetadata{}, false
	}
	return awsOpenAPIMetadata(ctx, content)
}

func awsOpenAPIFallbackSource(sourceURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(sourceURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(parsed.EscapedPath())
	switch host {
	case "raw.githubusercontent.com":
		return strings.Contains(path, "/apis-guru/openapi-directory/") && strings.Contains(path, "/apis/amazonaws.com/")
	case "github.com":
		return strings.Contains(path, "/apis-guru/openapi-directory/") && strings.Contains(path, "/apis/amazonaws.com/")
	default:
		return false
	}
}

func awsOpenAPIMetadata(ctx context.Context, content []byte) (SpecMetadata, bool) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return SpecMetadata{}, false
		}
	}
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
	if !strings.HasPrefix(strings.TrimSpace(openapi), "3.") {
		return SpecMetadata{}, false
	}
	if containsExternalRef(root, 0) {
		return SpecMetadata{}, false
	}
	info, ok := root["info"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	title, _ := info["title"].(string)
	version, _ := info["version"].(string)
	if !awsTitle(title) || strings.TrimSpace(version) == "" {
		return SpecMetadata{}, false
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	description, _ := info["description"].(string)
	return SpecMetadata{
		Title:          strings.TrimSpace(title),
		Description:    strings.TrimSpace(description),
		OpenAPI:        strings.TrimSpace(openapi),
		OperationCount: looseOperationCount(paths),
	}, true
}

func awsTitle(title string) bool {
	normalized := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(normalized, "aws") || strings.Contains(normalized, "amazon")
}

func looseOperationCount(paths map[string]any) int {
	count := 0
	for _, value := range paths {
		pathItem, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
			if _, ok := pathItem[method].(map[string]any); ok {
				count++
			}
		}
	}
	return count
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

// maxRefScanDepth bounds containsExternalRef recursion so a pathologically
// nested document cannot exhaust the goroutine stack. OpenAPI documents in
// practice nest far below this limit; anything deeper is treated as if it
// contained an external ref and rejected.
const maxRefScanDepth = 256

func containsExternalRef(value any, depth int) bool {
	if depth > maxRefScanDepth {
		return true
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" {
				ref, _ := child.(string)
				if isExternalRef(ref) {
					return true
				}
			}
			if containsExternalRef(child, depth+1) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsExternalRef(child, depth+1) {
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

func countNonNilOps[T any](ops ...*T) int {
	n := 0
	for _, op := range ops {
		if op != nil {
			n++
		}
	}
	return n
}

func operationCountV30(paths *openapi30.Paths) int {
	if paths == nil {
		return 0
	}
	n := 0
	for _, p := range paths.Paths {
		if p == nil {
			continue
		}
		n += countNonNilOps(p.Get, p.Put, p.Post, p.Delete, p.Options, p.Head, p.Patch, p.Trace)
	}
	return n
}

func operationCountV31(paths *openapi31.Paths) int {
	if paths == nil {
		return 0
	}
	n := 0
	for _, p := range paths.Paths {
		if p == nil {
			continue
		}
		n += countNonNilOps(p.Get, p.Put, p.Post, p.Delete, p.Options, p.Head, p.Patch, p.Trace)
	}
	return n
}

func operationCountV2(paths *openapi20.Paths) int {
	if paths == nil {
		return 0
	}
	n := 0
	for _, p := range paths.Paths {
		if p == nil {
			continue
		}
		n += countNonNilOps(p.Get, p.Put, p.Post, p.Delete, p.Options, p.Head, p.Patch)
	}
	return n
}
