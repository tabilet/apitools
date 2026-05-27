package openrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Parse decodes an OpenRPC JSON document into a metadata model. It performs no
// network access, server validation, JSON-RPC invocation, or credential lookup.
func Parse(data []byte) (*Model, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("openrpc: empty document")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("openrpc: decode document: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("openrpc: document contains trailing content")
	}
	if len(root) == 0 {
		return nil, fmt.Errorf("openrpc: document root must be an object")
	}
	version := stringField(root, "openrpc")
	if version == "" {
		return nil, fmt.Errorf("openrpc: missing openrpc version")
	}
	infoRoot, ok := objectField(root, "info")
	if !ok {
		return nil, fmt.Errorf("openrpc: missing info object")
	}
	methodValues, ok := arrayField(root, "methods")
	if !ok {
		return nil, fmt.Errorf("openrpc: missing methods array")
	}

	model := &Model{
		OpenRPC:    version,
		Info:       parseInfo(infoRoot),
		Servers:    parseServers(root["servers"]),
		Components: parseComponents(root["components"]),
		Raw:        root,
	}
	seen := map[string]struct{}{}
	for i, value := range methodValues {
		methodRoot, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("openrpc: methods[%d] must be an object", i)
		}
		method := parseMethod(methodRoot)
		if method.Name == "" {
			return nil, fmt.Errorf("openrpc: methods[%d] missing name", i)
		}
		if _, exists := seen[method.Name]; exists {
			return nil, fmt.Errorf("openrpc: duplicate method name %q", method.Name)
		}
		seen[method.Name] = struct{}{}
		model.Methods = append(model.Methods, method)
	}
	return model, nil
}

func parseInfo(root map[string]any) Info {
	return Info{
		Title:          stringField(root, "title"),
		Version:        stringField(root, "version"),
		Description:    stringField(root, "description"),
		TermsOfService: stringField(root, "termsOfService"),
		Contact:        objectValue(root["contact"]),
		License:        objectValue(root["license"]),
		Raw:            root,
	}
}

func parseServers(value any) []*Server {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]*Server, 0, len(values))
	for _, item := range values {
		root, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, &Server{
			Name:        stringField(root, "name"),
			URL:         stringField(root, "url"),
			Summary:     stringField(root, "summary"),
			Description: stringField(root, "description"),
			Variables:   objectValue(root["variables"]),
			Raw:         root,
		})
	}
	return out
}

func parseMethod(root map[string]any) *Method {
	method := &Method{
		Name:           stringField(root, "name"),
		Summary:        stringField(root, "summary"),
		Description:    stringField(root, "description"),
		ParamStructure: stringField(root, "paramStructure"),
		Params:         parseContentDescriptorArray(root["params"]),
		Result:         parseContentDescriptor(root["result"]),
		Errors:         parseErrorArray(root["errors"]),
		Tags:           parseTagNames(root["tags"]),
		ExternalDocs:   objectValue(root["externalDocs"]),
		Deprecated:     boolField(root, "deprecated"),
		Servers:        parseServers(root["servers"]),
		Raw:            root,
	}
	return method
}

func parseComponents(value any) Components {
	root, ok := value.(map[string]any)
	if !ok {
		return Components{}
	}
	return Components{
		Schemas:            objectValue(root["schemas"]),
		ContentDescriptors: parseNamedContentDescriptors(root["contentDescriptors"]),
		Errors:             parseNamedErrors(root["errors"]),
		Examples:           objectValue(root["examples"]),
		Links:              objectValue(root["links"]),
		Tags:               objectValue(root["tags"]),
		Raw:                root,
	}
}

func parseNamedContentDescriptors(value any) map[string]*ContentDescriptor {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]*ContentDescriptor, len(root))
	for name, item := range root {
		descriptor := parseContentDescriptor(item)
		if descriptor == nil {
			continue
		}
		if descriptor.Name == "" && descriptor.Ref == "" {
			descriptor.Name = name
		}
		out[name] = descriptor
	}
	return out
}

func parseNamedErrors(value any) map[string]*Error {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]*Error, len(root))
	for name, item := range root {
		errDef := parseError(item)
		if errDef == nil {
			continue
		}
		if errDef.Message == "" && errDef.Ref == "" {
			errDef.Message = name
		}
		out[name] = errDef
	}
	return out
}

func parseContentDescriptorArray(value any) []*ContentDescriptor {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]*ContentDescriptor, 0, len(values))
	for _, item := range values {
		descriptor := parseContentDescriptor(item)
		if descriptor != nil {
			out = append(out, descriptor)
		}
	}
	return out
}

func parseContentDescriptor(value any) *ContentDescriptor {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &ContentDescriptor{
		Name:        stringField(root, "name"),
		Summary:     stringField(root, "summary"),
		Description: stringField(root, "description"),
		Required:    boolField(root, "required"),
		Deprecated:  boolField(root, "deprecated"),
		Ref:         stringField(root, "$ref"),
		Schema:      objectValue(root["schema"]),
		Raw:         root,
	}
}

func parseErrorArray(value any) []*Error {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]*Error, 0, len(values))
	for _, item := range values {
		errDef := parseError(item)
		if errDef != nil {
			out = append(out, errDef)
		}
	}
	return out
}

func parseError(value any) *Error {
	root, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &Error{
		Code:    scalarString(root["code"]),
		Message: stringField(root, "message"),
		Data:    root["data"],
		Ref:     stringField(root, "$ref"),
		Raw:     root,
	}
}

func parseTagNames(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, item := range values {
		name := ""
		switch typed := item.(type) {
		case string:
			name = strings.TrimSpace(typed)
		case map[string]any:
			name = stringField(typed, "name")
			if name == "" {
				name = stringField(typed, "$ref")
			}
		}
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func objectField(root map[string]any, key string) (map[string]any, bool) {
	value, ok := root[key].(map[string]any)
	return value, ok
}

func arrayField(root map[string]any, key string) ([]any, bool) {
	value, ok := root[key].([]any)
	return value, ok
}

func objectValue(value any) map[string]any {
	root, _ := value.(map[string]any)
	return root
}

func stringField(root map[string]any, key string) string {
	return strings.TrimSpace(scalarString(root[key]))
}

func boolField(root map[string]any, key string) bool {
	value, _ := root[key].(bool)
	return value
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}
