package grpcproto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/OpenUdon/apitools/internal/sourceguard"
)

// DetectLocalArtifact classifies local gRPC/protobuf artifact bytes without
// server reflection, endpoint probes, or network access.
func DetectLocalArtifact(data []byte, path string) (ArtifactDetection, bool) {
	model, err := Parse(data)
	if err != nil {
		return ArtifactDetection{}, false
	}
	reason := "gRPC/protobuf source metadata parsed locally"
	format := grpcProtoFormatForPath(path, model.SourceKind)
	switch model.SourceKind {
	case ArtifactKindProtoFile:
		reason = "protobuf .proto source file"
	case ArtifactKindDescriptorJSON:
		reason = "protobuf descriptor JSON artifact"
	case ArtifactKindDescriptorSet:
		reason = "protobuf FileDescriptorSet artifact"
	case ArtifactKindGeneratedDescriptor:
		reason = "reviewed generated protobuf descriptor artifact"
	}
	return ArtifactDetection{Kind: model.SourceKind, Format: format, Path: strings.TrimSpace(path), Reason: reason}, true
}

// Parse decodes a .proto source file, descriptor JSON artifact, or binary
// FileDescriptorSet into metadata. It performs no RPC execution, server
// reflection, channel creation, TLS negotiation, endpoint probing, or
// credential lookup.
func Parse(data []byte) (*Model, error) {
	if err := sourceguard.CheckDocument("grpcproto", data); err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("grpcproto: empty document")
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		if model, err := parseDescriptorJSON(trimmed); err == nil {
			return model, nil
		} else if json.Valid(trimmed) {
			return nil, err
		}
	}
	if looksLikeProtoText(trimmed) {
		if model, err := parseProtoText(string(trimmed)); err == nil {
			return model, nil
		} else {
			return nil, err
		}
	}
	if model, err := parseBinaryDescriptorSet(data); err == nil {
		return model, nil
	}
	if model, err := parseProtoText(string(trimmed)); err == nil {
		return model, nil
	}
	return nil, fmt.Errorf("grpcproto: document does not contain protobuf source metadata")
}

func parseDescriptorJSON(data []byte) (*Model, error) {
	if err := sourceguard.CheckJSON("grpcproto", data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("grpcproto: decode descriptor JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("grpcproto: descriptor JSON contains trailing content")
	}
	files := descriptorJSONFiles(root)
	if len(files) == 0 {
		return nil, fmt.Errorf("grpcproto: descriptor JSON missing file descriptors")
	}
	model := &Model{SourceKind: ArtifactKindDescriptorJSON}
	for _, fileRoot := range files {
		file := parseDescriptorJSONFile(fileRoot)
		if file != nil {
			model.Files = append(model.Files, file)
		}
	}
	if len(model.Files) == 0 {
		return nil, fmt.Errorf("grpcproto: descriptor JSON has no parseable file descriptors")
	}
	finalizeModel(model)
	if !modelHasMetadata(model) {
		return nil, fmt.Errorf("grpcproto: descriptor JSON has no services, messages, or enums")
	}
	return model, nil
}

func descriptorJSONFiles(root any) []map[string]any {
	switch typed := root.(type) {
	case []any:
		return objectsFromArray(typed)
	case map[string]any:
		for _, key := range []string{"file", "files", "fileDescriptorProto", "file_descriptor_proto"} {
			if values := objectsFromArray(arrayValue(typed[key])); len(values) > 0 {
				return values
			}
		}
		if _, ok := typed["service"]; ok || typed["package"] != nil || typed["messageType"] != nil || typed["message_type"] != nil {
			return []map[string]any{typed}
		}
	}
	return nil
}

func parseDescriptorJSONFile(root map[string]any) *File {
	file := &File{
		Name:    stringField(root, "name"),
		Package: stringField(root, "package"),
		Syntax:  stringField(root, "syntax"),
		Imports: stringsFromArray(firstArray(root, "dependency", "dependencies", "import", "imports")),
	}
	for _, messageRoot := range objectsFromArray(firstArray(root, "messageType", "message_type", "messages")) {
		if message := parseDescriptorJSONMessage(messageRoot, file.Package, ""); message != nil {
			file.Messages = append(file.Messages, message)
		}
	}
	for _, enumRoot := range objectsFromArray(firstArray(root, "enumType", "enum_type", "enums")) {
		if enum := parseDescriptorJSONEnum(enumRoot, file.Package, ""); enum != nil {
			file.Enums = append(file.Enums, enum)
		}
	}
	for _, serviceRoot := range objectsFromArray(firstArray(root, "service", "services")) {
		if service := parseDescriptorJSONService(serviceRoot, file.Package); service != nil {
			file.Services = append(file.Services, service)
		}
	}
	return file
}

func parseDescriptorJSONService(root map[string]any, packageName string) *Service {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	service := &Service{Name: name, Package: packageName, FullName: joinName(packageName, name)}
	for _, methodRoot := range objectsFromArray(firstArray(root, "method", "methods")) {
		method := parseDescriptorJSONMethod(methodRoot, service)
		if method != nil {
			service.Methods = append(service.Methods, method)
		}
	}
	return service
}

func parseDescriptorJSONMethod(root map[string]any, service *Service) *Method {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	method := &Method{
		Name:            name,
		ServiceName:     service.FullName,
		Package:         service.Package,
		RequestType:     normalizeTypeName(firstString(root, "inputType", "input_type", "requestType", "request_type")),
		ResponseType:    normalizeTypeName(firstString(root, "outputType", "output_type", "responseType", "response_type")),
		ClientStreaming: boolField(root, "clientStreaming") || boolField(root, "client_streaming"),
		ServerStreaming: boolField(root, "serverStreaming") || boolField(root, "server_streaming"),
	}
	return method
}

func parseDescriptorJSONMessage(root map[string]any, packageName, parent string) *Message {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	fullName := joinName(packageName, joinName(parent, name))
	message := &Message{Name: name, FullName: fullName}
	for _, fieldRoot := range objectsFromArray(firstArray(root, "field", "fields")) {
		if field := parseDescriptorJSONField(fieldRoot, fullName); field != nil {
			message.Fields = append(message.Fields, field)
		}
	}
	for _, nestedRoot := range objectsFromArray(firstArray(root, "nestedType", "nested_type", "messages")) {
		if nested := parseDescriptorJSONMessage(nestedRoot, packageName, joinName(parent, name)); nested != nil {
			message.Messages = append(message.Messages, nested)
		}
	}
	for _, enumRoot := range objectsFromArray(firstArray(root, "enumType", "enum_type", "enums")) {
		if enum := parseDescriptorJSONEnum(enumRoot, packageName, joinName(parent, name)); enum != nil {
			message.Enums = append(message.Enums, enum)
		}
	}
	return message
}

func parseDescriptorJSONField(root map[string]any, parentFullName string) *Field {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	label := protoLabelName(root["label"])
	field := &Field{
		Name:     name,
		Number:   intField(root, "number"),
		Label:    label,
		Type:     protoTypeName(root["type"]),
		TypeName: normalizeTypeName(firstString(root, "typeName", "type_name")),
		JSONName: firstString(root, "jsonName", "json_name"),
		Repeated: label == "repeated",
		Required: label == "required",
	}
	field.Selector = MessageSelector(parentFullName) + "/fields/" + escapeJSONPointer(name)
	return field
}

func parseDescriptorJSONEnum(root map[string]any, packageName, parent string) *Enum {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	enum := &Enum{Name: name, FullName: joinName(packageName, joinName(parent, name))}
	for _, valueRoot := range objectsFromArray(firstArray(root, "value", "values")) {
		valueName := stringField(valueRoot, "name")
		if valueName == "" {
			continue
		}
		enum.Values = append(enum.Values, &EnumValue{Name: valueName, Number: intField(valueRoot, "number")})
	}
	return enum
}

func parseProtoText(input string) (*Model, error) {
	tokens, err := lexProto(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("grpcproto: empty proto source")
	}
	parser := protoParser{tokens: tokens}
	file := parser.parseFile()
	if parser.err != nil {
		return nil, parser.err
	}
	if len(file.Services) == 0 && len(file.Messages) == 0 && len(file.Enums) == 0 {
		return nil, fmt.Errorf("grpcproto: proto source has no services, messages, or enums")
	}
	model := &Model{SourceKind: ArtifactKindProtoFile, Files: []*File{file}}
	finalizeModel(model)
	return model, nil
}

type protoTokenKind string

const (
	protoTokenName   protoTokenKind = "name"
	protoTokenString protoTokenKind = "string"
	protoTokenNumber protoTokenKind = "number"
	protoTokenPunct  protoTokenKind = "punct"
)

type protoToken struct {
	kind  protoTokenKind
	value string
}

func lexProto(input string) ([]protoToken, error) {
	var tokens []protoToken
	depth := 0
	appendToken := func(token protoToken) error {
		tokens = append(tokens, token)
		if len(tokens) > sourceguard.MaxWorkItems {
			return fmt.Errorf("grpcproto: token count exceeds maximum %d", sourceguard.MaxWorkItems)
		}
		switch token.value {
		case "{", "[", "(", "<":
			depth++
			if depth > sourceguard.MaxNestingDepth {
				return fmt.Errorf("grpcproto: nesting exceeds maximum depth %d", sourceguard.MaxNestingDepth)
			}
		case "}", "]", ")", ">":
			if depth > 0 {
				depth--
			}
		}
		return nil
	}
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if strings.HasPrefix(input[i:], "//") {
			for i < len(input) && input[i] != '\n' && input[i] != '\r' {
				i++
			}
			continue
		}
		if strings.HasPrefix(input[i:], "/*") {
			end := strings.Index(input[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("grpcproto: unterminated block comment")
			}
			i += end + 4
			continue
		}
		if isNameStart(r) {
			start := i
			i += size
			for i < len(input) {
				next, nextSize := utf8.DecodeRuneInString(input[i:])
				if !isNameContinue(next) {
					break
				}
				i += nextSize
			}
			if err := appendToken(protoToken{kind: protoTokenName, value: input[start:i]}); err != nil {
				return nil, err
			}
			continue
		}
		if isProtoNumberStart(input, i) {
			start := i
			i++
			for i < len(input) && isProtoNumberContinue(input[i]) {
				i++
			}
			if err := appendToken(protoToken{kind: protoTokenNumber, value: input[start:i]}); err != nil {
				return nil, err
			}
			continue
		}
		if input[i] == '"' || input[i] == '\'' {
			value, next, err := readProtoString(input, i)
			if err != nil {
				return nil, err
			}
			if err := appendToken(protoToken{kind: protoTokenString, value: value}); err != nil {
				return nil, err
			}
			i = next
			continue
		}
		switch input[i] {
		case '{', '}', '(', ')', '[', ']', '<', '>', ';', ':', '=', ',', '.', '/', '-':
			if err := appendToken(protoToken{kind: protoTokenPunct, value: input[i : i+1]}); err != nil {
				return nil, err
			}
			i++
		default:
			return nil, fmt.Errorf("grpcproto: unexpected character %q", input[i])
		}
	}
	return tokens, nil
}

func readProtoString(input string, start int) (string, int, error) {
	quote := input[start]
	var b strings.Builder
	for i := start + 1; i < len(input); i++ {
		if input[i] == '\\' {
			if i+1 >= len(input) {
				return "", 0, fmt.Errorf("grpcproto: unterminated string escape")
			}
			b.WriteByte(input[i+1])
			i++
			continue
		}
		if input[i] == quote {
			return b.String(), i + 1, nil
		}
		b.WriteByte(input[i])
	}
	return "", 0, fmt.Errorf("grpcproto: unterminated string")
}

func isNameStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func isNameContinue(r rune) bool {
	return isNameStart(r) || r >= '0' && r <= '9'
}

func isProtoNumberStart(input string, i int) bool {
	return input[i] >= '0' && input[i] <= '9'
}

func isProtoNumberContinue(b byte) bool {
	return b >= '0' && b <= '9' || b == '.' || b == 'x' || b == 'X' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

type protoParser struct {
	tokens []protoToken
	pos    int
	err    error
}

func (p *protoParser) parseFile() *File {
	file := &File{}
	for !p.done() {
		switch p.peek().value {
		case "syntax":
			p.advance()
			if p.consume("=") && p.peekKind(protoTokenString) {
				file.Syntax = p.advance().value
			}
			p.skipStatement()
		case "package":
			p.advance()
			file.Package = p.parseQualifiedName()
			p.skipStatement()
		case "import":
			p.advance()
			if p.peekValue("public") || p.peekValue("weak") {
				p.advance()
			}
			if p.peekKind(protoTokenString) {
				file.Imports = append(file.Imports, p.advance().value)
			}
			p.skipStatement()
		case "message":
			p.advance()
			if message := p.parseMessage(file.Package, "", 0); message != nil {
				file.Messages = append(file.Messages, message)
			}
		case "enum":
			p.advance()
			if enum := p.parseEnum(file.Package, ""); enum != nil {
				file.Enums = append(file.Enums, enum)
			}
		case "service":
			p.advance()
			if service := p.parseService(file.Package); service != nil {
				file.Services = append(file.Services, service)
			}
		default:
			p.skipStatementOrBlock()
		}
	}
	return file
}

func (p *protoParser) parseService(packageName string) *Service {
	name := p.consumeName()
	if name == "" {
		return nil
	}
	service := &Service{Name: name, Package: packageName, FullName: joinName(packageName, name)}
	if !p.consume("{") {
		return service
	}
	for !p.done() && !p.consume("}") {
		switch p.peek().value {
		case "rpc":
			p.advance()
			if method := p.parseRPC(service); method != nil {
				service.Methods = append(service.Methods, method)
			}
		default:
			p.skipStatementOrBlock()
		}
	}
	return service
}

func (p *protoParser) parseRPC(service *Service) *Method {
	name := p.consumeName()
	if name == "" {
		p.skipStatementOrBlock()
		return nil
	}
	method := &Method{Name: name, ServiceName: service.FullName, Package: service.Package}
	if p.consume("(") {
		if p.consume("stream") {
			method.ClientStreaming = true
		}
		method.RequestType = normalizeTypeName(p.parseQualifiedName())
		p.consume(")")
	}
	if p.consume("returns") && p.consume("(") {
		if p.consume("stream") {
			method.ServerStreaming = true
		}
		method.ResponseType = normalizeTypeName(p.parseQualifiedName())
		p.consume(")")
	}
	if p.peekValue("{") {
		p.skipBalanced("{", "}")
	} else {
		p.skipStatement()
	}
	return method
}

func (p *protoParser) parseMessage(packageName, parent string, depth int) *Message {
	if depth >= sourceguard.MaxNestingDepth {
		p.err = fmt.Errorf("grpcproto: message nesting exceeds maximum depth %d", sourceguard.MaxNestingDepth)
		return nil
	}
	name := p.consumeName()
	if name == "" {
		return nil
	}
	fullName := joinName(packageName, joinName(parent, name))
	message := &Message{Name: name, FullName: fullName}
	if !p.consume("{") {
		return message
	}
	for !p.done() && !p.consume("}") {
		switch p.peek().value {
		case "message":
			p.advance()
			if nested := p.parseMessage(packageName, joinName(parent, name), depth+1); nested != nil {
				message.Messages = append(message.Messages, nested)
			}
		case "enum":
			p.advance()
			if enum := p.parseEnum(packageName, joinName(parent, name)); enum != nil {
				message.Enums = append(message.Enums, enum)
			}
		case "reserved", "extensions", "option":
			p.skipStatementOrBlock()
		case "oneof":
			p.advance()
			message.Fields = append(message.Fields, p.parseOneofFields(fullName)...)
		default:
			if field := p.parseField(fullName); field != nil {
				message.Fields = append(message.Fields, field)
			} else {
				p.skipStatementOrBlock()
			}
		}
	}
	return message
}

func (p *protoParser) parseOneofFields(parentFullName string) []*Field {
	p.consumeName()
	if !p.consume("{") {
		return nil
	}
	var fields []*Field
	for !p.done() && !p.consume("}") {
		if p.peekValue("option") {
			p.skipStatementOrBlock()
			continue
		}
		if field := p.parseField(parentFullName); field != nil {
			fields = append(fields, field)
		} else {
			p.skipStatementOrBlock()
		}
	}
	return fields
}

func (p *protoParser) parseField(parentFullName string) *Field {
	label := ""
	if p.peekValue("optional") || p.peekValue("required") || p.peekValue("repeated") {
		label = p.advance().value
	}
	typeName := ""
	fieldType := ""
	if p.peekValue("map") {
		fieldType = "map"
		p.advance()
		if p.peekValue("<") {
			p.skipBalanced("<", ">")
		}
	} else {
		fieldType = normalizeTypeName(p.parseQualifiedName())
	}
	if fieldType == "" {
		return nil
	}
	name := p.consumeName()
	if name == "" || !p.consume("=") {
		return nil
	}
	numberToken := p.advance()
	if numberToken.kind != protoTokenNumber && numberToken.kind != protoTokenName {
		return nil
	}
	number, _ := strconv.Atoi(numberToken.value)
	if strings.Contains(fieldType, ".") || len(fieldType) > 0 && unicode.IsUpper(rune(fieldType[0])) {
		typeName = fieldType
	}
	field := &Field{
		Name:     name,
		Number:   number,
		Label:    label,
		Type:     fieldType,
		TypeName: typeName,
		Repeated: label == "repeated",
		Required: label == "required",
	}
	field.Selector = MessageSelector(parentFullName) + "/fields/" + escapeJSONPointer(name)
	if p.peekValue("[") {
		p.skipBalanced("[", "]")
	}
	p.skipStatement()
	return field
}

func (p *protoParser) parseEnum(packageName, parent string) *Enum {
	name := p.consumeName()
	if name == "" {
		return nil
	}
	enum := &Enum{Name: name, FullName: joinName(packageName, joinName(parent, name))}
	if !p.consume("{") {
		return enum
	}
	for !p.done() && !p.consume("}") {
		if p.peekValue("option") || p.peekValue("reserved") {
			p.skipStatementOrBlock()
			continue
		}
		valueName := p.consumeName()
		if valueName == "" {
			p.skipStatementOrBlock()
			continue
		}
		if !p.consume("=") {
			p.skipStatementOrBlock()
			continue
		}
		numberToken := p.advance()
		number, _ := strconv.Atoi(numberToken.value)
		enum.Values = append(enum.Values, &EnumValue{Name: valueName, Number: number})
		p.skipStatement()
	}
	return enum
}

func (p *protoParser) parseQualifiedName() string {
	var parts []string
	leadingDot := p.consume(".")
	for !p.done() {
		name := p.consumeName()
		if name == "" {
			break
		}
		parts = append(parts, name)
		if !p.consume(".") {
			break
		}
	}
	out := strings.Join(parts, ".")
	if leadingDot && out != "" {
		return "." + out
	}
	return out
}

func (p *protoParser) skipStatementOrBlock() {
	if p.peekValue("{") {
		p.skipBalanced("{", "}")
		return
	}
	p.skipStatement()
}

func (p *protoParser) skipStatement() {
	for !p.done() {
		if p.consume(";") {
			return
		}
		if p.peekValue("{") {
			p.skipBalanced("{", "}")
			return
		}
		if p.peekValue("}") {
			p.err = fmt.Errorf("grpcproto: unexpected closing brace")
			p.advance()
			return
		}
		p.advance()
	}
}

func (p *protoParser) skipBalanced(open, close string) {
	if !p.consume(open) {
		return
	}
	depth := 1
	for !p.done() && depth > 0 {
		tok := p.advance()
		switch tok.value {
		case open:
			depth++
		case close:
			depth--
		}
	}
	if depth > 0 && p.err == nil {
		p.err = fmt.Errorf("grpcproto: unterminated %s block", open)
	}
}

func (p *protoParser) consumeName() string {
	if !p.peekKind(protoTokenName) {
		return ""
	}
	return p.advance().value
}

func (p *protoParser) consume(value string) bool {
	if !p.peekValue(value) {
		return false
	}
	p.advance()
	return true
}

func (p *protoParser) peekKind(kind protoTokenKind) bool {
	return !p.done() && p.peek().kind == kind
}

func (p *protoParser) peekValue(value string) bool {
	return !p.done() && p.peek().value == value
}

func (p *protoParser) peek() protoToken {
	return p.tokens[p.pos]
}

func (p *protoParser) advance() protoToken {
	if p.done() {
		if p.err == nil {
			p.err = fmt.Errorf("grpcproto: unexpected end of source")
		}
		return protoToken{}
	}
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

func (p *protoParser) done() bool {
	return p.pos >= len(p.tokens)
}

func finalizeModel(model *Model) {
	for _, file := range model.Files {
		file.Selector = "#/files/" + escapeJSONPointer(firstNonEmpty(file.Name, file.Package, "proto"))
		for _, message := range file.Messages {
			finalizeMessage(message)
		}
		for _, enum := range file.Enums {
			enum.Selector = "#/enums/" + escapeJSONPointer(enum.FullName)
		}
		for _, service := range file.Services {
			if service.Package == "" {
				service.Package = file.Package
			}
			if service.FullName == "" {
				service.FullName = joinName(service.Package, service.Name)
			}
			service.Selector = ServiceSelector(service.FullName)
			for _, method := range service.Methods {
				method.Package = firstNonEmpty(method.Package, service.Package)
				method.ServiceName = service.FullName
				method.FullName = joinName(service.FullName, method.Name)
				method.SourceOperationID = service.FullName + "/" + method.Name
				method.Selector = MethodSelector(service.FullName, method.Name)
				method.SourceOperationRef = method.Selector
			}
		}
		sort.SliceStable(file.Services, func(i, j int) bool { return file.Services[i].FullName < file.Services[j].FullName })
		sort.SliceStable(file.Messages, func(i, j int) bool { return file.Messages[i].FullName < file.Messages[j].FullName })
	}
	sort.SliceStable(model.Files, func(i, j int) bool { return model.Files[i].Name < model.Files[j].Name })
}

func modelHasMetadata(model *Model) bool {
	for _, file := range model.Files {
		if len(file.Services) > 0 || len(file.Messages) > 0 || len(file.Enums) > 0 {
			return true
		}
	}
	return false
}

func finalizeMessage(message *Message) {
	if message == nil {
		return
	}
	message.Selector = MessageSelector(message.FullName)
	for _, field := range message.Fields {
		if field.Selector == "" {
			field.Selector = message.Selector + "/fields/" + escapeJSONPointer(field.Name)
		}
		if field.JSONName == "" {
			field.JSONName = lowerCamel(field.Name)
		}
	}
	for _, nested := range message.Messages {
		finalizeMessage(nested)
	}
	for _, enum := range message.Enums {
		enum.Selector = "#/enums/" + escapeJSONPointer(enum.FullName)
	}
}

func looksLikeProtoText(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))
	return strings.HasPrefix(trimmed, "syntax") ||
		strings.HasPrefix(trimmed, "package") ||
		strings.HasPrefix(trimmed, "message") ||
		strings.HasPrefix(trimmed, "service") ||
		strings.HasPrefix(trimmed, "enum") ||
		strings.Contains(trimmed, " service ") ||
		strings.Contains(trimmed, "\nservice ") ||
		strings.Contains(trimmed, " message ") ||
		strings.Contains(trimmed, "\nmessage ")
}

func grpcProtoFormatForPath(path string, kind ArtifactKind) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".proto":
		return "proto"
	case ".json":
		return "json"
	case ".pb", ".bin", ".desc", ".descriptor":
		return "binary"
	}
	switch kind {
	case ArtifactKindDescriptorJSON:
		return "json"
	case ArtifactKindDescriptorSet:
		return "binary"
	default:
		return "proto"
	}
}

func normalizeTypeName(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), ".")
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

func lowerCamel(value string) string {
	parts := strings.Split(value, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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

func arrayValue(value any) []any {
	out, _ := value.([]any)
	return out
}

func firstArray(root map[string]any, keys ...string) []any {
	for _, key := range keys {
		if values, ok := root[key].([]any); ok {
			return values
		}
	}
	return nil
}

func stringsFromArray(values []any) []string {
	var out []string
	for _, value := range values {
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return out
}

func firstString(root map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(root, key); value != "" {
			return value
		}
	}
	return ""
}

func stringField(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return strings.TrimSpace(value)
}

func boolField(root map[string]any, key string) bool {
	value, _ := root[key].(bool)
	return value
}

func intField(root map[string]any, key string) int {
	switch value := root[key].(type) {
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	case float64:
		return int(value)
	case string:
		i, _ := strconv.Atoi(value)
		return i
	default:
		return 0
	}
}

func protoLabelName(value any) string {
	switch typed := value.(type) {
	case string:
		switch strings.TrimPrefix(strings.ToUpper(typed), "LABEL_") {
		case "OPTIONAL":
			return "optional"
		case "REQUIRED":
			return "required"
		case "REPEATED":
			return "repeated"
		default:
			return strings.ToLower(typed)
		}
	case json.Number:
		i, _ := typed.Int64()
		return protoLabelNumberName(int(i))
	case float64:
		return protoLabelNumberName(int(typed))
	default:
		return ""
	}
}

func protoLabelNumberName(value int) string {
	switch value {
	case 1:
		return "optional"
	case 2:
		return "required"
	case 3:
		return "repeated"
	default:
		return ""
	}
}

func protoTypeName(value any) string {
	switch typed := value.(type) {
	case string:
		name := strings.TrimPrefix(strings.ToUpper(typed), "TYPE_")
		return strings.ToLower(name)
	case json.Number:
		i, _ := typed.Int64()
		return protoTypeNumberName(int(i))
	case float64:
		return protoTypeNumberName(int(typed))
	default:
		return ""
	}
}

func protoTypeNumberName(value int) string {
	switch value {
	case 1:
		return "double"
	case 2:
		return "float"
	case 3:
		return "int64"
	case 4:
		return "uint64"
	case 5:
		return "int32"
	case 6:
		return "fixed64"
	case 7:
		return "fixed32"
	case 8:
		return "bool"
	case 9:
		return "string"
	case 10:
		return "group"
	case 11:
		return "message"
	case 12:
		return "bytes"
	case 13:
		return "uint32"
	case 14:
		return "enum"
	case 15:
		return "sfixed32"
	case 16:
		return "sfixed64"
	case 17:
		return "sint32"
	case 18:
		return "sint64"
	default:
		return ""
	}
}
