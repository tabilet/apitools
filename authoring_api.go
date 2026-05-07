package apitools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// AuthoringAPIDocument is the prompt-safe OpenAPI document shape used by
// generic interactive authoring loops.
type AuthoringAPIDocument struct {
	ID           string             `json:"id,omitempty"`
	Path         string             `json:"path,omitempty"`
	RelativePath string             `json:"relative_path,omitempty"`
	Title        string             `json:"title,omitempty"`
	Description  string             `json:"description,omitempty"`
	Operations   []OperationSummary `json:"operations,omitempty"`
}

// AuthoringAPIDocumentOptions configures prompt-safe OpenAPI document context.
type AuthoringAPIDocumentOptions struct {
	Documents []InventoryDocument `json:"documents,omitempty"`
	BaseDir   string              `json:"base_dir,omitempty"`
	Query     string              `json:"query,omitempty"`
}

// AuthoringOperationRef identifies an already-selected OpenAPI operation.
type AuthoringOperationRef struct {
	DocumentPath string `json:"document_path,omitempty"`
	OperationID  string `json:"operation_id,omitempty"`
}

// AuthoringOperationRankingOptions controls ranked operation prompt context.
type AuthoringOperationRankingOptions struct {
	Query              string                  `json:"query,omitempty"`
	SelectedOperations []AuthoringOperationRef `json:"selected_operations,omitempty"`
	Limit              int                     `json:"limit,omitempty"`
}

// OperationRequestFieldInfo describes one allowed request mapping field.
type OperationRequestFieldInfo struct {
	Type string `json:"type,omitempty"`
	Body bool   `json:"body,omitempty"`
}

// BuildAuthoringAPIDocuments builds grouped, prompt-safe OpenAPI context.
func BuildAuthoringAPIDocuments(ctx context.Context, opts AuthoringAPIDocumentOptions) ([]AuthoringAPIDocument, error) {
	if len(opts.Documents) == 0 {
		return nil, nil
	}
	docs := append([]InventoryDocument(nil), opts.Documents...)
	for i := range docs {
		if strings.TrimSpace(docs[i].RelativePath) == "" {
			docs[i].RelativePath = authoringRelativePath(docs[i].Path, opts.BaseDir)
		}
	}
	inventory, err := BuildOperationInventory(ctx, InventoryOptions{
		Documents: docs,
		Query:     opts.Query,
	})
	if err != nil {
		return nil, err
	}
	out := make([]AuthoringAPIDocument, 0, len(inventory.Documents))
	for _, summary := range inventory.Documents {
		doc := AuthoringAPIDocument{
			ID:           authoringDocumentID(summary),
			Path:         summary.Path,
			RelativePath: firstNonEmpty(summary.RelativePath, authoringRelativePath(summary.Path, opts.BaseDir), summary.Path, summary.URL),
			Title:        summary.Title,
			Description:  summary.Description,
		}
		for _, op := range inventory.Operations {
			if operationBelongsToDocument(op, summary) {
				doc.Operations = append(doc.Operations, op)
			}
		}
		out = append(out, doc)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return authoringDocSortKey(out[i]) < authoringDocSortKey(out[j])
	})
	return out, nil
}

// RankAuthoringAPIDocuments returns copies of docs with operations ranked for a
// prompt. Selected operations are preserved even beyond Limit.
func RankAuthoringAPIDocuments(docs []AuthoringAPIDocument, opts AuthoringOperationRankingOptions) []AuthoringAPIDocument {
	limit := opts.Limit
	if limit <= 0 {
		limit = 12
	}
	selected := authoringSelectedOperations(opts.SelectedOperations)
	type candidate struct {
		docIndex int
		opIndex  int
		op       OperationSummary
		score    int
		selected bool
		rank     int
	}
	var candidates []candidate
	query := opts.Query
	for docIndex, doc := range docs {
		for opIndex, op := range doc.Operations {
			key := authoringOperationKey(firstNonEmpty(doc.RelativePath, doc.Path), op.OperationID)
			isSelected := selected[key] || selected[authoringOperationKey("", op.OperationID)]
			score := ScoreText(query, authoringOperationSearchText(doc, op))
			if isSelected {
				score += 1000
			}
			candidates = append(candidates, candidate{
				docIndex: docIndex,
				opIndex:  opIndex,
				op:       op,
				score:    score,
				selected: isSelected,
			})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].docIndex != candidates[j].docIndex {
			return candidates[i].docIndex < candidates[j].docIndex
		}
		return candidates[i].opIndex < candidates[j].opIndex
	})
	include := map[string]int{}
	unselected := 0
	for rank, candidate := range candidates {
		candidate.rank = rank
		key := fmt.Sprintf("%d/%d", candidate.docIndex, candidate.opIndex)
		if candidate.selected || unselected < limit {
			include[key] = rank
			if !candidate.selected {
				unselected++
			}
		}
	}
	out := make([]AuthoringAPIDocument, 0, len(docs))
	for docIndex, doc := range docs {
		copyDoc := doc
		copyDoc.Operations = nil
		for opIndex, op := range doc.Operations {
			key := fmt.Sprintf("%d/%d", docIndex, opIndex)
			if _, ok := include[key]; ok {
				copyDoc.Operations = append(copyDoc.Operations, op)
			}
		}
		sort.SliceStable(copyDoc.Operations, func(i, j int) bool {
			left := include[fmt.Sprintf("%d/%d", docIndex, operationIndex(doc.Operations, copyDoc.Operations[i]))]
			right := include[fmt.Sprintf("%d/%d", docIndex, operationIndex(doc.Operations, copyDoc.Operations[j]))]
			return left < right
		})
		out = append(out, copyDoc)
	}
	return out
}

// OperationLabel renders a concise human-readable operation label.
func OperationLabel(op OperationSummary) string {
	id := firstNonEmpty(op.OperationID, op.ID)
	text := strings.TrimSpace(strings.Join(nonEmptyStrings(op.Method, op.Path, id), " "))
	if op.Summary != "" {
		text += " - " + strings.TrimSpace(op.Summary)
	}
	return strings.TrimSpace(text)
}

// RequiredOperationFields returns required request and credential field names.
func RequiredOperationFields(op OperationSummary) []string {
	var out []string
	for _, parameter := range op.Parameters {
		if parameter.Required && strings.TrimSpace(parameter.Name) != "" {
			out = append(out, parameter.Name)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Required {
		fields := requiredLeafRequestFields(op.RequestBody.Fields)
		if len(fields) == 0 {
			fields = []string{"body"}
		}
		out = append(out, fields...)
	}
	out = append(out, OperationCredentialFields(op)...)
	return sortedUniqueStrings(out)
}

// OperationCredentialFields returns prompt-safe symbolic credential field names.
func OperationCredentialFields(op OperationSummary) []string {
	var out []string
	for _, security := range op.Security {
		if field := SecurityCredentialFieldName(security); field != "" {
			out = append(out, field)
		}
	}
	return sortedUniqueStrings(out)
}

// OperationNeedsCredential reports whether an operation declares security.
func OperationNeedsCredential(op OperationSummary) bool {
	return len(op.Security) > 0
}

// OperationRequestFieldTypes returns all valid request mapping fields.
func OperationRequestFieldTypes(op OperationSummary) map[string]OperationRequestFieldInfo {
	out := map[string]OperationRequestFieldInfo{}
	for _, parameter := range op.Parameters {
		if strings.TrimSpace(parameter.Name) != "" {
			out[parameter.Name] = OperationRequestFieldInfo{Type: parameter.Type}
		}
	}
	for _, field := range OperationCredentialFields(op) {
		out[field] = OperationRequestFieldInfo{Type: "string"}
	}
	if op.RequestBody != nil {
		for _, field := range op.RequestBody.Fields {
			if strings.TrimSpace(field.Path) != "" {
				out[field.Path] = OperationRequestFieldInfo{Type: field.Type, Body: true}
			}
		}
	}
	return out
}

// SecurityCredentialFieldName maps an OpenAPI security scheme to a symbolic
// request field name for authoring prompts.
func SecurityCredentialFieldName(security SecuritySummary) string {
	name := firstNonEmpty(security.Name, security.ParameterName)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "api") && strings.Contains(lower, "key") {
		return camelToSnakeAuthoring(name)
	}
	if strings.EqualFold(security.Scheme, "bearer") || strings.Contains(lower, "bearer") || strings.Contains(lower, "auth") || strings.Contains(lower, "token") {
		return "Authorization"
	}
	return camelToSnakeAuthoring(name)
}

func authoringRelativePath(path, baseDir string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return filepath.ToSlash(path)
	}
	if rel, err := filepath.Rel(baseDir, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func authoringDocumentID(summary DocumentSummary) string {
	id := firstNonEmpty(summary.Name, summary.Title, summary.RelativePath, summary.Path, summary.URL)
	if id == "" {
		return ""
	}
	if base := filepath.Base(id); base != "." && base != string(filepath.Separator) {
		id = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return sanitizeName(id)
}

func operationBelongsToDocument(op OperationSummary, doc DocumentSummary) bool {
	return op.DocumentPath == doc.Path &&
		op.DocumentRelativePath == doc.RelativePath &&
		op.DocumentURL == doc.URL &&
		op.DocumentName == doc.Name
}

func authoringDocSortKey(doc AuthoringAPIDocument) string {
	return firstNonEmpty(doc.RelativePath, doc.Path, doc.ID, doc.Title)
}

func authoringSelectedOperations(refs []AuthoringOperationRef) map[string]bool {
	out := map[string]bool{}
	for _, ref := range refs {
		op := strings.TrimSpace(ref.OperationID)
		if op == "" {
			continue
		}
		out[authoringOperationKey(ref.DocumentPath, op)] = true
		if strings.TrimSpace(ref.DocumentPath) == "" {
			out[authoringOperationKey("", op)] = true
		}
	}
	return out
}

func authoringOperationKey(docPath, operationID string) string {
	return strings.TrimSpace(docPath) + "\x00" + strings.TrimSpace(operationID)
}

func authoringOperationSearchText(doc AuthoringAPIDocument, op OperationSummary) string {
	var parts []string
	parts = append(parts, doc.ID, doc.RelativePath, doc.Path, doc.Title, doc.Description)
	parts = append(parts, op.OperationID, op.ID, op.Method, op.Path, op.Summary, op.Description)
	parts = append(parts, op.Tags...)
	for _, parameter := range op.Parameters {
		parts = append(parts, parameter.Name, parameter.In, parameter.Type, parameter.Description)
	}
	if op.RequestBody != nil {
		parts = append(parts, op.RequestBody.Description)
		parts = append(parts, op.RequestBody.RequiredFieldPaths...)
		for _, field := range op.RequestBody.Fields {
			parts = append(parts, field.Path, field.Type, field.Description)
		}
	}
	for _, field := range OperationCredentialFields(op) {
		parts = append(parts, field)
	}
	return strings.Join(parts, " ")
}

func operationIndex(ops []OperationSummary, op OperationSummary) int {
	for i := range ops {
		if ops[i].ID == op.ID && ops[i].OperationID == op.OperationID && ops[i].Method == op.Method && ops[i].Path == op.Path {
			return i
		}
	}
	return -1
}

func requiredLeafRequestFields(fields []RequestFieldSummary) []string {
	var required []RequestFieldSummary
	for _, field := range fields {
		if field.Required {
			required = append(required, field)
		}
	}
	var out []string
	for _, field := range required {
		if requestFieldHasRequiredDescendant(field.Path, required) {
			continue
		}
		out = append(out, field.Path)
	}
	sort.Strings(out)
	return out
}

func requestFieldHasRequiredDescendant(path string, fields []RequestFieldSummary) bool {
	prefix := path + "."
	arrayPrefix := path + "[]."
	for _, field := range fields {
		if field.Path == path {
			continue
		}
		if strings.HasPrefix(field.Path, prefix) || strings.HasPrefix(field.Path, arrayPrefix) {
			return true
		}
	}
	return false
}

func camelToSnakeAuthoring(value string) string {
	var out []rune
	var prev rune
	for i, r := range value {
		if r == '-' || r == ' ' {
			r = '_'
		}
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
			r = unicode.ToLower(r)
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out = append(out, r)
			prev = r
		}
	}
	return strings.Trim(string(out), "_")
}
