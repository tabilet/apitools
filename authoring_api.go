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
	Documents     []InventoryDocument `json:"documents,omitempty"`
	BaseDir       string              `json:"base_dir,omitempty"`
	Query         string              `json:"query,omitempty"`
	MaxBytes      int64               `json:"max_bytes,omitempty"`
	MaxOperations int                 `json:"max_operations,omitempty"`
	PromptBudget  PromptBudget        `json:"prompt_budget,omitempty"`
}

// AuthoringAPIDocumentReport preserves prompt-safety diagnostics that would
// otherwise be lost while grouping inventory operations by source document.
type AuthoringAPIDocumentReport struct {
	Documents       []AuthoringAPIDocument `json:"documents,omitempty"`
	Diagnostics     []Diagnostic           `json:"diagnostics,omitempty"`
	ReadinessIssues []ReadinessIssue       `json:"readiness_issues,omitempty"`
	Truncated       bool                   `json:"truncated"`
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
	PromptBudget       PromptBudget            `json:"prompt_budget,omitempty"`
}

// AuthoringPromptContextReport is a bounded ranked prompt payload. Bytes is
// the encoded size of Documents, or the selected-only size when a selected
// operation set exceeds the aggregate budget.
type AuthoringPromptContextReport struct {
	Documents          []AuthoringAPIDocument `json:"documents,omitempty"`
	Diagnostics        []Diagnostic           `json:"diagnostics,omitempty"`
	Truncated          bool                   `json:"truncated"`
	Bytes              int                    `json:"bytes"`
	SelectedOperations int                    `json:"selected_operations,omitempty"`
}

type authoringCandidate struct {
	docIndex int
	opIndex  int
	op       OperationSummary
	score    int
	selected bool
}

// OperationRequestFieldInfo describes one allowed request mapping field.
type OperationRequestFieldInfo struct {
	Type string `json:"type,omitempty"`
	Body bool   `json:"body,omitempty"`
}

// BuildAuthoringAPIDocuments builds grouped, prompt-safe OpenAPI context.
func BuildAuthoringAPIDocuments(ctx context.Context, opts AuthoringAPIDocumentOptions) (AuthoringAPIDocumentReport, error) {
	if len(opts.Documents) == 0 {
		return AuthoringAPIDocumentReport{}, nil
	}
	docs := append([]InventoryDocument(nil), opts.Documents...)
	for i := range docs {
		if strings.TrimSpace(docs[i].RelativePath) == "" {
			docs[i].RelativePath = authoringRelativePath(docs[i].Path, opts.BaseDir)
		}
	}
	inventory, err := BuildOperationInventory(ctx, InventoryOptions{
		Documents:     docs,
		Query:         opts.Query,
		MaxBytes:      opts.MaxBytes,
		MaxOperations: opts.MaxOperations,
	})
	if err != nil {
		return AuthoringAPIDocumentReport{}, err
	}
	sanitizeInventory(&inventory, opts.PromptBudget)
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
	return AuthoringAPIDocumentReport{
		Documents: out, Diagnostics: inventory.Diagnostics,
		ReadinessIssues: inventory.ReadinessIssues, Truncated: inventory.Truncated,
	}, nil
}

// RankAuthoringAPIDocuments returns a bounded prompt context with operations
// ranked by exact document and operation indexes. Selected operations are
// never silently omitted; if they cannot fit together, the function returns a
// blocking diagnostic and error.
func RankAuthoringAPIDocuments(docs []AuthoringAPIDocument, opts AuthoringOperationRankingOptions) (AuthoringPromptContextReport, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 12
	}
	selected := authoringSelectedOperations(opts.SelectedOperations)
	budget := resolvedPromptBudget(opts.PromptBudget)
	var candidates []authoringCandidate
	query := opts.Query
	for docIndex, doc := range docs {
		for opIndex, op := range doc.Operations {
			key := authoringOperationKey(firstNonEmpty(doc.RelativePath, doc.Path), op.OperationID)
			isSelected := selected[key] || selected[authoringOperationKey("", op.OperationID)]
			score := ScoreText(query, authoringOperationSearchText(doc, op))
			if isSelected {
				score += 1000
			}
			candidates = append(candidates, authoringCandidate{
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
	report := AuthoringPromptContextReport{}
	for i := range candidates {
		diagnostics := sanitizeOperationSummary(&candidates[i].op, budget)
		report.Diagnostics = append(report.Diagnostics, diagnostics...)
	}
	report.Truncated = len(report.Diagnostics) > 0
	if errors := errorDiagnostics(report.Diagnostics); len(errors) > 0 {
		return report, DiagnosticError{Diagnostics: errors}
	}
	include := map[string]bool{}
	for _, candidate := range candidates {
		if !candidate.selected {
			continue
		}
		include[authoringCandidateKey(candidate.docIndex, candidate.opIndex)] = true
		report.SelectedOperations++
	}
	selectedDocs := buildRankedAuthoringDocuments(docs, candidates, include, budget)
	selectedBytes := promptJSONSize(selectedDocs)
	if selectedBytes > budget.MaxContextBytes {
		diagnostic := Diagnostic{
			Severity: "error", Code: "prompt.selected_budget",
			Message:     fmt.Sprintf("selected operations require %d bytes, exceeding the %d-byte authoring context budget", selectedBytes, budget.MaxContextBytes),
			Remediation: "Narrow the selected operations or increase MaxContextBytes for a reviewed prompt.",
		}
		report.Diagnostics = append(report.Diagnostics, diagnostic)
		report.Bytes = selectedBytes
		return report, DiagnosticError{Diagnostics: []Diagnostic{diagnostic}}
	}

	unselected := 0
	skippedForBudget := false
	for _, candidate := range candidates {
		if candidate.selected || unselected >= limit {
			continue
		}
		key := authoringCandidateKey(candidate.docIndex, candidate.opIndex)
		include[key] = true
		tentative := buildRankedAuthoringDocuments(docs, candidates, include, budget)
		if promptJSONSize(tentative) > budget.MaxContextBytes {
			delete(include, key)
			skippedForBudget = true
			continue
		}
		unselected++
	}
	report.Documents = buildRankedAuthoringDocuments(docs, candidates, include, budget)
	report.Bytes = promptJSONSize(report.Documents)
	if skippedForBudget {
		report.Truncated = true
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Severity: "warning", Code: "prompt.context_budget",
			Message:     fmt.Sprintf("ranked operations were truncated to the %d-byte authoring context budget", budget.MaxContextBytes),
			Remediation: "Narrow the query or increase MaxContextBytes for a reviewed prompt.",
		})
	}
	return report, nil
}

func authoringCandidateKey(docIndex, opIndex int) string {
	return fmt.Sprintf("%d/%d", docIndex, opIndex)
}

func buildRankedAuthoringDocuments(docs []AuthoringAPIDocument, candidates []authoringCandidate, include map[string]bool, budget PromptBudget) []AuthoringAPIDocument {
	copies := make([]AuthoringAPIDocument, len(docs))
	for i, doc := range docs {
		copies[i] = doc
		copies[i].ID, _ = sanitizePromptString(doc.ID, budget.MaxIdentifierRunes)
		copies[i].Path, _ = sanitizePromptString(doc.Path, budget.MaxTextRunes)
		copies[i].RelativePath, _ = sanitizePromptString(doc.RelativePath, budget.MaxTextRunes)
		copies[i].Title, _ = sanitizePromptString(doc.Title, budget.MaxTextRunes)
		copies[i].Description, _ = sanitizePromptString(doc.Description, budget.MaxTextRunes)
		copies[i].Operations = nil
	}
	for _, candidate := range candidates {
		if include[authoringCandidateKey(candidate.docIndex, candidate.opIndex)] {
			copies[candidate.docIndex].Operations = append(copies[candidate.docIndex].Operations, candidate.op)
		}
	}
	out := make([]AuthoringAPIDocument, 0, len(copies))
	for _, doc := range copies {
		if len(doc.Operations) > 0 {
			out = append(out, doc)
		}
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
	parts = append(parts, OperationCredentialFields(op)...)
	return strings.Join(parts, " ")
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
