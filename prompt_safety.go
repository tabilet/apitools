package apitools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultPromptIdentifierRunes = 256
	DefaultPromptTextRunes       = 2_048
	DefaultPromptCollectionItems = 32
	DefaultPromptFieldItems      = 60
	DefaultPromptOperationBytes  = 32 << 10
	DefaultPromptContextBytes    = 512 << 10
)

// PromptBudget bounds untrusted metadata after parsing and before it is
// exposed to authoring prompts. Zero values select the documented defaults.
type PromptBudget struct {
	MaxIdentifierRunes int `json:"max_identifier_runes,omitempty"`
	MaxTextRunes       int `json:"max_text_runes,omitempty"`
	MaxCollectionItems int `json:"max_collection_items,omitempty"`
	MaxFields          int `json:"max_fields,omitempty"`
	MaxOperations      int `json:"max_operations,omitempty"`
	MaxOperationBytes  int `json:"max_operation_bytes,omitempty"`
	MaxContextBytes    int `json:"max_context_bytes,omitempty"`
}

// PromptOperationSummaryReport contains sanitized copies of source operation
// summaries and every visible truncation or compaction diagnostic.
type PromptOperationSummaryReport struct {
	Operations  []OperationSummary `json:"operations,omitempty"`
	Diagnostics []Diagnostic       `json:"diagnostics,omitempty"`
	Truncated   bool               `json:"truncated"`
}

// DefaultPromptBudget returns the standard authoring metadata budget.
func DefaultPromptBudget() PromptBudget {
	return PromptBudget{
		MaxIdentifierRunes: DefaultPromptIdentifierRunes,
		MaxTextRunes:       DefaultPromptTextRunes,
		MaxCollectionItems: DefaultPromptCollectionItems,
		MaxFields:          DefaultPromptFieldItems,
		MaxOperations:      DefaultMaxInventoryOperations,
		MaxOperationBytes:  DefaultPromptOperationBytes,
		MaxContextBytes:    DefaultPromptContextBytes,
	}
}

// SanitizeOperationSummaries applies the shared prompt budget without
// mutating caller-owned summaries. Compaction that removes technical metadata
// is returned as a blocking DiagnosticError.
func SanitizeOperationSummaries(operations []OperationSummary, budget PromptBudget) (PromptOperationSummaryReport, error) {
	report := PromptOperationSummaryReport{}
	if len(operations) == 0 {
		return report, nil
	}
	budget = resolvedPromptBudget(budget)
	if len(operations) > budget.MaxOperations {
		diagnostic := Diagnostic{
			Severity: "error", Code: "prompt.operation_count",
			Message:     fmt.Sprintf("operation summaries contain %d operations, exceeding the %d-operation prompt work budget", len(operations), budget.MaxOperations),
			Remediation: "Narrow the source or increase MaxOperations for a reviewed prompt.",
		}
		report.Diagnostics = []Diagnostic{diagnostic}
		report.Truncated = true
		return report, DiagnosticError{Diagnostics: []Diagnostic{diagnostic}}
	}
	encoded, err := json.Marshal(operations)
	if err != nil {
		return report, fmt.Errorf("encode operation summaries for prompt sanitization: %w", err)
	}
	if err := json.Unmarshal(encoded, &report.Operations); err != nil {
		return report, fmt.Errorf("clone operation summaries for prompt sanitization: %w", err)
	}
	for i := range report.Operations {
		diagnostics := sanitizeOperationSummary(&report.Operations[i], budget)
		report.Diagnostics = append(report.Diagnostics, diagnostics...)
	}
	report.Truncated = len(report.Diagnostics) > 0
	if errors := errorDiagnostics(report.Diagnostics); len(errors) > 0 {
		return report, DiagnosticError{Diagnostics: errors}
	}
	return report, nil
}

var ansiControlSequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func resolvedPromptBudget(budget PromptBudget) PromptBudget {
	defaults := DefaultPromptBudget()
	if budget.MaxIdentifierRunes <= 0 {
		budget.MaxIdentifierRunes = defaults.MaxIdentifierRunes
	}
	if budget.MaxTextRunes <= 0 {
		budget.MaxTextRunes = defaults.MaxTextRunes
	}
	if budget.MaxCollectionItems <= 0 {
		budget.MaxCollectionItems = defaults.MaxCollectionItems
	}
	if budget.MaxFields <= 0 {
		budget.MaxFields = defaults.MaxFields
	}
	if budget.MaxOperations <= 0 {
		budget.MaxOperations = defaults.MaxOperations
	}
	if budget.MaxOperationBytes <= 0 {
		budget.MaxOperationBytes = defaults.MaxOperationBytes
	}
	if budget.MaxContextBytes <= 0 {
		budget.MaxContextBytes = defaults.MaxContextBytes
	}
	return budget
}

func sanitizeInventory(inventory *OperationInventory, budget PromptBudget) {
	if inventory == nil {
		return
	}
	budget = resolvedPromptBudget(budget)
	for i := range inventory.Documents {
		if sanitizeDocumentSummary(&inventory.Documents[i], budget) {
			inventory.Diagnostics = append(inventory.Diagnostics, Diagnostic{
				Severity: "warning", Code: "prompt.document_sanitized",
				Message:     "document metadata contained controls or values beyond the prompt-safety budget and was sanitized",
				Path:        inventory.Documents[i].Path,
				Remediation: "Review the source metadata if the removed or shortened text is required.",
			})
			inventory.Truncated = true
		}
	}
	for i := range inventory.Operations {
		diagnostics := sanitizeOperationSummary(&inventory.Operations[i], budget)
		if len(diagnostics) > 0 {
			inventory.Diagnostics = append(inventory.Diagnostics, diagnostics...)
			inventory.Truncated = true
		}
	}
}

func sanitizeDocumentSummary(summary *DocumentSummary, budget PromptBudget) bool {
	if summary == nil {
		return false
	}
	changed := false
	summary.Name, changed = sanitizePromptValue(summary.Name, budget.MaxIdentifierRunes, changed)
	summary.Path, changed = sanitizePromptValue(summary.Path, budget.MaxTextRunes, changed)
	summary.RelativePath, changed = sanitizePromptValue(summary.RelativePath, budget.MaxTextRunes, changed)
	summary.URL, changed = sanitizePromptValue(summary.URL, budget.MaxTextRunes, changed)
	summary.Title, changed = sanitizePromptValue(summary.Title, budget.MaxTextRunes, changed)
	summary.Description, changed = sanitizePromptValue(summary.Description, budget.MaxTextRunes, changed)
	summary.OpenAPI, changed = sanitizePromptValue(summary.OpenAPI, budget.MaxIdentifierRunes, changed)
	summary.Swagger, changed = sanitizePromptValue(summary.Swagger, budget.MaxIdentifierRunes, changed)
	return changed
}

func sanitizeOperationSummary(operation *OperationSummary, budget PromptBudget) []Diagnostic {
	if operation == nil {
		return nil
	}
	budget = resolvedPromptBudget(budget)
	securityCompacted := len(operation.SecurityRequirementSets) > budget.MaxCollectionItems
	if !securityCompacted {
		for _, set := range operation.SecurityRequirementSets {
			if len(set.Requirements) > budget.MaxCollectionItems {
				securityCompacted = true
				break
			}
		}
	}
	changed := sanitizeOperationFields(operation, budget)
	compacted := securityCompacted
	byteCompacted := false
	if promptJSONSize(operation) > budget.MaxOperationBytes {
		compactOperationDetails(operation)
		changed = true
		compacted = true
		byteCompacted = true
	}
	if promptJSONSize(operation) > budget.MaxOperationBytes {
		minimal := minimalPromptOperation(*operation, budget)
		*operation = minimal
		changed = true
		compacted = true
		byteCompacted = true
	}
	if !changed {
		return nil
	}
	severity := "warning"
	code := "prompt.operation_sanitized"
	message := "operation metadata contained controls or values beyond the prompt-safety budget and was sanitized"
	remediation := "Review the source metadata if the removed or shortened text is required."
	if compacted {
		severity = "error"
		code = "prompt.operation_budget"
		if byteCompacted {
			message = fmt.Sprintf("operation metadata exceeded the %d-byte prompt budget and was compacted", budget.MaxOperationBytes)
			remediation = "Narrow the operation schema or use the source document directly during reviewed mapping."
		} else {
			message = fmt.Sprintf("operation security metadata exceeded the %d-item prompt collection budget and was compacted", budget.MaxCollectionItems)
			remediation = "Narrow the operation security alternatives or review the source document directly before selecting authentication."
		}
		operation.ReadinessIssues = append(operation.ReadinessIssues, ReadinessIssue{
			Severity: "error", Code: code, Message: message, OperationID: operation.OperationID,
			Path: operation.Provenance, Remediation: remediation,
		})
		if promptJSONSize(operation) > budget.MaxOperationBytes {
			issue := operation.ReadinessIssues[len(operation.ReadinessIssues)-1]
			*operation = minimalPromptOperation(*operation, budget)
			operation.ReadinessIssues = []ReadinessIssue{issue}
		}
	}
	return []Diagnostic{{Severity: severity, Code: code, Message: message, Path: operation.Provenance, Remediation: remediation}}
}

func sanitizeOperationFields(operation *OperationSummary, budget PromptBudget) bool {
	changed := false
	id := func(value string) string {
		out, unsafe := sanitizePromptString(value, budget.MaxIdentifierRunes)
		changed = changed || unsafe
		return out
	}
	text := func(value string) string {
		out, unsafe := sanitizePromptString(value, budget.MaxTextRunes)
		changed = changed || unsafe
		return out
	}
	operation.ID = id(operation.ID)
	operation.DocumentName = id(operation.DocumentName)
	operation.DocumentPath = text(operation.DocumentPath)
	operation.DocumentRelativePath = text(operation.DocumentRelativePath)
	operation.DocumentURL = text(operation.DocumentURL)
	operation.OperationID = id(operation.OperationID)
	operation.Method = id(operation.Method)
	operation.Path = text(operation.Path)
	operation.Summary = text(operation.Summary)
	operation.Description = text(operation.Description)
	operation.Provenance = text(operation.Provenance)
	operation.Tags, changed = sanitizeStringCollection(operation.Tags, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	operation.Extensions, changed = sanitizeStringMap(operation.Extensions, budget, changed)

	if len(operation.Parameters) > budget.MaxCollectionItems {
		operation.Parameters = operation.Parameters[:budget.MaxCollectionItems]
		changed = true
	}
	for i := range operation.Parameters {
		parameter := &operation.Parameters[i]
		parameter.Name = id(parameter.Name)
		parameter.In = id(parameter.In)
		parameter.Description = text(parameter.Description)
		parameter.Type = id(parameter.Type)
		parameter.Format = id(parameter.Format)
		parameter.Ref = text(parameter.Ref)
	}
	if operation.RequestBody != nil {
		changed = sanitizeRequestBody(operation.RequestBody, budget, changed)
	}
	if operation.ResponseBody != nil {
		changed = sanitizeResponseBody(operation.ResponseBody, budget, changed)
	}
	if len(operation.SecurityRequirementSets) > budget.MaxCollectionItems {
		operation.SecurityRequirementSets = operation.SecurityRequirementSets[:budget.MaxCollectionItems]
		changed = true
	}
	for setIndex := range operation.SecurityRequirementSets {
		securitySet := &operation.SecurityRequirementSets[setIndex]
		if len(securitySet.Requirements) > budget.MaxCollectionItems {
			securitySet.Requirements = securitySet.Requirements[:budget.MaxCollectionItems]
			changed = true
		}
		for requirementIndex := range securitySet.Requirements {
			changed = sanitizeSecuritySummary(&securitySet.Requirements[requirementIndex], budget, changed)
		}
	}
	if len(operation.ReadinessIssues) > budget.MaxCollectionItems {
		operation.ReadinessIssues = operation.ReadinessIssues[:budget.MaxCollectionItems]
		changed = true
	}
	for i := range operation.ReadinessIssues {
		issue := &operation.ReadinessIssues[i]
		issue.Severity = id(issue.Severity)
		issue.Code = id(issue.Code)
		issue.Message = text(issue.Message)
		issue.OperationID = id(issue.OperationID)
		issue.Path = text(issue.Path)
		issue.Slot = id(issue.Slot)
		issue.SuggestedAnswer = text(issue.SuggestedAnswer)
		issue.Remediation = text(issue.Remediation)
	}
	return changed
}

func sanitizeRequestBody(body *RequestBodySummary, budget PromptBudget, changed bool) bool {
	body.Description, changed = sanitizePromptValue(body.Description, budget.MaxTextRunes, changed)
	body.Ref, changed = sanitizePromptValue(body.Ref, budget.MaxTextRunes, changed)
	body.ContentTypes, changed = sanitizeStringCollection(body.ContentTypes, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	if body.Schema != nil {
		changed = sanitizeSchemaSummary(body.Schema, budget, changed)
	}
	body.Fields, changed = sanitizeRequestFields(body.Fields, budget, changed)
	body.RequiredFieldPaths, changed = sanitizeStringCollection(body.RequiredFieldPaths, budget.MaxIdentifierRunes, budget.MaxFields, changed)
	return changed
}

func sanitizeResponseBody(body *ResponseBodySummary, budget PromptBudget, changed bool) bool {
	body.StatusCode, changed = sanitizePromptValue(body.StatusCode, budget.MaxIdentifierRunes, changed)
	body.Description, changed = sanitizePromptValue(body.Description, budget.MaxTextRunes, changed)
	body.Ref, changed = sanitizePromptValue(body.Ref, budget.MaxTextRunes, changed)
	body.ContentTypes, changed = sanitizeStringCollection(body.ContentTypes, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	if body.Schema != nil {
		changed = sanitizeSchemaSummary(body.Schema, budget, changed)
	}
	body.Fields, changed = sanitizeRequestFields(body.Fields, budget, changed)
	return changed
}

func sanitizeSchemaSummary(schema *SchemaSummary, budget PromptBudget, changed bool) bool {
	schema.Type, changed = sanitizePromptValue(schema.Type, budget.MaxIdentifierRunes, changed)
	schema.Format, changed = sanitizePromptValue(schema.Format, budget.MaxIdentifierRunes, changed)
	schema.Ref, changed = sanitizePromptValue(schema.Ref, budget.MaxTextRunes, changed)
	schema.Description, changed = sanitizePromptValue(schema.Description, budget.MaxTextRunes, changed)
	schema.Required, changed = sanitizeStringCollection(schema.Required, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	if len(schema.Properties) > budget.MaxFields {
		schema.Properties = schema.Properties[:budget.MaxFields]
		changed = true
	}
	for i := range schema.Properties {
		property := &schema.Properties[i]
		property.Name, changed = sanitizePromptValue(property.Name, budget.MaxIdentifierRunes, changed)
		property.Type, changed = sanitizePromptValue(property.Type, budget.MaxIdentifierRunes, changed)
		property.Format, changed = sanitizePromptValue(property.Format, budget.MaxIdentifierRunes, changed)
		property.Ref, changed = sanitizePromptValue(property.Ref, budget.MaxTextRunes, changed)
		property.Description, changed = sanitizePromptValue(property.Description, budget.MaxTextRunes, changed)
	}
	return changed
}

func sanitizeRequestFields(fields []RequestFieldSummary, budget PromptBudget, changed bool) ([]RequestFieldSummary, bool) {
	if len(fields) > budget.MaxFields {
		fields = fields[:budget.MaxFields]
		changed = true
	}
	for i := range fields {
		field := &fields[i]
		field.Path, changed = sanitizePromptValue(field.Path, budget.MaxIdentifierRunes, changed)
		field.Type, changed = sanitizePromptValue(field.Type, budget.MaxIdentifierRunes, changed)
		field.Format, changed = sanitizePromptValue(field.Format, budget.MaxIdentifierRunes, changed)
		field.Ref, changed = sanitizePromptValue(field.Ref, budget.MaxTextRunes, changed)
		field.Description, changed = sanitizePromptValue(field.Description, budget.MaxTextRunes, changed)
	}
	return fields, changed
}

func sanitizeSecuritySummary(security *SecuritySummary, budget PromptBudget, changed bool) bool {
	security.Name, changed = sanitizePromptValue(security.Name, budget.MaxIdentifierRunes, changed)
	security.Type, changed = sanitizePromptValue(security.Type, budget.MaxIdentifierRunes, changed)
	security.Scheme, changed = sanitizePromptValue(security.Scheme, budget.MaxIdentifierRunes, changed)
	security.In, changed = sanitizePromptValue(security.In, budget.MaxIdentifierRunes, changed)
	security.ParameterName, changed = sanitizePromptValue(security.ParameterName, budget.MaxIdentifierRunes, changed)
	security.AuthorizationURL, changed = sanitizePromptValue(security.AuthorizationURL, budget.MaxTextRunes, changed)
	security.TokenURL, changed = sanitizePromptValue(security.TokenURL, budget.MaxTextRunes, changed)
	security.RefreshURL, changed = sanitizePromptValue(security.RefreshURL, budget.MaxTextRunes, changed)
	security.Description, changed = sanitizePromptValue(security.Description, budget.MaxTextRunes, changed)
	security.Flows, changed = sanitizeStringCollection(security.Flows, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	security.Scopes, changed = sanitizeStringCollection(security.Scopes, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	security.Extensions, changed = sanitizeStringMap(security.Extensions, budget, changed)
	if len(security.OAuthFlows) > budget.MaxCollectionItems {
		security.OAuthFlows = security.OAuthFlows[:budget.MaxCollectionItems]
		changed = true
	}
	for i := range security.OAuthFlows {
		flow := &security.OAuthFlows[i]
		flow.Name, changed = sanitizePromptValue(flow.Name, budget.MaxIdentifierRunes, changed)
		flow.AuthorizationURL, changed = sanitizePromptValue(flow.AuthorizationURL, budget.MaxTextRunes, changed)
		flow.TokenURL, changed = sanitizePromptValue(flow.TokenURL, budget.MaxTextRunes, changed)
		flow.RefreshURL, changed = sanitizePromptValue(flow.RefreshURL, budget.MaxTextRunes, changed)
		flow.Scopes, changed = sanitizeStringCollection(flow.Scopes, budget.MaxIdentifierRunes, budget.MaxCollectionItems, changed)
	}
	return changed
}

func sanitizeStringCollection(values []string, maxRunes, maxItems int, changed bool) ([]string, bool) {
	if len(values) > maxItems {
		values = values[:maxItems]
		changed = true
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		sanitized, unsafe := sanitizePromptString(value, maxRunes)
		changed = changed || unsafe
		if sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return out, changed
}

func sanitizeStringMap(values map[string]string, budget PromptBudget, changed bool) (map[string]string, bool) {
	if len(values) == 0 {
		return nil, changed
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > budget.MaxCollectionItems {
		keys = keys[:budget.MaxCollectionItems]
		changed = true
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		safeKey, keyUnsafe := sanitizePromptString(key, budget.MaxIdentifierRunes)
		safeValue, valueUnsafe := sanitizePromptString(values[key], budget.MaxTextRunes)
		changed = changed || keyUnsafe || valueUnsafe
		if safeKey != "" {
			out[safeKey] = safeValue
		}
	}
	return out, changed
}

func sanitizePromptValue(value string, maxRunes int, changed bool) (string, bool) {
	out, unsafe := sanitizePromptString(value, maxRunes)
	return out, changed || unsafe
}

func sanitizePromptString(value string, maxRunes int) (string, bool) {
	original := value
	value = ansiControlSequence.ReplaceAllString(value, " ")
	var builder strings.Builder
	removedControl := value != original
	for _, r := range value {
		if unicode.IsControl(r) {
			builder.WriteByte(' ')
			removedControl = true
			continue
		}
		builder.WriteRune(r)
	}
	value = strings.Join(strings.Fields(builder.String()), " ")
	truncated := utf8.RuneCountInString(value) > maxRunes
	if truncated {
		runes := []rune(value)
		if maxRunes <= 1 {
			value = "…"
		} else {
			value = string(runes[:maxRunes-1]) + "…"
		}
	}
	return value, removedControl || truncated
}

func compactOperationDetails(operation *OperationSummary) {
	operation.Description = ""
	operation.Extensions = nil
	if utf8.RuneCountInString(operation.Summary) > 512 {
		operation.Summary, _ = sanitizePromptString(operation.Summary, 512)
	}
	for i := range operation.Parameters {
		operation.Parameters[i].Description = ""
	}
	if len(operation.Parameters) > 16 {
		operation.Parameters = operation.Parameters[:16]
	}
	if operation.RequestBody != nil {
		operation.RequestBody.Description = ""
		if operation.RequestBody.Schema != nil {
			operation.RequestBody.Schema.Description = ""
			operation.RequestBody.Schema.Properties = nil
		}
		if len(operation.RequestBody.Fields) > 32 {
			operation.RequestBody.Fields = operation.RequestBody.Fields[:32]
		}
		for i := range operation.RequestBody.Fields {
			operation.RequestBody.Fields[i].Description = ""
		}
	}
	if operation.ResponseBody != nil {
		operation.ResponseBody.Description = ""
		if operation.ResponseBody.Schema != nil {
			operation.ResponseBody.Schema.Description = ""
			operation.ResponseBody.Schema.Properties = nil
		}
		if len(operation.ResponseBody.Fields) > 32 {
			operation.ResponseBody.Fields = operation.ResponseBody.Fields[:32]
		}
		for i := range operation.ResponseBody.Fields {
			operation.ResponseBody.Fields[i].Description = ""
		}
	}
	if len(operation.SecurityRequirementSets) > 8 {
		operation.SecurityRequirementSets = operation.SecurityRequirementSets[:8]
	}
	for setIndex := range operation.SecurityRequirementSets {
		securitySet := &operation.SecurityRequirementSets[setIndex]
		if len(securitySet.Requirements) > 8 {
			securitySet.Requirements = securitySet.Requirements[:8]
		}
		for requirementIndex := range securitySet.Requirements {
			security := &securitySet.Requirements[requirementIndex]
			security.Description = ""
			if len(security.OAuthFlows) > 4 {
				security.OAuthFlows = security.OAuthFlows[:4]
			}
			if len(security.Scopes) > 8 {
				security.Scopes = security.Scopes[:8]
			}
		}
	}
	if len(operation.Tags) > 8 {
		operation.Tags = operation.Tags[:8]
	}
	if len(operation.ReadinessIssues) > 8 {
		operation.ReadinessIssues = operation.ReadinessIssues[:8]
	}
}

func minimalPromptOperation(operation OperationSummary, budget PromptBudget) OperationSummary {
	minimal := OperationSummary{
		ID: operation.ID, DocumentName: operation.DocumentName,
		DocumentPath: operation.DocumentPath, DocumentRelativePath: operation.DocumentRelativePath,
		DocumentURL: operation.DocumentURL, OperationID: operation.OperationID,
		Method: operation.Method, Path: operation.Path, Summary: operation.Summary,
		Score: operation.Score, Provenance: operation.Provenance,
	}
	minimal.Summary, _ = sanitizePromptString(minimal.Summary, min(256, budget.MaxTextRunes))
	return minimal
}

func promptJSONSize(value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return len(data)
}

func errorDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	var out []Diagnostic
	for _, diagnostic := range diagnostics {
		if strings.EqualFold(diagnostic.Severity, "error") {
			out = append(out, diagnostic)
		}
	}
	return out
}
