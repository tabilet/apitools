package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DetectLocalArtifact classifies local GraphQL artifact bytes without opening
// sockets or consulting remote endpoints.
func DetectLocalArtifact(data []byte, path string) (ArtifactDetection, bool) {
	model, err := Parse(data)
	if err != nil {
		return ArtifactDetection{}, false
	}
	reason := "GraphQL source metadata parsed locally"
	switch model.SourceKind {
	case ArtifactKindIntrospectionJSON:
		reason = "GraphQL introspection __schema object"
	case ArtifactKindSDL:
		reason = "GraphQL SDL schema/type definitions"
	case ArtifactKindOperationDocument:
		reason = "GraphQL operation document"
	}
	return ArtifactDetection{
		Kind:   model.SourceKind,
		Format: graphQLFormatForPath(path, model.SourceKind),
		Path:   strings.TrimSpace(path),
		Reason: reason,
	}, true
}

// Parse decodes a GraphQL SDL, introspection JSON, or operation document into
// metadata. It does not execute operations, probe endpoints, resolve variables,
// or look up credentials.
func Parse(data []byte) (*Model, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("graphql: empty document")
	}
	if trimmed[0] == '{' {
		model, err := parseIntrospectionJSON(trimmed)
		if err == nil {
			return model, nil
		}
		if json.Valid(trimmed) {
			return nil, err
		}
	}
	model, err := parseGraphQLText(string(trimmed))
	if err != nil {
		return nil, err
	}
	return model, nil
}

func parseIntrospectionJSON(data []byte) (*Model, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("graphql: decode introspection JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("graphql: introspection JSON contains trailing content")
	}
	schemaRoot, ok := introspectionSchemaRoot(root)
	if !ok {
		return nil, fmt.Errorf("graphql: missing introspection __schema object")
	}
	model := &Model{
		SourceKind:       ArtifactKindIntrospectionJSON,
		RawIntrospection: root,
		Schema: Schema{
			QueryType:        namedTypeField(schemaRoot, "queryType"),
			MutationType:     namedTypeField(schemaRoot, "mutationType"),
			SubscriptionType: namedTypeField(schemaRoot, "subscriptionType"),
			Description:      stringField(schemaRoot, "description"),
		},
	}
	typeValues, _ := schemaRoot["types"].([]any)
	seenTypes := map[string]struct{}{}
	for _, value := range typeValues {
		typeRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		graphType := parseIntrospectionType(typeRoot)
		if graphType == nil || graphType.Name == "" {
			continue
		}
		if _, exists := seenTypes[graphType.Name]; exists {
			continue
		}
		seenTypes[graphType.Name] = struct{}{}
		model.Types = append(model.Types, graphType)
	}
	model.addSchemaRootOperations()
	sortModel(model)
	return model, nil
}

func introspectionSchemaRoot(root map[string]any) (map[string]any, bool) {
	if schema, ok := root["__schema"].(map[string]any); ok {
		return schema, true
	}
	data, ok := root["data"].(map[string]any)
	if !ok {
		return nil, false
	}
	schema, ok := data["__schema"].(map[string]any)
	return schema, ok
}

func parseIntrospectionType(root map[string]any) *Type {
	name := stringField(root, "name")
	graphType := &Type{
		Name:        name,
		Kind:        stringField(root, "kind"),
		Description: stringField(root, "description"),
		Selector:    TypeSelector(name),
	}
	for _, value := range arrayValue(root["fields"]) {
		fieldRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		field := parseIntrospectionField(name, fieldRoot)
		if field != nil {
			graphType.Fields = append(graphType.Fields, field)
		}
	}
	for _, value := range arrayValue(root["inputFields"]) {
		argRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		arg := parseIntrospectionArgument(argRoot)
		if arg != nil {
			graphType.InputFields = append(graphType.InputFields, arg)
		}
	}
	for _, value := range arrayValue(root["enumValues"]) {
		enumRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		enumValue := &EnumValue{
			Name:              stringField(enumRoot, "name"),
			Description:       stringField(enumRoot, "description"),
			Deprecated:        boolField(enumRoot, "isDeprecated"),
			DeprecationReason: stringField(enumRoot, "deprecationReason"),
		}
		if enumValue.Name != "" {
			graphType.EnumValues = append(graphType.EnumValues, enumValue)
		}
	}
	for _, value := range arrayValue(root["interfaces"]) {
		if name := typeRefFromIntrospection(value).Name; name != "" {
			graphType.Interfaces = append(graphType.Interfaces, name)
		}
	}
	for _, value := range arrayValue(root["possibleTypes"]) {
		if name := typeRefFromIntrospection(value).Name; name != "" {
			graphType.PossibleTypes = append(graphType.PossibleTypes, name)
		}
	}
	return graphType
}

func parseIntrospectionField(parentType string, root map[string]any) *Field {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	field := &Field{
		Name:              name,
		Description:       stringField(root, "description"),
		Type:              typeRefFromIntrospection(root["type"]),
		Deprecated:        boolField(root, "isDeprecated"),
		DeprecationReason: stringField(root, "deprecationReason"),
		Selector:          SchemaFieldSelector(parentType, name),
	}
	for _, value := range arrayValue(root["args"]) {
		argRoot, ok := value.(map[string]any)
		if !ok {
			continue
		}
		arg := parseIntrospectionArgument(argRoot)
		if arg != nil {
			field.Args = append(field.Args, arg)
		}
	}
	return field
}

func parseIntrospectionArgument(root map[string]any) *Argument {
	name := stringField(root, "name")
	if name == "" {
		return nil
	}
	typeRef := typeRefFromIntrospection(root["type"])
	return &Argument{
		Name:         name,
		Description:  stringField(root, "description"),
		Type:         typeRef,
		DefaultValue: stringField(root, "defaultValue"),
		Required:     typeRef.Required,
	}
}

func typeRefFromIntrospection(value any) TypeRef {
	root, ok := value.(map[string]any)
	if !ok {
		return TypeRef{}
	}
	ref := TypeRef{
		Name: stringField(root, "name"),
		Kind: stringField(root, "kind"),
	}
	if ofType := typeRefFromIntrospection(root["ofType"]); ofType.Display != "" {
		ref.OfType = &ofType
	}
	ref.Display = introspectionTypeDisplay(ref)
	ref.Required = ref.Kind == "NON_NULL"
	ref.List = ref.Kind == "LIST" || ref.OfType != nil && ref.OfType.List
	return ref
}

func introspectionTypeDisplay(ref TypeRef) string {
	switch ref.Kind {
	case "NON_NULL":
		if ref.OfType != nil && ref.OfType.Display != "" {
			return ref.OfType.Display + "!"
		}
		return "!"
	case "LIST":
		if ref.OfType != nil && ref.OfType.Display != "" {
			return "[" + ref.OfType.Display + "]"
		}
		return "[]"
	default:
		if ref.Name != "" {
			return ref.Name
		}
		if ref.OfType != nil {
			return ref.OfType.Display
		}
		return ""
	}
}

func parseGraphQLText(input string) (*Model, error) {
	tokens, err := lexGraphQL(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("graphql: empty document")
	}
	parser := graphQLParser{tokens: tokens}
	model := parser.parseDocument()
	if len(model.Types) == 0 && len(model.Operations) == 0 {
		return nil, fmt.Errorf("graphql: document does not contain schema or operation definitions")
	}
	if len(model.Types) > 0 {
		model.SourceKind = ArtifactKindSDL
	} else {
		model.SourceKind = ArtifactKindOperationDocument
	}
	model.addSchemaRootOperations()
	sortModel(model)
	return model, nil
}

type tokenKind string

const (
	tokenName   tokenKind = "name"
	tokenString tokenKind = "string"
	tokenPunct  tokenKind = "punct"
)

type graphQLToken struct {
	kind  tokenKind
	value string
}

func lexGraphQL(input string) ([]graphQLToken, error) {
	var tokens []graphQLToken
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		if unicode.IsSpace(r) || r == ',' {
			i += size
			continue
		}
		if r == '#' {
			for i < len(input) && input[i] != '\n' && input[i] != '\r' {
				i++
			}
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
			tokens = append(tokens, graphQLToken{kind: tokenName, value: input[start:i]})
			continue
		}
		if isGraphQLNumberStart(input, i) {
			start := i
			i++
			for i < len(input) && isGraphQLNumberContinue(input[i]) {
				i++
			}
			tokens = append(tokens, graphQLToken{kind: tokenName, value: input[start:i]})
			continue
		}
		if input[i] == '"' {
			value, next, err := readGraphQLString(input, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, graphQLToken{kind: tokenString, value: value})
			i = next
			continue
		}
		switch input[i] {
		case '!', '$', '&', '(', ')', ':', '=', '@', '[', ']', '{', '|', '}':
			tokens = append(tokens, graphQLToken{kind: tokenPunct, value: input[i : i+1]})
			i++
		case '.':
			if strings.HasPrefix(input[i:], "...") {
				tokens = append(tokens, graphQLToken{kind: tokenPunct, value: "..."})
				i += 3
				continue
			}
			return nil, fmt.Errorf("graphql: unexpected character %q", input[i])
		default:
			return nil, fmt.Errorf("graphql: unexpected character %q", input[i])
		}
	}
	return tokens, nil
}

func readGraphQLString(input string, start int) (string, int, error) {
	if strings.HasPrefix(input[start:], `"""`) {
		end := strings.Index(input[start+3:], `"""`)
		if end < 0 {
			return "", 0, fmt.Errorf("graphql: unterminated block string")
		}
		return strings.TrimSpace(input[start+3 : start+3+end]), start + 3 + end + 3, nil
	}
	var b strings.Builder
	for i := start + 1; i < len(input); i++ {
		if input[i] == '\\' {
			if i+1 >= len(input) {
				return "", 0, fmt.Errorf("graphql: unterminated string escape")
			}
			b.WriteByte(input[i+1])
			i++
			continue
		}
		if input[i] == '"' {
			return b.String(), i + 1, nil
		}
		b.WriteByte(input[i])
	}
	return "", 0, fmt.Errorf("graphql: unterminated string")
}

func isNameStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func isNameContinue(r rune) bool {
	return isNameStart(r) || r >= '0' && r <= '9'
}

func isGraphQLNumberStart(input string, i int) bool {
	if input[i] >= '0' && input[i] <= '9' {
		return true
	}
	return input[i] == '-' && i+1 < len(input) && input[i+1] >= '0' && input[i+1] <= '9'
}

func isGraphQLNumberContinue(b byte) bool {
	return b >= '0' && b <= '9' || b == '.' || b == 'e' || b == 'E' || b == '+' || b == '-'
}

type graphQLParser struct {
	tokens []graphQLToken
	pos    int
}

func (p *graphQLParser) parseDocument() *Model {
	model := &Model{}
	pendingDescription := ""
	for !p.done() {
		if p.peekKind(tokenString) {
			pendingDescription = strings.TrimSpace(p.advance().value)
			continue
		}
		if p.peekValue("extend") {
			p.advance()
		}
		if !p.peekKind(tokenName) && !p.peekValue("{") {
			p.advance()
			pendingDescription = ""
			continue
		}
		switch p.peek().value {
		case "schema":
			p.advance()
			p.parseSchema(model, pendingDescription)
		case "type", "interface":
			kind := strings.ToUpper(p.advance().value)
			model.Types = append(model.Types, p.parseObjectType(kind, pendingDescription))
		case "input":
			p.advance()
			model.Types = append(model.Types, p.parseInputType(pendingDescription))
		case "enum":
			p.advance()
			model.Types = append(model.Types, p.parseEnumType(pendingDescription))
		case "scalar":
			p.advance()
			model.Types = append(model.Types, p.parseScalarType("SCALAR", pendingDescription))
		case "union":
			p.advance()
			model.Types = append(model.Types, p.parseUnionType(pendingDescription))
		case "query", "mutation", "subscription":
			model.Operations = append(model.Operations, p.parseOperation(""))
		case "{":
			model.Operations = append(model.Operations, p.parseAnonymousQuery())
		default:
			p.skipDefinition()
		}
		pendingDescription = ""
	}
	model.Types = compactTypes(model.Types)
	model.Operations = compactOperations(model.Operations)
	return model
}

func (p *graphQLParser) parseSchema(model *Model, description string) {
	model.Schema.Description = description
	p.skipUntil("{")
	if !p.consume("{") {
		return
	}
	for !p.done() && !p.consume("}") {
		key := p.consumeName()
		if key == "" {
			p.advance()
			continue
		}
		if !p.consume(":") {
			continue
		}
		value := p.consumeName()
		switch key {
		case "query":
			model.Schema.QueryType = value
		case "mutation":
			model.Schema.MutationType = value
		case "subscription":
			model.Schema.SubscriptionType = value
		}
	}
}

func (p *graphQLParser) parseObjectType(kind, description string) *Type {
	name := p.consumeName()
	graphType := &Type{Name: name, Kind: kind, Description: description, Selector: TypeSelector(name)}
	p.skipUntil("{")
	if !p.consume("{") {
		return graphType
	}
	pendingDescription := ""
	for !p.done() && !p.consume("}") {
		if p.peekKind(tokenString) {
			pendingDescription = strings.TrimSpace(p.advance().value)
			continue
		}
		fieldName := p.consumeName()
		if fieldName == "" {
			p.advance()
			pendingDescription = ""
			continue
		}
		field := &Field{Name: fieldName, Description: pendingDescription, Selector: SchemaFieldSelector(name, fieldName)}
		pendingDescription = ""
		if p.peekValue("(") {
			field.Args = p.parseArgumentsDefinition()
		}
		if p.consume(":") {
			field.Type = p.parseTypeRef()
		}
		p.skipDirectives()
		graphType.Fields = append(graphType.Fields, field)
	}
	return graphType
}

func (p *graphQLParser) parseInputType(description string) *Type {
	name := p.consumeName()
	graphType := &Type{Name: name, Kind: "INPUT_OBJECT", Description: description, Selector: TypeSelector(name)}
	p.skipUntil("{")
	if !p.consume("{") {
		return graphType
	}
	pendingDescription := ""
	for !p.done() && !p.consume("}") {
		if p.peekKind(tokenString) {
			pendingDescription = strings.TrimSpace(p.advance().value)
			continue
		}
		argName := p.consumeName()
		if argName == "" {
			p.advance()
			pendingDescription = ""
			continue
		}
		arg := &Argument{Name: argName, Description: pendingDescription}
		pendingDescription = ""
		if p.consume(":") {
			arg.Type = p.parseTypeRef()
			arg.Required = arg.Type.Required
		}
		if p.consume("=") {
			arg.DefaultValue = p.collectDefaultValue()
		}
		p.skipDirectives()
		graphType.InputFields = append(graphType.InputFields, arg)
	}
	return graphType
}

func (p *graphQLParser) parseEnumType(description string) *Type {
	name := p.consumeName()
	graphType := &Type{Name: name, Kind: "ENUM", Description: description, Selector: TypeSelector(name)}
	p.skipUntil("{")
	if !p.consume("{") {
		return graphType
	}
	pendingDescription := ""
	for !p.done() && !p.consume("}") {
		if p.peekKind(tokenString) {
			pendingDescription = strings.TrimSpace(p.advance().value)
			continue
		}
		valueName := p.consumeName()
		if valueName == "" {
			p.advance()
			pendingDescription = ""
			continue
		}
		graphType.EnumValues = append(graphType.EnumValues, &EnumValue{Name: valueName, Description: pendingDescription})
		pendingDescription = ""
		p.skipDirectives()
	}
	return graphType
}

func (p *graphQLParser) parseScalarType(kind, description string) *Type {
	name := p.consumeName()
	return &Type{Name: name, Kind: kind, Description: description, Selector: TypeSelector(name)}
}

func (p *graphQLParser) parseUnionType(description string) *Type {
	name := p.consumeName()
	graphType := &Type{Name: name, Kind: "UNION", Description: description, Selector: TypeSelector(name)}
	for !p.done() && !p.peekValue("{") {
		if p.consume("=") || p.consume("|") {
			continue
		}
		member := p.consumeName()
		if member != "" {
			graphType.UnionMembers = append(graphType.UnionMembers, member)
			continue
		}
		if p.peekDefinitionBoundary() {
			break
		}
		p.advance()
	}
	return graphType
}

func (p *graphQLParser) parseOperation(forcedKind string) *Operation {
	kind := forcedKind
	if kind == "" {
		kind = p.advance().value
	}
	name := ""
	if p.peekKind(tokenName) {
		name = p.advance().value
	}
	operation := &Operation{Name: name, Kind: kind}
	if p.peekValue("(") {
		operation.Variables = p.parseVariablesDefinition()
	}
	p.skipDirectives()
	hasSelection := false
	if p.peekValue("{") {
		operation.SelectionNames = p.parseSelectionSet()
		hasSelection = true
	}
	if !hasSelection {
		return nil
	}
	if operation.Name == "" {
		operation.Name = "anonymous"
	}
	operation.ID = sourceOperationID(kind, operation.Name)
	operation.Selector = OperationSelector(operation.ID)
	operation.SourceRef = operation.Selector
	return operation
}

func (p *graphQLParser) parseAnonymousQuery() *Operation {
	operation := &Operation{Name: "anonymous", Kind: "query"}
	operation.SelectionNames = p.parseSelectionSet()
	operation.ID = sourceOperationID(operation.Kind, operation.Name)
	operation.Selector = OperationSelector(operation.ID)
	operation.SourceRef = operation.Selector
	return operation
}

func (p *graphQLParser) parseArgumentsDefinition() []*Argument {
	if !p.consume("(") {
		return nil
	}
	var args []*Argument
	for !p.done() && !p.consume(")") {
		name := p.consumeName()
		if name == "" {
			p.advance()
			continue
		}
		arg := &Argument{Name: name}
		if p.consume(":") {
			arg.Type = p.parseTypeRef()
			arg.Required = arg.Type.Required
		}
		if p.consume("=") {
			arg.DefaultValue = p.collectDefaultValue()
		}
		p.skipDirectives()
		args = append(args, arg)
	}
	return args
}

func (p *graphQLParser) parseVariablesDefinition() []*Variable {
	if !p.consume("(") {
		return nil
	}
	var variables []*Variable
	for !p.done() && !p.consume(")") {
		p.consume("$")
		name := p.consumeName()
		if name == "" {
			p.advance()
			continue
		}
		variable := &Variable{Name: name}
		if p.consume(":") {
			variable.Type = p.parseTypeRef()
			variable.Required = variable.Type.Required
		}
		if p.consume("=") {
			variable.DefaultValue = p.collectDefaultValue()
		}
		p.skipDirectives()
		variables = append(variables, variable)
	}
	return variables
}

func (p *graphQLParser) parseTypeRef() TypeRef {
	if p.consume("[") {
		inner := p.parseTypeRef()
		p.consume("]")
		ref := TypeRef{Display: "[" + inner.Display + "]", List: true, OfType: &inner}
		if p.consume("!") {
			ref.Display += "!"
			ref.Required = true
		}
		return ref
	}
	name := p.consumeName()
	ref := TypeRef{Name: name, Display: name}
	if p.consume("!") {
		ref.Display += "!"
		ref.Required = true
	}
	return ref
}

func (p *graphQLParser) parseSelectionSet() []string {
	if !p.consume("{") {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for !p.done() && !p.consume("}") {
		if p.consume("...") {
			p.consumeName()
			p.skipDirectives()
			continue
		}
		name := p.consumeName()
		if name == "" {
			p.advance()
			continue
		}
		selectionName := name
		if p.consume(":") {
			if target := p.consumeName(); target != "" {
				selectionName = target
			}
		}
		if _, exists := seen[selectionName]; !exists {
			seen[selectionName] = struct{}{}
			out = append(out, selectionName)
		}
		if p.peekValue("(") {
			p.skipBalanced("(", ")")
		}
		p.skipDirectives()
		if p.peekValue("{") {
			p.skipBalanced("{", "}")
		}
	}
	return out
}

func (p *graphQLParser) collectDefaultValue() string {
	var parts []string
	depth := 0
	for !p.done() {
		tok := p.peek()
		if depth == 0 && (tok.value == ")" || tok.value == "}" || tok.value == "@" || tok.kind == tokenName && p.peekNextValue(":")) {
			break
		}
		if tok.value == "[" || tok.value == "{" || tok.value == "(" {
			depth++
		}
		if tok.value == "]" || tok.value == "}" || tok.value == ")" {
			if depth == 0 {
				break
			}
			depth--
		}
		parts = append(parts, p.advance().value)
	}
	return strings.Join(parts, "")
}

func (p *graphQLParser) skipDirectives() {
	for p.consume("@") {
		p.consumeName()
		if p.peekValue("(") {
			p.skipBalanced("(", ")")
		}
	}
}

func (p *graphQLParser) skipDefinition() {
	if p.peekValue("{") {
		p.skipBalanced("{", "}")
		return
	}
	p.advance()
	for !p.done() && !p.peekDefinitionBoundary() {
		if p.peekValue("{") {
			p.skipBalanced("{", "}")
			return
		}
		p.advance()
	}
}

func (p *graphQLParser) skipUntil(value string) {
	for !p.done() && !p.peekValue(value) && !p.peekDefinitionBoundary() {
		p.advance()
	}
}

func (p *graphQLParser) skipBalanced(open, close string) {
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
}

func (p *graphQLParser) peekDefinitionBoundary() bool {
	if !p.peekKind(tokenName) {
		return false
	}
	switch p.peek().value {
	case "schema", "type", "interface", "input", "enum", "scalar", "union", "query", "mutation", "subscription", "fragment", "extend":
		return true
	default:
		return false
	}
}

func (p *graphQLParser) consumeName() string {
	if !p.peekKind(tokenName) {
		return ""
	}
	return p.advance().value
}

func (p *graphQLParser) consume(value string) bool {
	if !p.peekValue(value) {
		return false
	}
	p.advance()
	return true
}

func (p *graphQLParser) peekKind(kind tokenKind) bool {
	return !p.done() && p.peek().kind == kind
}

func (p *graphQLParser) peekValue(value string) bool {
	return !p.done() && p.peek().value == value
}

func (p *graphQLParser) peekNextValue(value string) bool {
	return p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].value == value
}

func (p *graphQLParser) peek() graphQLToken {
	return p.tokens[p.pos]
}

func (p *graphQLParser) advance() graphQLToken {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

func (p *graphQLParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (m *Model) addSchemaRootOperations() {
	if m == nil {
		return
	}
	for _, root := range []struct {
		kind string
		name string
	}{
		{kind: "query", name: firstNonEmpty(m.Schema.QueryType, "Query")},
		{kind: "mutation", name: firstNonEmpty(m.Schema.MutationType, "Mutation")},
		{kind: "subscription", name: firstNonEmpty(m.Schema.SubscriptionType, "Subscription")},
	} {
		graphType := m.typeByName(root.name)
		if graphType == nil || len(graphType.Fields) == 0 {
			continue
		}
		if root.kind == "query" && m.Schema.QueryType == "" {
			m.Schema.QueryType = root.name
		}
		if root.kind == "mutation" && m.Schema.MutationType == "" {
			m.Schema.MutationType = root.name
		}
		if root.kind == "subscription" && m.Schema.SubscriptionType == "" {
			m.Schema.SubscriptionType = root.name
		}
		for _, field := range graphType.Fields {
			if field == nil || strings.TrimSpace(field.Name) == "" {
				continue
			}
			id := sourceOperationID(root.kind, field.Name)
			if _, exists := m.OperationByID(id); exists {
				continue
			}
			m.Operations = append(m.Operations, &Operation{
				Name:        field.Name,
				Kind:        root.kind,
				ID:          id,
				Selector:    SchemaFieldSelector(root.name, field.Name),
				SourceRef:   SchemaFieldSelector(root.name, field.Name),
				Description: field.Description,
				Summary:     field.Description,
				FieldName:   field.Name,
				FieldType:   field.Type,
				RootType:    root.name,
			})
		}
	}
}

func (m *Model) typeByName(name string) *Type {
	name = strings.TrimSpace(name)
	for _, graphType := range m.Types {
		if graphType != nil && graphType.Name == name {
			return graphType
		}
	}
	return nil
}

func sourceOperationID(kind, name string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	name = strings.TrimSpace(name)
	if name == "" {
		name = "anonymous"
	}
	return kind + "." + name
}

func sortModel(model *Model) {
	sort.SliceStable(model.Types, func(i, j int) bool {
		return model.Types[i].Name < model.Types[j].Name
	})
	sort.SliceStable(model.Operations, func(i, j int) bool {
		return model.Operations[i].ID < model.Operations[j].ID
	})
}

func compactTypes(in []*Type) []*Type {
	byName := map[string]*Type{}
	var out []*Type
	for _, graphType := range in {
		if graphType == nil || strings.TrimSpace(graphType.Name) == "" {
			continue
		}
		if existing := byName[graphType.Name]; existing != nil {
			existing.Fields = append(existing.Fields, graphType.Fields...)
			existing.InputFields = append(existing.InputFields, graphType.InputFields...)
			existing.EnumValues = append(existing.EnumValues, graphType.EnumValues...)
			existing.Interfaces = append(existing.Interfaces, graphType.Interfaces...)
			existing.PossibleTypes = append(existing.PossibleTypes, graphType.PossibleTypes...)
			existing.UnionMembers = append(existing.UnionMembers, graphType.UnionMembers...)
			if existing.Description == "" {
				existing.Description = graphType.Description
			}
			continue
		}
		byName[graphType.Name] = graphType
		out = append(out, graphType)
	}
	return out
}

func compactOperations(in []*Operation) []*Operation {
	seen := map[string]int{}
	var out []*Operation
	for _, operation := range in {
		if operation == nil || strings.TrimSpace(operation.Kind) == "" {
			continue
		}
		if operation.Name == "" {
			operation.Name = "anonymous"
		}
		baseID := sourceOperationID(operation.Kind, operation.Name)
		seen[baseID]++
		if seen[baseID] > 1 {
			operation.ID = fmt.Sprintf("%s.%d", baseID, seen[baseID])
		} else {
			operation.ID = baseID
		}
		operation.Selector = OperationSelector(operation.ID)
		operation.SourceRef = operation.Selector
		out = append(out, operation)
	}
	return out
}

func graphQLFormatForPath(path string, kind ArtifactKind) string {
	switch kind {
	case ArtifactKindIntrospectionJSON:
		return "json"
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".graphql", ".graphqls", ".gql":
		return "graphql"
	case ".json":
		return "json"
	default:
		return "graphql"
	}
}

func namedTypeField(root map[string]any, key string) string {
	obj, ok := root[key].(map[string]any)
	if !ok {
		return ""
	}
	return stringField(obj, "name")
}

func arrayValue(value any) []any {
	out, _ := value.([]any)
	return out
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
