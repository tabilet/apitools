package awssmithy

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const openAPIVersion = "3.0.1"

var identifierRE = regexp.MustCompile(`[^A-Za-z0-9_]+`)

// Convert converts an AWS Smithy JSON service model into bounded OpenAPI 3.0.1
// shaped metadata.
func Convert(data []byte) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse smithy document: %w", err)
	}
	doc, err := ConvertMap(raw)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

// ConvertMap converts an already-decoded AWS Smithy JSON service model into
// bounded OpenAPI-shaped metadata.
func ConvertMap(raw map[string]any) (map[string]any, error) {
	c := &converter{
		raw:            raw,
		componentNames: map[string]string{},
		schemas:        map[string]map[string]any{},
		inflight:       map[string]bool{},
	}
	return c.convert()
}

type converter struct {
	raw            map[string]any
	shapes         map[string]any
	serviceID      string
	service        map[string]any
	protocol       string
	componentNames map[string]string
	schemas        map[string]map[string]any
	inflight       map[string]bool
}

func (c *converter) convert() (map[string]any, error) {
	if c.raw == nil {
		return nil, fmt.Errorf("smithy document root must be an object")
	}
	if got := stringValue(c.raw["smithy"]); got != "2.0" {
		return nil, fmt.Errorf("unsupported smithy version %q", got)
	}
	shapes, ok, err := requiredMapField(c.raw, "shapes", "smithy document")
	if err != nil {
		return nil, err
	}
	if !ok || len(shapes) == 0 {
		return nil, fmt.Errorf("smithy document missing shapes")
	}
	c.shapes = shapes
	if err := c.selectService(); err != nil {
		return nil, err
	}
	if err := c.assignComponentNames(); err != nil {
		return nil, err
	}

	components, err := c.convertComponents()
	if err != nil {
		return nil, err
	}
	paths, err := c.convertPaths()
	if err != nil {
		return nil, err
	}

	info := map[string]any{
		"title":   c.serviceTitle(),
		"version": firstNonEmpty(stringValue(c.service["version"]), "0.0.0"),
	}
	if description := smithyDocumentation(c.service); description != "" {
		info["description"] = description
	}

	out := map[string]any{
		"openapi":               openAPIVersion,
		"info":                  info,
		"servers":               []any{c.server()},
		"paths":                 paths,
		"x-smithy-protocol":     c.protocol,
		"x-smithy-service":      c.serviceID,
		"x-aws-service-id":      c.awsServiceID(),
		"x-aws-endpoint-prefix": c.awsEndpointPrefix(),
	}
	if signingName := c.awsSigningName(); signingName != "" {
		out["x-aws-signing-name"] = signingName
	}
	if len(components) > 0 {
		out["components"] = map[string]any{"schemas": components}
	}
	return out, nil
}

func (c *converter) selectService() error {
	var serviceIDs []string
	for _, id := range sortedKeys(c.shapes) {
		shape := mapValueOrNil(c.shapes[id])
		if stringValue(shape["type"]) == "service" {
			serviceIDs = append(serviceIDs, id)
		}
	}
	switch len(serviceIDs) {
	case 0:
		return fmt.Errorf("smithy document must contain exactly one service shape")
	case 1:
	default:
		return fmt.Errorf("smithy document contains multiple service shapes")
	}
	c.serviceID = serviceIDs[0]
	c.service = mapValueOrNil(c.shapes[c.serviceID])
	c.protocol = c.serviceProtocol()
	if c.protocol == "" {
		return fmt.Errorf("service %q uses unsupported or missing AWS Smithy protocol", c.serviceID)
	}
	return nil
}

func (c *converter) serviceProtocol() string {
	traits := traits(c.service)
	for _, protocol := range []string{"restJson1", "restXml", "awsQuery"} {
		if _, ok := traits["aws.protocols#"+protocol]; ok {
			return protocol
		}
	}
	return ""
}

func (c *converter) assignComponentNames() error {
	used := map[string]string{}
	for _, id := range sortedKeys(c.shapes) {
		shape := mapValueOrNil(c.shapes[id])
		switch stringValue(shape["type"]) {
		case "", "service", "operation", "resource":
			continue
		}
		name := sanitizeIdentifier(localName(id))
		if name == "" {
			name = sanitizeIdentifier(id)
		}
		if prior := used[name]; prior != "" && prior != id {
			name = sanitizeIdentifier(strings.NewReplacer("#", "_", "$", "_", ".", "_").Replace(id))
		}
		if name == "" {
			return fmt.Errorf("shape %q cannot be mapped to a component name", id)
		}
		used[name] = id
		c.componentNames[id] = name
	}
	return nil
}

func (c *converter) convertComponents() (map[string]any, error) {
	out := map[string]any{}
	for _, id := range sortedKeys(c.shapes) {
		name := c.componentNames[id]
		if name == "" {
			continue
		}
		schema, err := c.convertShape(id)
		if err != nil {
			return nil, err
		}
		out[name] = schema
	}
	return out, nil
}

func (c *converter) convertPaths() (map[string]any, error) {
	operationIDs, err := c.serviceOperations()
	if err != nil {
		return nil, err
	}
	paths := map[string]any{}
	for _, id := range operationIDs {
		operation := mapValueOrNil(c.shapes[id])
		if operation == nil {
			return nil, fmt.Errorf("operation %q is missing", id)
		}
		path, method, op, err := c.convertOperation(id, operation)
		if err != nil {
			return nil, err
		}
		if path == "" {
			continue
		}
		pathItem := mapValueOrNil(paths[path])
		if pathItem == nil {
			pathItem = map[string]any{}
			paths[path] = pathItem
		}
		pathItem[strings.ToLower(method)] = op
	}
	return paths, nil
}

func (c *converter) serviceOperations() ([]string, error) {
	seen := map[string]bool{}
	var out []string
	var addRef func(ref any) error
	addRef = func(ref any) error {
		target := targetValue(ref)
		if target == "" {
			return nil
		}
		shape := mapValueOrNil(c.shapes[target])
		if shape == nil {
			return fmt.Errorf("service references missing operation %q", target)
		}
		if stringValue(shape["type"]) != "operation" {
			return fmt.Errorf("service target %q is not an operation", target)
		}
		if !seen[target] {
			seen[target] = true
			out = append(out, target)
		}
		return nil
	}
	for _, ref := range sliceValueOrNil(c.service["operations"]) {
		if err := addRef(ref); err != nil {
			return nil, err
		}
	}
	var walkResource func(string) error
	walkResource = func(id string) error {
		resource := mapValueOrNil(c.shapes[id])
		if resource == nil {
			return fmt.Errorf("service references missing resource %q", id)
		}
		for _, key := range []string{"operations", "collectionOperations"} {
			for _, ref := range sliceValueOrNil(resource[key]) {
				if err := addRef(ref); err != nil {
					return err
				}
			}
		}
		for _, key := range []string{"put", "create", "read", "update", "delete", "list"} {
			if err := addRef(resource[key]); err != nil {
				return err
			}
		}
		for _, ref := range sliceValueOrNil(resource["resources"]) {
			if err := walkResource(targetValue(ref)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, ref := range sliceValueOrNil(c.service["resources"]) {
		if err := walkResource(targetValue(ref)); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return localName(out[i]) < localName(out[j])
	})
	return out, nil
}

func (c *converter) convertOperation(id string, operation map[string]any) (string, string, map[string]any, error) {
	opName := localName(id)
	httpTrait := mapValueOrNil(traits(operation)["smithy.api#http"])
	method := strings.ToUpper(stringValue(httpTrait["method"]))
	uri := firstNonEmpty(stringValue(httpTrait["uri"]), "/")
	code := intValue(httpTrait["code"], 200)
	if c.protocol == "awsQuery" {
		method = "POST"
		uri = "/"
	} else if len(httpTrait) == 0 {
		return "", "", nil, nil
	}
	path, queryLiterals, greedyLabels := splitSmithyURI(uri)
	if method == "" {
		return "", "", nil, fmt.Errorf("operation %q missing HTTP method", id)
	}
	inputTarget, err := c.operationTarget(id, operation, "input")
	if err != nil {
		return "", "", nil, err
	}
	outputTarget, err := c.operationTarget(id, operation, "output")
	if err != nil {
		return "", "", nil, err
	}

	op := map[string]any{
		"operationId":           sanitizeIdentifier(opName),
		"summary":               firstNonEmpty(summaryFromDocumentation(smithyDocumentation(operation)), opName),
		"description":           smithyDocumentation(operation),
		"responses":             c.responsesFor(outputTarget, code),
		"x-smithy-id":           id,
		"x-smithy-protocol":     c.protocol,
		"x-smithy-service":      c.serviceID,
		"x-aws-operation-name":  opName,
		"x-aws-service-id":      c.awsServiceID(),
		"x-aws-endpoint-prefix": c.awsEndpointPrefix(),
	}
	if signingName := c.awsSigningName(); signingName != "" {
		op["x-aws-signing-name"] = signingName
	}
	if c.protocol == "awsQuery" {
		op["x-aws-query-action"] = opName
		op["x-aws-query-version"] = stringValue(c.service["version"])
	}
	if len(queryLiterals) > 0 {
		op["x-smithy-http-query-literals"] = queryLiterals
	}
	if len(greedyLabels) > 0 {
		op["x-smithy-greedy-labels"] = greedyLabels
	}

	parameters, boundMembers, err := c.operationParameters(inputTarget, path, queryLiterals)
	if err != nil {
		return "", "", nil, err
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	requestBody, err := c.operationRequestBody(opName, inputTarget, boundMembers)
	if err != nil {
		return "", "", nil, err
	}
	if requestBody != nil {
		op["requestBody"] = requestBody
	}
	return path, method, op, nil
}

func (c *converter) operationTarget(operationID string, operation map[string]any, key string) (string, error) {
	target := targetValue(operation[key])
	if target == "" || target == "smithy.api#Unit" {
		return target, nil
	}
	if _, ok := c.shapes[target]; !ok && preludeSchema(target) == nil {
		return "", fmt.Errorf("operation %q references missing %s shape %q", localName(operationID), key, target)
	}
	return target, nil
}

func (c *converter) operationParameters(inputTarget, path string, queryLiterals map[string]string) ([]any, map[string]bool, error) {
	bound := map[string]bool{}
	var params []any
	if inputTarget != "" && inputTarget != "smithy.api#Unit" {
		input := mapValueOrNil(c.shapes[inputTarget])
		if stringValue(input["type"]) != "structure" {
			return nil, nil, fmt.Errorf("input shape %q must be a structure", inputTarget)
		}
		members := mapValueOrNil(input["members"])
		for _, name := range sortedKeys(members) {
			member := mapValueOrNil(members[name])
			mtraits := traits(member)
			in := ""
			paramName := name
			required := hasTrait(member, "smithy.api#required")
			switch {
			case hasTrait(member, "smithy.api#httpLabel"):
				in = "path"
				required = true
			case hasTrait(member, "smithy.api#httpQuery"):
				in = "query"
				if value := stringValue(mtraits["smithy.api#httpQuery"]); value != "" {
					paramName = value
				}
			case hasTrait(member, "smithy.api#httpHeader"):
				in = "header"
				if value := stringValue(mtraits["smithy.api#httpHeader"]); value != "" {
					paramName = value
				}
			default:
				continue
			}
			bound[name] = true
			schema, err := c.memberSchema(member)
			if err != nil {
				return nil, nil, err
			}
			param := map[string]any{
				"name":     paramName,
				"in":       in,
				"required": required,
				"schema":   schema,
			}
			if doc := smithyDocumentation(member); doc != "" {
				param["description"] = doc
			}
			params = append(params, param)
		}
	}
	for _, label := range pathLabels(path) {
		if hasParameter(params, "path", label) {
			continue
		}
		params = append(params, map[string]any{
			"name":     label,
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
	}
	for name, value := range queryLiterals {
		if hasParameter(params, "query", name) {
			continue
		}
		params = append(params, map[string]any{
			"name":     name,
			"in":       "query",
			"required": true,
			"schema":   map[string]any{"type": "string", "default": value},
		})
	}
	sort.SliceStable(params, func(i, j int) bool {
		left := params[i].(map[string]any)
		right := params[j].(map[string]any)
		if stringValue(left["in"]) != stringValue(right["in"]) {
			return stringValue(left["in"]) < stringValue(right["in"])
		}
		return stringValue(left["name"]) < stringValue(right["name"])
	})
	return params, bound, nil
}

func (c *converter) operationRequestBody(opName, inputTarget string, bound map[string]bool) (map[string]any, error) {
	if c.protocol == "awsQuery" {
		schema, err := c.awsQueryRequestSchema(opName, inputTarget)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"required": true,
			"content": map[string]any{
				"application/x-www-form-urlencoded": map[string]any{
					"schema": schema,
				},
			},
			"x-aws-query-action":  opName,
			"x-aws-query-version": stringValue(c.service["version"]),
		}, nil
	}
	if inputTarget == "" || inputTarget == "smithy.api#Unit" {
		return nil, nil
	}
	input := mapValueOrNil(c.shapes[inputTarget])
	members := mapValueOrNil(input["members"])
	payloadMember := ""
	for _, name := range sortedKeys(members) {
		if hasTrait(mapValueOrNil(members[name]), "smithy.api#httpPayload") {
			payloadMember = name
			break
		}
	}
	var schema map[string]any
	var required bool
	if payloadMember != "" {
		member := mapValueOrNil(members[payloadMember])
		var err error
		schema, err = c.memberSchema(member)
		if err != nil {
			return nil, err
		}
		required = hasTrait(member, "smithy.api#required")
	} else {
		bodySchema, err := c.structureRequestSchema(inputTarget, bound)
		if err != nil {
			return nil, err
		}
		if bodySchema == nil {
			return nil, nil
		}
		schema = bodySchema
		required = len(stringSlice(schema["required"])) > 0
	}
	mediaType := c.requestMediaType(schema)
	return map[string]any{
		"required": required,
		"content": map[string]any{
			mediaType: map[string]any{
				"schema": schema,
			},
		},
	}, nil
}

func (c *converter) structureRequestSchema(inputTarget string, bound map[string]bool) (map[string]any, error) {
	input := mapValueOrNil(c.shapes[inputTarget])
	members := mapValueOrNil(input["members"])
	props := map[string]any{}
	var required []string
	for _, name := range sortedKeys(members) {
		if bound[name] {
			continue
		}
		member := mapValueOrNil(members[name])
		if hasTrait(member, "smithy.api#httpResponseCode") {
			continue
		}
		schema, err := c.memberSchema(member)
		if err != nil {
			return nil, err
		}
		props[name] = schema
		if hasTrait(member, "smithy.api#required") {
			required = append(required, name)
		}
	}
	if len(props) == 0 {
		return nil, nil
	}
	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out, nil
}

func (c *converter) awsQueryRequestSchema(opName, inputTarget string) (map[string]any, error) {
	props := map[string]any{
		"Action":  map[string]any{"type": "string", "default": opName},
		"Version": map[string]any{"type": "string", "default": stringValue(c.service["version"])},
	}
	required := []string{"Action", "Version"}
	if inputTarget != "" && inputTarget != "smithy.api#Unit" {
		input := mapValueOrNil(c.shapes[inputTarget])
		if stringValue(input["type"]) != "structure" {
			return nil, fmt.Errorf("input shape %q must be a structure", inputTarget)
		}
		members := mapValueOrNil(input["members"])
		for _, name := range sortedKeys(members) {
			member := mapValueOrNil(members[name])
			schema, err := c.memberSchema(member)
			if err != nil {
				return nil, err
			}
			props[name] = schema
			if hasTrait(member, "smithy.api#required") {
				required = append(required, name)
			}
		}
	}
	return map[string]any{
		"type":        "object",
		"properties":  props,
		"required":    required,
		"x-aws-query": true,
	}, nil
}

func (c *converter) responsesFor(outputTarget string, code int) map[string]any {
	response := map[string]any{"description": "Success"}
	if outputTarget != "" && outputTarget != "smithy.api#Unit" {
		mediaType := "application/json"
		if c.protocol == "restXml" || c.protocol == "awsQuery" {
			mediaType = "application/xml"
		}
		response["content"] = map[string]any{
			mediaType: map[string]any{
				"schema": c.refForTarget(outputTarget),
			},
		}
	}
	return map[string]any{
		fmt.Sprintf("%d", code): response,
	}
}

func (c *converter) requestMediaType(schema map[string]any) string {
	if stringValue(schema["type"]) == "string" && stringValue(schema["format"]) == "binary" {
		return "application/octet-stream"
	}
	if c.protocol == "restXml" {
		return "application/xml"
	}
	return "application/json"
}

func (c *converter) convertShape(id string) (map[string]any, error) {
	if schema := preludeSchema(id); schema != nil {
		return schema, nil
	}
	if cached := c.schemas[id]; cached != nil {
		return cached, nil
	}
	shape := mapValueOrNil(c.shapes[id])
	if shape == nil {
		return nil, fmt.Errorf("shape %q is missing", id)
	}
	if c.inflight[id] {
		return c.refForTarget(id), nil
	}
	c.inflight[id] = true
	defer func() { c.inflight[id] = false }()

	out, err := c.convertShapeNoCache(id, shape)
	if err != nil {
		return nil, err
	}
	if doc := smithyDocumentation(shape); doc != "" {
		out["description"] = doc
	}
	if enumValues := enumValues(shape); len(enumValues) > 0 {
		out["enum"] = enumValues
		if out["type"] == nil {
			out["type"] = "string"
		}
	}
	c.schemas[id] = out
	return out, nil
}

func (c *converter) convertShapeNoCache(id string, shape map[string]any) (map[string]any, error) {
	switch stringValue(shape["type"]) {
	case "structure":
		props := map[string]any{}
		var required []string
		members := mapValueOrNil(shape["members"])
		for _, name := range sortedKeys(members) {
			member := mapValueOrNil(members[name])
			schema, err := c.memberSchema(member)
			if err != nil {
				return nil, err
			}
			props[name] = schema
			if hasTrait(member, "smithy.api#required") {
				required = append(required, name)
			}
		}
		out := map[string]any{"type": "object"}
		if len(props) > 0 {
			out["properties"] = props
		}
		if len(required) > 0 {
			out["required"] = required
		}
		return out, nil
	case "list", "set":
		member := mapValueOrNil(shape["member"])
		items, err := c.memberSchema(member)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": items}, nil
	case "map":
		value := mapValueOrNil(shape["value"])
		additional, err := c.memberSchema(value)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "object", "additionalProperties": additional}, nil
	case "union":
		var oneOf []any
		members := mapValueOrNil(shape["members"])
		for _, name := range sortedKeys(members) {
			memberSchema, err := c.memberSchema(mapValueOrNil(members[name]))
			if err != nil {
				return nil, err
			}
			oneOf = append(oneOf, memberSchema)
		}
		return map[string]any{"oneOf": oneOf}, nil
	case "enum":
		return map[string]any{"type": "string"}, nil
	default:
		if schema := primitiveSchema(stringValue(shape["type"])); schema != nil {
			return schema, nil
		}
		return nil, fmt.Errorf("shape %q has unsupported type %q", id, stringValue(shape["type"]))
	}
}

func (c *converter) memberSchema(member map[string]any) (map[string]any, error) {
	target := targetValue(member)
	if target == "" {
		return map[string]any{"type": "object"}, nil
	}
	if _, ok := c.shapes[target]; !ok {
		if schema := preludeSchema(target); schema != nil {
			return schema, nil
		}
		return nil, fmt.Errorf("shape references missing target %q", target)
	}
	schema := c.refForTarget(target)
	if doc := smithyDocumentation(member); doc != "" {
		schema = cloneMap(schema)
		schema["description"] = doc
	}
	return schema, nil
}

func (c *converter) refForTarget(target string) map[string]any {
	if schema := preludeSchema(target); schema != nil {
		return schema
	}
	name := c.componentNames[target]
	if name == "" {
		name = sanitizeIdentifier(localName(target))
	}
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

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

func (c *converter) serviceTitle() string {
	return firstNonEmpty(
		stringValue(traits(c.service)["smithy.api#title"]),
		stringValue(mapValueOrNil(traits(c.service)["aws.api#service"])["sdkId"]),
		localName(c.serviceID),
		"AWS Service",
	)
}

func (c *converter) awsServiceID() string {
	return stringValue(mapValueOrNil(traits(c.service)["aws.api#service"])["sdkId"])
}

func (c *converter) awsEndpointPrefix() string {
	return stringValue(mapValueOrNil(traits(c.service)["aws.api#service"])["endpointPrefix"])
}

func (c *converter) awsSigningName() string {
	return stringValue(mapValueOrNil(traits(c.service)["aws.auth#sigv4"])["name"])
}

func (c *converter) server() map[string]any {
	prefix := firstNonEmpty(c.awsEndpointPrefix(), c.awsSigningName(), strings.ToLower(c.awsServiceID()))
	if prefix == "" {
		prefix = "service"
	}
	return map[string]any{
		"url": fmt.Sprintf("https://%s.{region}.amazonaws.com", prefix),
		"variables": map[string]any{
			"region": map[string]any{
				"default":     "us-east-1",
				"description": "AWS region placeholder for derived Smithy metadata.",
			},
		},
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

func hasParameter(params []any, in, name string) bool {
	for _, param := range params {
		m := mapValueOrNil(param)
		if stringValue(m["in"]) == in && stringValue(m["name"]) == name {
			return true
		}
	}
	return false
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
