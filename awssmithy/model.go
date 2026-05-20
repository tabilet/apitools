package awssmithy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	ProtocolRestJSON1 = "restJson1"
	ProtocolRestXML   = "restXml"
	ProtocolAWSQuery  = "awsQuery"
	ProtocolEC2Query  = "ec2Query"
	ProtocolAWSJSON10 = "awsJson1_0"
	ProtocolAWSJSON11 = "awsJson1_1"
)

var supportedAWSProtocols = []string{
	ProtocolRestJSON1,
	ProtocolRestXML,
	ProtocolAWSQuery,
	ProtocolEC2Query,
	ProtocolAWSJSON10,
	ProtocolAWSJSON11,
}

// Model is a parsed AWS Smithy JSON service model. It preserves Smithy
// protocol metadata directly instead of treating the service as OpenAPI.
type Model struct {
	SmithyVersion  string
	Shapes         map[string]*Shape
	ServiceID      string
	Service        *Shape
	Protocol       string
	Version        string
	Title          string
	Description    string
	AWSServiceID   string
	EndpointPrefix string
	SigningName    string
	Operations     []*Operation
}

// Shape is a normalized Smithy shape record with its original JSON object
// retained for callers that need traits not modeled by this package yet.
type Shape struct {
	ID     string
	Type   string
	Traits map[string]any
	Raw    map[string]any
}

// Operation is a protocol-aware Smithy operation plan suitable for downstream
// generators and serializers.
type Operation struct {
	ID                string
	Name              string
	Input             string
	Output            string
	Method            string
	URI               string
	Path              string
	Code              int
	QueryLiterals     map[string]string
	GreedyLabels      []string
	InputBindings     []*MemberBinding
	UnboundInput      []*MemberBinding
	Payload           *MemberBinding
	OutputBindings    []*MemberBinding
	StaticQuery       map[string]string
	StaticHeaders     map[string]string
	StaticPayload     map[string]string
	RequestMediaType  string
	ResponseMediaType string
}

// MemberBinding describes how a Smithy structure member participates in an
// operation request or response.
type MemberBinding struct {
	MemberName string
	Target     string
	Location   string
	WireName   string
	Required   bool
	Traits     map[string]any
}

// Parse parses an AWS Smithy JSON service model into native Smithy metadata.
func Parse(data []byte) (*Model, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse smithy document: %w", err)
	}
	return ParseMap(raw)
}

// ParseMap parses an already-decoded AWS Smithy JSON service model into native
// Smithy metadata.
func ParseMap(raw map[string]any) (*Model, error) {
	p := &modelParser{raw: raw}
	return p.parse()
}

// OperationByName returns an operation by local Smithy operation name.
func (m *Model) OperationByName(name string) (*Operation, bool) {
	if m == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	for _, op := range m.Operations {
		if op != nil && (op.Name == name || op.ID == name) {
			return op, true
		}
	}
	return nil, false
}

// Shape returns the parsed shape with the given absolute Smithy shape ID.
func (m *Model) Shape(id string) (*Shape, bool) {
	if m == nil {
		return nil, false
	}
	shape, ok := m.Shapes[id]
	return shape, ok
}

func (m *Model) ServerURL() string {
	if m == nil {
		return ""
	}
	prefix := firstNonEmpty(m.EndpointPrefix, m.SigningName, strings.ToLower(m.AWSServiceID))
	if prefix == "" {
		prefix = "service"
	}
	return fmt.Sprintf("https://%s.{region}.amazonaws.com", prefix)
}

type modelParser struct {
	raw       map[string]any
	rawShapes map[string]any
	model     *Model
}

func (p *modelParser) parse() (*Model, error) {
	if p.raw == nil {
		return nil, fmt.Errorf("smithy document root must be an object")
	}
	if got := stringValue(p.raw["smithy"]); got != "2.0" {
		return nil, fmt.Errorf("unsupported smithy version %q", got)
	}
	shapes, ok, err := requiredMapField(p.raw, "shapes", "smithy document")
	if err != nil {
		return nil, err
	}
	if !ok || len(shapes) == 0 {
		return nil, fmt.Errorf("smithy document missing shapes")
	}
	p.rawShapes = shapes
	model := &Model{
		SmithyVersion: "2.0",
		Shapes:        make(map[string]*Shape, len(shapes)),
	}
	for _, id := range sortedKeys(shapes) {
		rawShape := mapValueOrNil(shapes[id])
		model.Shapes[id] = &Shape{
			ID:     id,
			Type:   stringValue(rawShape["type"]),
			Traits: traits(rawShape),
			Raw:    rawShape,
		}
	}
	p.model = model
	if err := p.selectService(); err != nil {
		return nil, err
	}
	ops, err := p.serviceOperations()
	if err != nil {
		return nil, err
	}
	model.Operations = ops
	return model, nil
}

func (p *modelParser) selectService() error {
	var serviceIDs []string
	for _, id := range sortedKeys(p.rawShapes) {
		shape := p.model.Shapes[id]
		if shape != nil && shape.Type == "service" {
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
	service := p.model.Shapes[serviceIDs[0]]
	protocol := serviceProtocolFromTraits(service.Traits)
	if protocol == "" {
		return fmt.Errorf("service %q uses unsupported or missing AWS Smithy protocol", serviceIDs[0])
	}
	awsService := mapValueOrNil(service.Traits["aws.api#service"])
	p.model.ServiceID = service.ID
	p.model.Service = service
	p.model.Protocol = protocol
	p.model.Version = stringValue(service.Raw["version"])
	p.model.Title = firstNonEmpty(
		stringValue(service.Traits["smithy.api#title"]),
		stringValue(awsService["sdkId"]),
		localName(service.ID),
		"AWS Service",
	)
	p.model.Description = smithyDocumentation(service.Raw)
	p.model.AWSServiceID = stringValue(awsService["sdkId"])
	p.model.EndpointPrefix = stringValue(awsService["endpointPrefix"])
	p.model.SigningName = stringValue(mapValueOrNil(service.Traits["aws.auth#sigv4"])["name"])
	return nil
}

func serviceProtocolFromTraits(traits map[string]any) string {
	for _, protocol := range supportedAWSProtocols {
		if _, ok := traits["aws.protocols#"+protocol]; ok {
			return protocol
		}
	}
	return ""
}

func (p *modelParser) serviceOperations() ([]*Operation, error) {
	ids, err := serviceOperationIDs(p.rawShapes, p.model.Service.Raw)
	if err != nil {
		return nil, err
	}
	out := make([]*Operation, 0, len(ids))
	for _, id := range ids {
		op, err := p.parseOperation(id)
		if err != nil {
			return nil, err
		}
		out = append(out, op)
	}
	return out, nil
}

func serviceOperationIDs(shapes map[string]any, service map[string]any) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	var addRef func(ref any) error
	addRef = func(ref any) error {
		target := targetValue(ref)
		if target == "" {
			return nil
		}
		shape := mapValueOrNil(shapes[target])
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
	for _, ref := range sliceValueOrNil(service["operations"]) {
		if err := addRef(ref); err != nil {
			return nil, err
		}
	}
	var walkResource func(string) error
	walkResource = func(id string) error {
		resource := mapValueOrNil(shapes[id])
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
	for _, ref := range sliceValueOrNil(service["resources"]) {
		if err := walkResource(targetValue(ref)); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return localName(out[i]) < localName(out[j])
	})
	return out, nil
}

func (p *modelParser) parseOperation(id string) (*Operation, error) {
	raw := mapValueOrNil(p.rawShapes[id])
	if raw == nil {
		return nil, fmt.Errorf("operation %q is missing", id)
	}
	input, err := p.operationTarget(id, raw, "input")
	if err != nil {
		return nil, err
	}
	output, err := p.operationTarget(id, raw, "output")
	if err != nil {
		return nil, err
	}
	httpTrait := mapValueOrNil(traits(raw)["smithy.api#http"])
	method := strings.ToUpper(stringValue(httpTrait["method"]))
	uri := firstNonEmpty(stringValue(httpTrait["uri"]), "/")
	code := intValue(httpTrait["code"], 200)
	if p.model.Protocol == ProtocolAWSQuery || p.model.Protocol == ProtocolEC2Query ||
		p.model.Protocol == ProtocolAWSJSON10 || p.model.Protocol == ProtocolAWSJSON11 {
		method = "POST"
		uri = "/"
	} else if len(httpTrait) == 0 {
		return nil, fmt.Errorf("operation %q missing smithy.api#http trait for %s", id, p.model.Protocol)
	}
	if method == "" {
		return nil, fmt.Errorf("operation %q missing HTTP method", id)
	}
	httpPath, queryLiterals, greedyLabels := splitSmithyURI(uri)
	op := &Operation{
		ID:                id,
		Name:              localName(id),
		Input:             input,
		Output:            output,
		Method:            method,
		URI:               uri,
		Path:              operationPathKey(httpPath, queryLiterals),
		Code:              code,
		QueryLiterals:     queryLiterals,
		GreedyLabels:      greedyLabels,
		StaticQuery:       cloneStringMap(queryLiterals),
		RequestMediaType:  p.requestMediaType(input),
		ResponseMediaType: p.responseMediaType(output),
	}
	if p.model.Protocol == ProtocolAWSQuery || p.model.Protocol == ProtocolEC2Query {
		op.Path = "/?Action=" + op.Name
		op.StaticPayload = map[string]string{
			"Action":  op.Name,
			"Version": p.model.Version,
		}
		op.RequestMediaType = "application/x-www-form-urlencoded"
		op.ResponseMediaType = "application/xml"
	}
	if p.model.Protocol == ProtocolAWSJSON10 || p.model.Protocol == ProtocolAWSJSON11 {
		op.StaticHeaders = map[string]string{
			"X-Amz-Target": p.jsonTarget(op.Name),
		}
	}
	if err := p.bindInput(op); err != nil {
		return nil, err
	}
	if err := p.bindOutput(op); err != nil {
		return nil, err
	}
	return op, nil
}

func (p *modelParser) operationTarget(operationID string, operation map[string]any, key string) (string, error) {
	target := targetValue(operation[key])
	if target == "" || target == "smithy.api#Unit" {
		return target, nil
	}
	if _, ok := p.rawShapes[target]; !ok && preludeSchema(target) == nil {
		return "", fmt.Errorf("operation %q references missing %s shape %q", localName(operationID), key, target)
	}
	return target, nil
}

func (p *modelParser) bindInput(op *Operation) error {
	if op.Input == "" || op.Input == "smithy.api#Unit" {
		return nil
	}
	input := mapValueOrNil(p.rawShapes[op.Input])
	if stringValue(input["type"]) != "structure" {
		return fmt.Errorf("input shape %q must be a structure", op.Input)
	}
	members := mapValueOrNil(input["members"])
	for _, name := range sortedKeys(members) {
		member := mapValueOrNil(members[name])
		binding := memberBinding(name, member)
		switch binding.Location {
		case "path", "query", "queryParams", "header", "prefixHeaders":
			op.InputBindings = append(op.InputBindings, binding)
		case "payload":
			op.Payload = binding
		default:
			op.UnboundInput = append(op.UnboundInput, binding)
		}
	}
	for _, label := range pathLabels(op.Path) {
		if !hasBinding(op.InputBindings, "path", label) {
			op.InputBindings = append(op.InputBindings, &MemberBinding{
				MemberName: label,
				Target:     "smithy.api#String",
				Location:   "path",
				WireName:   label,
				Required:   true,
			})
		}
	}
	for name := range op.QueryLiterals {
		if !hasBinding(op.InputBindings, "query", name) {
			op.InputBindings = append(op.InputBindings, &MemberBinding{
				MemberName: name,
				Target:     "smithy.api#String",
				Location:   "query",
				WireName:   name,
				Required:   true,
			})
		}
	}
	return nil
}

func (p *modelParser) bindOutput(op *Operation) error {
	if op.Output == "" || op.Output == "smithy.api#Unit" {
		return nil
	}
	output := mapValueOrNil(p.rawShapes[op.Output])
	if stringValue(output["type"]) != "structure" {
		return nil
	}
	members := mapValueOrNil(output["members"])
	for _, name := range sortedKeys(members) {
		member := mapValueOrNil(members[name])
		binding := memberBinding(name, member)
		switch binding.Location {
		case "header", "prefixHeaders", "payload":
			op.OutputBindings = append(op.OutputBindings, binding)
		case "":
			op.OutputBindings = append(op.OutputBindings, binding)
		}
	}
	return nil
}

func memberBinding(name string, member map[string]any) *MemberBinding {
	mtraits := traits(member)
	binding := &MemberBinding{
		MemberName: name,
		Target:     targetValue(member),
		WireName:   name,
		Required:   hasTrait(member, "smithy.api#required"),
		Traits:     mtraits,
	}
	switch {
	case hasTrait(member, "smithy.api#httpLabel"):
		binding.Location = "path"
		binding.Required = true
	case hasTrait(member, "smithy.api#httpQuery"):
		binding.Location = "query"
		if value := stringValue(mtraits["smithy.api#httpQuery"]); value != "" {
			binding.WireName = value
		}
	case hasTrait(member, "smithy.api#httpQueryParams"):
		binding.Location = "queryParams"
	case hasTrait(member, "smithy.api#httpHeader"):
		binding.Location = "header"
		if value := stringValue(mtraits["smithy.api#httpHeader"]); value != "" {
			binding.WireName = value
		}
	case hasTrait(member, "smithy.api#httpPrefixHeaders"):
		binding.Location = "prefixHeaders"
		if value := stringValue(mtraits["smithy.api#httpPrefixHeaders"]); value != "" {
			binding.WireName = value
		}
	case hasTrait(member, "smithy.api#httpPayload"):
		binding.Location = "payload"
	case hasTrait(member, "smithy.api#httpResponseCode"):
		binding.Location = "responseCode"
	}
	return binding
}

func (p *modelParser) requestMediaType(input string) string {
	if p.model.Protocol == ProtocolAWSJSON10 {
		return "application/x-amz-json-1.0"
	}
	if p.model.Protocol == ProtocolAWSJSON11 {
		return "application/x-amz-json-1.1"
	}
	if input != "" && input != "smithy.api#Unit" {
		raw := mapValueOrNil(p.rawShapes[input])
		for _, memberValue := range mapValueOrNil(raw["members"]) {
			member := mapValueOrNil(memberValue)
			if hasTrait(member, "smithy.api#httpPayload") && isBlobTarget(p.rawShapes, targetValue(member)) {
				return "application/octet-stream"
			}
		}
	}
	if p.model.Protocol == ProtocolRestXML {
		return "application/xml"
	}
	return "application/json"
}

func (p *modelParser) responseMediaType(output string) string {
	if p.model.Protocol == ProtocolRestXML {
		return "application/xml"
	}
	if p.model.Protocol == ProtocolAWSJSON10 {
		return "application/x-amz-json-1.0"
	}
	if p.model.Protocol == ProtocolAWSJSON11 {
		return "application/x-amz-json-1.1"
	}
	if output != "" && output != "smithy.api#Unit" && isBlobTarget(p.rawShapes, output) {
		return "application/octet-stream"
	}
	return "application/json"
}

func (p *modelParser) jsonTarget(operation string) string {
	return localName(p.model.ServiceID) + "." + operation
}

func isBlobTarget(shapes map[string]any, target string) bool {
	if target == "smithy.api#Blob" || target == "smithy.api#PrimitiveBlob" {
		return true
	}
	return stringValue(mapValueOrNil(shapes[target])["type"]) == "blob"
}

func hasBinding(bindings []*MemberBinding, location, wireName string) bool {
	for _, binding := range bindings {
		if binding != nil && binding.Location == location && binding.WireName == wireName {
			return true
		}
	}
	return false
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
