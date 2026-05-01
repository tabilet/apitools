package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DraftOptions configures neutral review-only artifact rendering.
type DraftOptions struct {
	Brief                string             `json:"brief,omitempty"`
	ProjectName          string             `json:"project_name,omitempty"`
	Inventory            OperationInventory `json:"inventory,omitempty"`
	SelectedOperationIDs []string           `json:"selected_operation_ids,omitempty"`
	Transcript           *Transcript        `json:"transcript,omitempty"`
}

// DraftArtifacts renders neutral, review-only project.md and intent.hcl draft
// skeletons from an operation inventory. The returned intent is UWS-aligned
// authoring text, not a promise of compatibility with any caller-specific
// renderer.
func DraftArtifacts(ctx context.Context, opts DraftOptions) (ArtifactSet, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ArtifactSet{}, err
	}
	operations, issues := selectedDraftOperations(opts.Inventory.Operations, opts.SelectedOperationIDs)
	slots := draftSlots(operations)
	bindings := draftSymbolicBindings(operations)
	assumptions := []Assumption{{
		Text:   "Generated artifacts are neutral drafts and require caller-specific validation and rendering before use.",
		Source: "apitools",
	}}
	if len(operations) == 0 {
		issues = append(issues, ReadinessIssue{
			Severity:    "error",
			Code:        "draft.no_operations",
			Message:     "no operations were selected for the draft",
			Remediation: "Build an operation inventory and select at least one operation.",
		})
	}
	issues = append(issues, inventoryOperationIssues(operations)...)
	questions := questionPlanForIssues(issues, slots, bindings)
	set := ArtifactSet{
		Diagnostics:      opts.Inventory.Diagnostics,
		Inventory:        &opts.Inventory,
		SymbolicBindings: bindings,
		ReadinessIssues:  issues,
		Slots:            slots,
		Assumptions:      assumptions,
		QuestionPlan:     questions,
	}
	project := renderDraftProject(opts, operations, slots, bindings, issues)
	intent := renderDraftIntent(opts, operations, slots, bindings)
	set.Artifacts = []Artifact{
		{Path: "project.md", MediaType: "text/markdown", Content: []byte(project)},
		{Path: "intent.hcl", MediaType: "text/plain", Content: []byte(intent)},
	}
	transcript := draftTranscript(opts.Transcript, operations, slots, bindings, issues, set.Artifacts)
	set.Transcript = &transcript
	return set, nil
}

func selectedDraftOperations(operations []OperationSummary, selected []string) ([]OperationSummary, []ReadinessIssue) {
	if len(selected) == 0 {
		return append([]OperationSummary(nil), operations...), nil
	}
	byID := make(map[string]OperationSummary)
	for _, op := range operations {
		byID[op.ID] = op
		if op.OperationID != "" {
			byID[op.OperationID] = op
		}
	}
	out := make([]OperationSummary, 0, len(selected))
	var issues []ReadinessIssue
	seen := make(map[string]bool)
	for _, id := range selected {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		op, ok := byID[id]
		if !ok {
			issues = append(issues, ReadinessIssue{
				Severity:    "warning",
				Code:        "draft.operation_not_found",
				Message:     fmt.Sprintf("selected operation %q was not found in the inventory", id),
				OperationID: id,
				Remediation: "Select an operation by inventory id or OpenAPI operationId.",
			})
			continue
		}
		if seen[op.ID] {
			continue
		}
		out = append(out, op)
		seen[op.ID] = true
	}
	return out, issues
}

func draftSlots(operations []OperationSummary) []Slot {
	seen := make(map[string]Slot)
	for _, op := range operations {
		for _, parameter := range op.Parameters {
			if !parameter.Required {
				continue
			}
			name := sanitizeDraftName(parameter.Name)
			if name == "" {
				continue
			}
			seen[name] = Slot{
				Name:        name,
				Type:        firstNonEmpty(parameter.Type, "string"),
				Description: firstNonEmpty(parameter.Description, fmt.Sprintf("Required %s parameter for %s.", parameter.In, draftOperationLabel(op))),
				Required:    true,
				Sensitive:   looksCredentialLike(parameter.Name),
				Source:      op.ID,
			}
		}
		if op.RequestBody == nil {
			continue
		}
		if len(op.RequestBody.Fields) > 0 {
			for _, field := range requiredDraftRequestFields(op.RequestBody.Fields) {
				name := sanitizeDraftName(field.Path)
				if name == "" {
					continue
				}
				seen[name] = Slot{
					Name:        name,
					Type:        firstNonEmpty(field.Type, "string"),
					Description: firstNonEmpty(field.Description, fmt.Sprintf("Required request body field for %s.", draftOperationLabel(op))),
					Required:    true,
					Sensitive:   looksCredentialLike(field.Path),
					Source:      op.ID,
				}
			}
			continue
		}
		if op.RequestBody.Schema == nil {
			continue
		}
		for _, property := range op.RequestBody.Schema.Properties {
			if !property.Required {
				continue
			}
			name := sanitizeDraftName(property.Name)
			if name == "" {
				continue
			}
			seen[name] = Slot{
				Name:        name,
				Type:        firstNonEmpty(property.Type, "string"),
				Description: firstNonEmpty(property.Description, fmt.Sprintf("Required request body field for %s.", draftOperationLabel(op))),
				Required:    true,
				Sensitive:   looksCredentialLike(property.Name),
				Source:      op.ID,
			}
		}
	}
	return sortedSlots(seen)
}

func requiredDraftRequestFields(fields []RequestFieldSummary) []RequestFieldSummary {
	out := make([]RequestFieldSummary, 0, len(fields))
	for _, field := range fields {
		if !field.Required || field.Path == "" || hasRequiredChildRequestField(field.Path, fields) {
			continue
		}
		out = append(out, field)
	}
	return out
}

func hasRequiredChildRequestField(path string, fields []RequestFieldSummary) bool {
	for _, field := range fields {
		if !field.Required || field.Path == path {
			continue
		}
		if strings.HasPrefix(field.Path, path+".") || strings.HasPrefix(field.Path, path+"[]") {
			return true
		}
	}
	return false
}

func draftSymbolicBindings(operations []OperationSummary) []SymbolicBinding {
	seen := make(map[string]SymbolicBinding)
	for _, op := range operations {
		for _, security := range op.Security {
			if strings.TrimSpace(security.Name) == "" {
				continue
			}
			name := sanitizeDraftName(security.Name)
			if name == "" {
				continue
			}
			seen[name] = SymbolicBinding{
				Name:        name,
				Kind:        firstNonEmpty(security.Type, "security"),
				Source:      op.ID,
				Description: fmt.Sprintf("Symbolic binding for OpenAPI security scheme %q.", security.Name),
			}
		}
	}
	out := make([]SymbolicBinding, 0, len(seen))
	for _, binding := range seen {
		out = append(out, binding)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func draftTranscript(base *Transcript, operations []OperationSummary, slots []Slot, bindings []SymbolicBinding, issues []ReadinessIssue, artifacts []Artifact) Transcript {
	var transcript Transcript
	if base != nil {
		transcript.Turns = append(transcript.Turns, base.Turns...)
	}
	var operationLabels []string
	for _, op := range operations {
		operationLabels = append(operationLabels, draftOperationLabel(op))
	}
	if len(operationLabels) == 0 {
		operationLabels = append(operationLabels, "none")
	}
	transcript.Turns = append(transcript.Turns, TranscriptTurn{
		Role:    "tool",
		Content: "Selected OpenAPI operations for neutral draft: " + strings.Join(operationLabels, ", ") + ".",
		Source:  "openapi.draft.selection",
	})
	transcript.Turns = append(transcript.Turns, TranscriptTurn{
		Role:    "tool",
		Content: fmt.Sprintf("Inferred %d required slot(s), %d symbolic binding(s), and %d readiness issue(s).", len(slots), len(bindings), len(issues)),
		Source:  "openapi.draft.metadata",
	})
	var artifactPaths []string
	for _, artifact := range artifacts {
		artifactPaths = append(artifactPaths, artifact.Path)
	}
	transcript.Turns = append(transcript.Turns, TranscriptTurn{
		Role:    "tool",
		Content: "Rendered neutral draft artifact(s): " + strings.Join(artifactPaths, ", ") + ".",
		Source:  "openapi.draft.render",
	})
	return transcript
}

func inventoryOperationIssues(operations []OperationSummary) []ReadinessIssue {
	var issues []ReadinessIssue
	for _, op := range operations {
		issues = append(issues, op.ReadinessIssues...)
	}
	return issues
}

func questionPlanForIssues(issues []ReadinessIssue, slots []Slot, bindings []SymbolicBinding) QuestionPlan {
	var questions []Question
	if len(slots) > 0 {
		var names []string
		for _, slot := range slots {
			if slot.Required {
				names = append(names, slot.Name)
			}
		}
		if len(names) > 0 {
			questions = append(questions, Question{
				Name:      "required_inputs",
				Prompt:    "What runtime inputs should supply the required fields: " + strings.Join(names, ", ") + "?",
				IssueCode: "draft.required_inputs",
			})
		}
	}
	if len(bindings) > 0 {
		var names []string
		for _, binding := range bindings {
			names = append(names, binding.Name)
		}
		questions = append(questions, Question{
			Name:      "symbolic_bindings",
			Prompt:    "Which runtime binding names should satisfy the symbolic security bindings: " + strings.Join(names, ", ") + "?",
			IssueCode: "draft.symbolic_bindings",
		})
	}
	for _, issue := range issues {
		if issue.Code == "schema.ref_unresolved" {
			questions = append(questions, Question{
				Name:      "resolve_schema_refs",
				Prompt:    "Should referenced schemas be resolved before caller-specific rendering?",
				Options:   []string{"Resolve before rendering", "Keep as review-only references"},
				IssueCode: issue.Code,
			})
			break
		}
	}
	return QuestionPlan{Questions: questions}
}

func renderDraftProject(opts DraftOptions, operations []OperationSummary, slots []Slot, bindings []SymbolicBinding, issues []ReadinessIssue) string {
	var b bytes.Buffer
	name := firstNonEmpty(opts.ProjectName, "OpenAPI Authoring Draft")
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintln(&b, "## Goal")
	if strings.TrimSpace(opts.Brief) != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(opts.Brief))
	} else {
		fmt.Fprintln(&b, "\nDraft an OpenAPI-backed workflow from the selected operations.")
	}
	fmt.Fprintln(&b, "\n## Inputs")
	if len(slots) == 0 {
		fmt.Fprintln(&b, "\n- No required runtime inputs were inferred from the selected operations.")
	} else {
		fmt.Fprintln(&b)
		for _, slot := range slots {
			sensitive := ""
			if slot.Sensitive {
				sensitive = " sensitive"
			}
			fmt.Fprintf(&b, "- `%s` (%s,%s required): %s\n", slot.Name, firstNonEmpty(slot.Type, "string"), sensitive, firstNonEmpty(slot.Description, "Review required input."))
		}
	}
	fmt.Fprintln(&b, "\n## Outputs\n\n- Review and name the expected workflow outputs before caller-specific rendering.")
	fmt.Fprintln(&b, "\n## Data Flow")
	if len(operations) == 0 {
		fmt.Fprintln(&b, "\n- No operation data flow is available yet.")
	} else {
		fmt.Fprintln(&b)
		for i, op := range operations {
			fmt.Fprintf(&b, "- Step %d calls `%s` (`%s %s`).\n", i+1, draftOperationLabel(op), op.Method, op.Path)
		}
	}
	fmt.Fprintln(&b, "\n## External Systems and OpenAPI")
	if len(operations) == 0 {
		fmt.Fprintln(&b, "\n- OpenAPI: select at least one operation.")
	} else {
		fmt.Fprintln(&b)
		for _, op := range operations {
			fmt.Fprintf(&b, "- `%s`: %s\n", draftOperationLabel(op), firstNonEmpty(op.DocumentPath, op.DocumentURL, op.DocumentName))
		}
	}
	fmt.Fprintln(&b, "\n## Credentials and Secrets")
	if len(bindings) == 0 {
		fmt.Fprintln(&b, "\n- No OpenAPI security bindings were inferred from the selected operations.")
	} else {
		fmt.Fprintln(&b)
		for _, binding := range bindings {
			fmt.Fprintf(&b, "- `%s`: %s\n", binding.Name, binding.Description)
		}
	}
	fmt.Fprintln(&b, "\n## Safety and Approval Boundary\n\n- This is a review-only draft.")
	fmt.Fprintln(&b, "- Validate and render through the caller-specific leaf adapter before execution.")
	fmt.Fprintln(&b, "- Do not include secret values in prompts, artifacts, logs, or committed files.")
	fmt.Fprintln(&b, "\n## Fallback Behavior\n\n- Stop if required inputs, OpenAPI operations, schema references, or symbolic bindings are unresolved.")
	if len(issues) > 0 {
		fmt.Fprintln(&b, "\n## Readiness Issues")
		for _, issue := range issues {
			fmt.Fprintf(&b, "\n- `%s`: %s", firstNonEmpty(issue.Code, issue.Severity), issue.Message)
			if issue.Remediation != "" {
				fmt.Fprintf(&b, " %s", issue.Remediation)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func renderDraftIntent(opts DraftOptions, operations []OperationSummary, slots []Slot, bindings []SymbolicBinding) string {
	var b bytes.Buffer
	workflowName := sanitizeDraftName(firstNonEmpty(opts.ProjectName, "openapi_authoring_draft"))
	if workflowName == "" {
		workflowName = "openapi_authoring_draft"
	}
	fmt.Fprintln(&b, "# Review-only neutral draft. Caller-specific renderers must validate before execution.")
	fmt.Fprintln(&b, "# This file declares symbolic bindings by name only and contains no credential values.")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "workflow {")
	fmt.Fprintf(&b, "  name        = %s\n", hclString(workflowName))
	fmt.Fprintf(&b, "  description = %s\n", hclString(firstNonEmpty(opts.Brief, "Draft OpenAPI-backed workflow.")))
	fmt.Fprintln(&b, "}")
	fmt.Fprintln(&b)
	for _, slot := range slots {
		fmt.Fprintf(&b, "input %s {\n", hclString(slot.Name))
		fmt.Fprintf(&b, "  type     = %s\n", hclString(firstNonEmpty(slot.Type, "string")))
		fmt.Fprintf(&b, "  required = %t\n", slot.Required)
		if slot.Sensitive {
			fmt.Fprintln(&b, "  sensitive = true")
		}
		if slot.Description != "" {
			fmt.Fprintf(&b, "  description = %s\n", hclString(slot.Description))
		}
		fmt.Fprintln(&b, "}")
		fmt.Fprintln(&b)
	}
	for _, binding := range bindings {
		fmt.Fprintf(&b, "binding %s {\n", hclString(binding.Name))
		fmt.Fprintf(&b, "  kind = %s\n", hclString(firstNonEmpty(binding.Kind, "security")))
		if binding.Description != "" {
			fmt.Fprintf(&b, "  description = %s\n", hclString(binding.Description))
		}
		fmt.Fprintln(&b, "}")
		fmt.Fprintln(&b)
	}
	for i, op := range operations {
		stepName := sanitizeDraftName(firstNonEmpty(op.OperationID, op.ID, fmt.Sprintf("step_%d", i+1)))
		if stepName == "" {
			stepName = fmt.Sprintf("step_%d", i+1)
		}
		fmt.Fprintf(&b, "step %s {\n", hclString(stepName))
		fmt.Fprintln(&b, "  type = \"openapi\"")
		fmt.Fprintf(&b, "  do   = %s\n", hclString(firstNonEmpty(op.Summary, op.Description, "Call "+draftOperationLabel(op)+".")))
		if op.DocumentPath != "" {
			fmt.Fprintf(&b, "  openapi = %s\n", hclString(op.DocumentPath))
		} else if op.DocumentURL != "" {
			fmt.Fprintf(&b, "  openapi = %s\n", hclString(op.DocumentURL))
		}
		fmt.Fprintf(&b, "  operation = %s\n", hclString(firstNonEmpty(op.OperationID, op.ID)))
		if i > 0 {
			prev := sanitizeDraftName(firstNonEmpty(operations[i-1].OperationID, operations[i-1].ID, fmt.Sprintf("step_%d", i)))
			fmt.Fprintf(&b, "  depends_on = [%s]\n", hclString(prev))
		}
		stepInputs := stepInputNames(op, slots)
		if len(stepInputs) > 0 {
			fmt.Fprintln(&b, "  with = {")
			for _, input := range stepInputs {
				fmt.Fprintf(&b, "    %s = %s\n", input, hclString("inputs."+input))
			}
			fmt.Fprintln(&b, "  }")
		}
		fmt.Fprintln(&b, "}")
		fmt.Fprintln(&b)
	}
	if len(operations) > 0 {
		last := operations[len(operations)-1]
		lastName := sanitizeDraftName(firstNonEmpty(last.OperationID, last.ID, "last_step"))
		fmt.Fprintln(&b, "output \"result\" {")
		fmt.Fprintf(&b, "  from = %s\n", hclString(lastName+".received_body"))
		fmt.Fprintln(&b, "}")
	}
	return b.String()
}

func stepInputNames(op OperationSummary, slots []Slot) []string {
	slotBySource := make(map[string]bool)
	for _, slot := range slots {
		if slot.Source == op.ID {
			slotBySource[slot.Name] = true
		}
	}
	var names []string
	for name := range slotBySource {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedSlots(values map[string]Slot) []Slot {
	out := make([]Slot, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func draftOperationLabel(op OperationSummary) string {
	return firstNonEmpty(op.OperationID, op.ID, op.Method+" "+op.Path)
}

func sanitizeDraftName(value string) string {
	name := sanitizeName(value)
	if name == "" {
		return ""
	}
	return strings.ReplaceAll(name, "-", "_")
}

func looksCredentialLike(value string) bool {
	value = strings.ToLower(value)
	for _, token := range []string{"api_key", "apikey", "authorization", "auth", "token", "secret", "password", "credential"} {
		if strings.Contains(value, token) {
			return true
		}
	}
	return false
}

func hclString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}
