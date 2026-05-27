package odata

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// DetectLocalArtifact classifies local OData artifact bytes without calling
// tenant metadata endpoints or executing OData operations.
func DetectLocalArtifact(data []byte, path string) (ArtifactDetection, bool) {
	model, err := Parse(data)
	if err != nil {
		return ArtifactDetection{}, false
	}
	reason := "OData CSDL/service metadata parsed locally"
	if model.SourceKind == ArtifactKindCSDLXML {
		reason = "OData CSDL XML metadata"
	} else if model.SourceKind == ArtifactKindCSDLJSON {
		reason = "OData JSON CSDL metadata"
	}
	return ArtifactDetection{
		Kind:   model.SourceKind,
		Format: odataFormatForPath(path, model.SourceKind),
		Path:   strings.TrimSpace(path),
		Reason: reason,
	}, true
}

// Parse decodes local OData CSDL XML or JSON CSDL/service metadata. It does not
// call tenant $metadata endpoints, execute $batch, execute OData operations,
// resolve credentials, log in to tenants, or choose accounts.
func Parse(data []byte) (*Model, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("odata: empty document")
	}
	if trimmed[0] == '{' {
		if model, err := parseJSONCSDL(trimmed); err == nil {
			return model, nil
		} else if json.Valid(trimmed) {
			return nil, err
		}
	}
	if trimmed[0] == '<' {
		return parseXMLCSDL(trimmed)
	}
	return nil, fmt.Errorf("odata: document does not contain OData CSDL metadata")
}

func parseXMLCSDL(data []byte) (*Model, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	model := &Model{SourceKind: ArtifactKindCSDLXML}
	var schema *Schema
	var entityType *EntityType
	var operation *Operation
	var container *EntityContainer
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("odata: decode CSDL XML: %w", err)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			name := tok.Name.Local
			switch name {
			case "Edmx":
				if version := xmlAttr(tok, "Version"); version != "" {
					model.Version = version
				}
			case "Schema":
				schema = &Schema{Namespace: xmlAttr(tok, "Namespace"), Alias: xmlAttr(tok, "Alias")}
				model.Schemas = append(model.Schemas, schema)
			case "EntityType", "ComplexType":
				if schema == nil {
					continue
				}
				entityType = &EntityType{
					Name:     xmlAttr(tok, "Name"),
					BaseType: normalizeTypeName(xmlAttr(tok, "BaseType")),
					OpenType: parseBool(xmlAttr(tok, "OpenType")),
				}
				entityType.FullName = joinName(schema.Namespace, entityType.Name)
				if name == "ComplexType" {
					schema.ComplexTypes = append(schema.ComplexTypes, entityType)
				} else {
					schema.EntityTypes = append(schema.EntityTypes, entityType)
				}
			case "Property":
				if entityType == nil {
					continue
				}
				entityType.Properties = append(entityType.Properties, &Property{
					Name:       xmlAttr(tok, "Name"),
					Type:       normalizeTypeName(xmlAttr(tok, "Type")),
					Nullable:   nullableXML(tok),
					Collection: isCollectionType(xmlAttr(tok, "Type")),
				})
			case "NavigationProperty":
				if entityType == nil {
					continue
				}
				nav := &NavigationProperty{
					Name:       xmlAttr(tok, "Name"),
					Type:       normalizeTypeName(xmlAttr(tok, "Type")),
					Nullable:   nullableXML(tok),
					Collection: isCollectionType(xmlAttr(tok, "Type")),
					Partner:    xmlAttr(tok, "Partner"),
					Contains:   parseBool(xmlAttr(tok, "ContainsTarget")),
				}
				nav.Selector = NavigationSelector(entityType.FullName, nav.Name)
				entityType.NavigationProperties = append(entityType.NavigationProperties, nav)
			case "Action", "Function":
				if schema == nil {
					continue
				}
				operation = &Operation{
					Name:          xmlAttr(tok, "Name"),
					Kind:          strings.ToLower(name),
					Bound:         parseBool(xmlAttr(tok, "IsBound")),
					EntitySetPath: xmlAttr(tok, "EntitySetPath"),
					Composable:    parseBool(xmlAttr(tok, "IsComposable")),
				}
				operation.FullName = joinName(schema.Namespace, operation.Name)
				if name == "Action" {
					schema.Actions = append(schema.Actions, operation)
				} else {
					schema.Functions = append(schema.Functions, operation)
				}
			case "Parameter":
				if operation == nil {
					continue
				}
				operation.Parameters = append(operation.Parameters, &Parameter{
					Name:       xmlAttr(tok, "Name"),
					Type:       normalizeTypeName(xmlAttr(tok, "Type")),
					Nullable:   nullableXML(tok),
					Collection: isCollectionType(xmlAttr(tok, "Type")),
				})
			case "ReturnType":
				if operation != nil {
					operation.ReturnType = normalizeTypeName(xmlAttr(tok, "Type"))
				}
			case "EntityContainer":
				if schema == nil {
					continue
				}
				container = &EntityContainer{Name: xmlAttr(tok, "Name")}
				container.FullName = joinName(schema.Namespace, container.Name)
				schema.EntityContainer = container
			case "EntitySet":
				if container == nil {
					continue
				}
				entitySet := &EntitySet{Name: xmlAttr(tok, "Name"), EntityType: normalizeTypeName(xmlAttr(tok, "EntityType"))}
				entitySet.Selector = EntitySetSelector(entitySet.Name)
				container.EntitySets = append(container.EntitySets, entitySet)
			case "Singleton":
				if container == nil {
					continue
				}
				singleton := &Singleton{Name: xmlAttr(tok, "Name"), EntityType: normalizeTypeName(xmlAttr(tok, "Type"))}
				singleton.Selector = SingletonSelector(singleton.Name)
				container.Singletons = append(container.Singletons, singleton)
			case "ActionImport", "FunctionImport":
				if container == nil {
					continue
				}
				kind := "action"
				operationName := normalizeTypeName(xmlAttr(tok, "Action"))
				if name == "FunctionImport" {
					kind = "function"
					operationName = normalizeTypeName(xmlAttr(tok, "Function"))
				}
				opImport := &OperationImport{
					Name:      xmlAttr(tok, "Name"),
					Kind:      kind,
					Operation: operationName,
					EntitySet: xmlAttr(tok, "EntitySet"),
				}
				if kind == "action" {
					container.ActionImports = append(container.ActionImports, opImport)
				} else {
					container.FunctionImports = append(container.FunctionImports, opImport)
				}
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "EntityType", "ComplexType":
				entityType = nil
			case "Action", "Function":
				operation = nil
			case "EntityContainer":
				container = nil
			case "Schema":
				schema = nil
			}
		}
	}
	if len(model.Schemas) == 0 {
		return nil, fmt.Errorf("odata: CSDL XML missing Schema")
	}
	finalizeModel(model)
	if !modelHasMetadata(model) {
		return nil, fmt.Errorf("odata: CSDL XML has no entity sets, singletons, actions, functions, or entity types")
	}
	return model, nil
}

func parseJSONCSDL(data []byte) (*Model, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("odata: decode CSDL JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("odata: CSDL JSON contains trailing content")
	}
	if !looksLikeJSONCSDL(root) {
		return nil, fmt.Errorf("odata: JSON document is not OData CSDL")
	}
	model := &Model{SourceKind: ArtifactKindCSDLJSON, Version: stringField(root, "$Version")}
	for namespace, value := range root {
		if strings.HasPrefix(namespace, "$") {
			continue
		}
		obj, ok := value.(map[string]any)
		if !ok {
			continue
		}
		schema := parseJSONSchema(namespace, obj)
		if schema != nil {
			model.Schemas = append(model.Schemas, schema)
		}
	}
	finalizeModel(model)
	if len(model.Schemas) == 0 || !modelHasMetadata(model) {
		return nil, fmt.Errorf("odata: CSDL JSON has no parseable schemas")
	}
	return model, nil
}

func parseJSONSchema(namespace string, root map[string]any) *Schema {
	schema := &Schema{Namespace: namespace}
	if alias := stringField(root, "$Alias"); alias != "" {
		schema.Alias = alias
	}
	for name, value := range root {
		if strings.HasPrefix(name, "$") {
			continue
		}
		obj, ok := value.(map[string]any)
		if !ok {
			continue
		}
		kind := strings.TrimPrefix(stringField(obj, "$Kind"), "Edm.")
		switch kind {
		case "EntityType", "":
			if kind == "" && !hasAny(obj, "$Key", "$BaseType") && !jsonObjectHasPropertyShape(obj) {
				continue
			}
			entityType := parseJSONEntityType(schema.Namespace, name, obj)
			schema.EntityTypes = append(schema.EntityTypes, entityType)
		case "ComplexType":
			entityType := parseJSONEntityType(schema.Namespace, name, obj)
			schema.ComplexTypes = append(schema.ComplexTypes, entityType)
		case "Action":
			schema.Actions = append(schema.Actions, parseJSONOperation(schema.Namespace, "action", name, obj))
		case "Function":
			schema.Functions = append(schema.Functions, parseJSONOperation(schema.Namespace, "function", name, obj))
		case "EntityContainer":
			schema.EntityContainer = parseJSONEntityContainer(schema.Namespace, name, obj)
		}
	}
	return schema
}

func parseJSONEntityType(namespace, name string, root map[string]any) *EntityType {
	entityType := &EntityType{
		Name:     name,
		FullName: joinName(namespace, name),
		BaseType: normalizeTypeName(stringField(root, "$BaseType")),
		OpenType: boolField(root, "$OpenType"),
	}
	for propName, value := range root {
		if strings.HasPrefix(propName, "$") {
			continue
		}
		propRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		propKind := stringField(propRoot, "$Kind")
		propType := normalizeTypeName(stringField(propRoot, "$Type"))
		if propKind == "NavigationProperty" {
			nav := &NavigationProperty{
				Name:       propName,
				Type:       propType,
				Nullable:   nullableJSON(propRoot),
				Collection: isCollectionType(stringField(propRoot, "$Type")),
				Partner:    stringField(propRoot, "$Partner"),
				Contains:   boolField(propRoot, "$ContainsTarget"),
			}
			nav.Selector = NavigationSelector(entityType.FullName, nav.Name)
			entityType.NavigationProperties = append(entityType.NavigationProperties, nav)
			continue
		}
		entityType.Properties = append(entityType.Properties, &Property{
			Name:       propName,
			Type:       propType,
			Nullable:   nullableJSON(propRoot),
			Collection: isCollectionType(stringField(propRoot, "$Type")),
		})
	}
	return entityType
}

func parseJSONOperation(namespace, kind, name string, root map[string]any) *Operation {
	operation := &Operation{
		Name:          name,
		FullName:      joinName(namespace, name),
		Kind:          kind,
		Bound:         boolField(root, "$IsBound"),
		EntitySetPath: stringField(root, "$EntitySetPath"),
		Composable:    boolField(root, "$IsComposable"),
		ReturnType:    jsonReturnType(root["$ReturnType"]),
	}
	for _, paramRoot := range objectsFromArray(arrayValue(root["$Parameter"])) {
		operation.Parameters = append(operation.Parameters, parseJSONParameter(paramRoot))
	}
	return operation
}

func parseJSONParameter(root map[string]any) *Parameter {
	return &Parameter{
		Name:       stringField(root, "$Name"),
		Type:       normalizeTypeName(stringField(root, "$Type")),
		Nullable:   nullableJSON(root),
		Collection: isCollectionType(stringField(root, "$Type")),
	}
}

func parseJSONEntityContainer(namespace, name string, root map[string]any) *EntityContainer {
	container := &EntityContainer{Name: name, FullName: joinName(namespace, name)}
	for memberName, value := range root {
		if strings.HasPrefix(memberName, "$") {
			continue
		}
		memberRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		kind := stringField(memberRoot, "$Kind")
		switch kind {
		case "EntitySet":
			entitySet := &EntitySet{Name: memberName, EntityType: normalizeTypeName(stringField(memberRoot, "$Type"))}
			entitySet.Selector = EntitySetSelector(entitySet.Name)
			container.EntitySets = append(container.EntitySets, entitySet)
		case "Singleton":
			singleton := &Singleton{Name: memberName, EntityType: normalizeTypeName(stringField(memberRoot, "$Type"))}
			singleton.Selector = SingletonSelector(singleton.Name)
			container.Singletons = append(container.Singletons, singleton)
		case "ActionImport":
			opImport := &OperationImport{Name: memberName, Kind: "action", Operation: normalizeTypeName(stringField(memberRoot, "$Action")), EntitySet: stringField(memberRoot, "$EntitySet")}
			container.ActionImports = append(container.ActionImports, opImport)
		case "FunctionImport":
			opImport := &OperationImport{Name: memberName, Kind: "function", Operation: normalizeTypeName(stringField(memberRoot, "$Function")), EntitySet: stringField(memberRoot, "$EntitySet")}
			container.FunctionImports = append(container.FunctionImports, opImport)
		}
	}
	return container
}

func finalizeModel(model *Model) {
	for _, schema := range model.Schemas {
		for _, entityType := range append(append([]*EntityType{}, schema.EntityTypes...), schema.ComplexTypes...) {
			entityType.Selector = "#/entityTypes/" + escapeJSONPointer(entityType.FullName)
			for _, nav := range entityType.NavigationProperties {
				if nav.Selector == "" {
					nav.Selector = NavigationSelector(entityType.FullName, nav.Name)
				}
			}
		}
		for _, operation := range schema.Actions {
			finalizeOperation(operation)
		}
		for _, operation := range schema.Functions {
			finalizeOperation(operation)
		}
		if schema.EntityContainer != nil {
			container := schema.EntityContainer
			container.Selector = "#/entityContainers/" + escapeJSONPointer(container.FullName)
			for _, entitySet := range container.EntitySets {
				if entitySet.Selector == "" {
					entitySet.Selector = EntitySetSelector(entitySet.Name)
				}
			}
			for _, singleton := range container.Singletons {
				if singleton.Selector == "" {
					singleton.Selector = SingletonSelector(singleton.Name)
				}
			}
			for _, opImport := range container.ActionImports {
				finalizeOperationImport(opImport)
			}
			for _, opImport := range container.FunctionImports {
				finalizeOperationImport(opImport)
			}
		}
	}
	sort.SliceStable(model.Schemas, func(i, j int) bool { return model.Schemas[i].Namespace < model.Schemas[j].Namespace })
}

func finalizeOperation(operation *Operation) {
	if operation == nil {
		return
	}
	operation.SourceOperationID = operation.Kind + "." + operation.FullName
	operation.Selector = OperationSelector(operation.Kind, operation.FullName)
	operation.SourceOperationRef = operation.Selector
}

func finalizeOperationImport(opImport *OperationImport) {
	if opImport == nil {
		return
	}
	opImport.SourceOperationID = opImport.Kind + "-import." + opImport.Name
	opImport.Selector = "#/" + opImport.Kind + "Imports/" + escapeJSONPointer(opImport.Name)
	opImport.SourceOperationRef = opImport.Selector
}

func looksLikeJSONCSDL(root map[string]any) bool {
	if root["$Version"] != nil || root["$EntityContainer"] != nil {
		return true
	}
	for key, value := range root {
		if strings.HasPrefix(key, "$") {
			continue
		}
		if obj, ok := value.(map[string]any); ok {
			for memberName, member := range obj {
				if strings.HasPrefix(memberName, "$") {
					continue
				}
				if memberObj, ok := member.(map[string]any); ok && stringField(memberObj, "$Kind") != "" {
					return true
				}
			}
		}
	}
	return false
}

func modelHasMetadata(model *Model) bool {
	for _, schema := range model.Schemas {
		if len(schema.EntityTypes) > 0 || len(schema.ComplexTypes) > 0 || len(schema.Actions) > 0 || len(schema.Functions) > 0 {
			return true
		}
		if schema.EntityContainer != nil && (len(schema.EntityContainer.EntitySets) > 0 || len(schema.EntityContainer.Singletons) > 0 || len(schema.EntityContainer.ActionImports) > 0 || len(schema.EntityContainer.FunctionImports) > 0) {
			return true
		}
	}
	return false
}

func xmlAttr(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func nullableXML(start xml.StartElement) bool {
	value := xmlAttr(start, "Nullable")
	return value == "" || !strings.EqualFold(value, "false")
}

func nullableJSON(root map[string]any) bool {
	value, ok := root["$Nullable"].(bool)
	return !ok || value
}

func jsonReturnType(value any) string {
	switch typed := value.(type) {
	case string:
		return normalizeTypeName(typed)
	case map[string]any:
		return normalizeTypeName(stringField(typed, "$Type"))
	default:
		return ""
	}
}

func normalizeTypeName(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "Collection(") && strings.HasSuffix(value, ")") {
		value = strings.TrimSuffix(strings.TrimPrefix(value, "Collection("), ")")
	}
	return strings.TrimSpace(value)
}

func isCollectionType(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "Collection(")
}

func parseBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func joinName(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), ".")
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, ".")
}

func odataFormatForPath(path string, kind ArtifactKind) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xml", ".csdl", ".edmx":
		return "xml"
	case ".json":
		return "json"
	}
	if kind == ArtifactKindCSDLJSON {
		return "json"
	}
	return "xml"
}

func hasAny(root map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := root[key]; ok {
			return true
		}
	}
	return false
}

func jsonObjectHasPropertyShape(root map[string]any) bool {
	for key, value := range root {
		if strings.HasPrefix(key, "$") {
			continue
		}
		if obj, ok := value.(map[string]any); ok && obj["$Type"] != nil {
			return true
		}
	}
	return false
}

func arrayValue(value any) []any {
	out, _ := value.([]any)
	return out
}

func objectsFromArray(values []any) []map[string]any {
	var out []map[string]any
	for _, value := range values {
		if obj, ok := value.(map[string]any); ok {
			out = append(out, obj)
		}
	}
	return out
}

func stringField(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return strings.TrimSpace(value)
}

func boolField(root map[string]any, key string) bool {
	value, _ := root[key].(bool)
	return value
}
