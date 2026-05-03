package apitools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Transcript records the authoring conversation and tool observations that led
// to draft artifacts.
type Transcript struct {
	Turns []TranscriptTurn `json:"turns,omitempty"`
}

// TranscriptTurn is one ordered entry in an authoring transcript.
type TranscriptTurn struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Diagnostic describes an authoring or validation issue.
type Diagnostic struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// Slot describes a missing or variable value needed by a draft artifact.
type Slot struct {
	Name        string     `json:"name"`
	Type        string     `json:"type,omitempty"`
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Sensitive   bool       `json:"sensitive,omitempty"`
	Source      string     `json:"source,omitempty"`
	Value       any        `json:"value,omitempty"`
	BindingRef  BindingRef `json:"binding_ref,omitempty"`
}

// Assumption is a provisional authoring choice that must remain reviewable.
type Assumption struct {
	ID      string     `json:"id,omitempty"`
	Text    string     `json:"text"`
	Source  string     `json:"source,omitempty"`
	Binding BindingRef `json:"binding,omitempty"`
}

// BindingRef identifies a runtime-provided binding referenced by generated
// authoring metadata without resolving the binding value.
type BindingRef struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`
}

// SymbolicBinding names a runtime-provided capability without resolving it.
type SymbolicBinding struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
}

// ReadinessIssue explains why an artifact or operation needs more review before
// validation, rendering, or execution.
type ReadinessIssue struct {
	Severity        string `json:"severity"`
	Code            string `json:"code,omitempty"`
	Message         string `json:"message"`
	OperationID     string `json:"operation_id,omitempty"`
	Path            string `json:"path,omitempty"`
	Slot            string `json:"slot,omitempty"`
	SuggestedAnswer string `json:"suggested_answer,omitempty"`
	Remediation     string `json:"remediation,omitempty"`
}

// QuestionPlan lists clarification questions that would reduce ambiguity.
type QuestionPlan struct {
	Questions []Question `json:"questions,omitempty"`
}

// Question is one caller-facing clarification prompt.
type Question struct {
	Name      string   `json:"name,omitempty"`
	Prompt    string   `json:"prompt"`
	Options   []string `json:"options,omitempty"`
	IssueCode string   `json:"issue_code,omitempty"`
}

// InteractiveQuestion is one next-question decision in an interactive
// authoring loop.
type InteractiveQuestion struct {
	Prompt          string   `json:"prompt"`
	SuggestedAnswer string   `json:"suggested_answer,omitempty"`
	Slots           []string `json:"slots,omitempty"`
	Grouped         bool     `json:"grouped,omitempty"`
}

// Artifact is a generated draft file or metadata payload.
type Artifact struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	Content   []byte `json:"content,omitempty"`
}

// ArtifactSet groups draft artifacts with the review metadata used to produce
// them.
type ArtifactSet struct {
	Artifacts            []Artifact            `json:"artifacts,omitempty"`
	Diagnostics          []Diagnostic          `json:"diagnostics,omitempty"`
	Transcript           *Transcript           `json:"transcript,omitempty"`
	Inventory            *OperationInventory   `json:"inventory,omitempty"`
	DocumentationContext *DocumentationContext `json:"documentation_context,omitempty"`
	SymbolicBindings     []SymbolicBinding     `json:"symbolic_bindings,omitempty"`
	ReadinessIssues      []ReadinessIssue      `json:"readiness_issues,omitempty"`
	Slots                []Slot                `json:"slots,omitempty"`
	Assumptions          []Assumption          `json:"assumptions,omitempty"`
	QuestionPlan         QuestionPlan          `json:"question_plan,omitempty"`
}

func (set ArtifactSet) MarshalJSON() ([]byte, error) {
	type artifactSetJSON struct {
		Artifacts            []Artifact            `json:"artifacts,omitempty"`
		Diagnostics          []Diagnostic          `json:"diagnostics,omitempty"`
		Transcript           *Transcript           `json:"transcript,omitempty"`
		Inventory            *OperationInventory   `json:"inventory,omitempty"`
		DocumentationContext *DocumentationContext `json:"documentation_context,omitempty"`
		SymbolicBindings     []SymbolicBinding     `json:"symbolic_bindings,omitempty"`
		ReadinessIssues      []ReadinessIssue      `json:"readiness_issues,omitempty"`
		Slots                []Slot                `json:"slots,omitempty"`
		Assumptions          []Assumption          `json:"assumptions,omitempty"`
		QuestionPlan         *QuestionPlan         `json:"question_plan,omitempty"`
	}
	out := artifactSetJSON{
		Artifacts:            set.Artifacts,
		Diagnostics:          set.Diagnostics,
		Transcript:           set.Transcript,
		Inventory:            set.Inventory,
		DocumentationContext: set.DocumentationContext,
		SymbolicBindings:     set.SymbolicBindings,
		ReadinessIssues:      set.ReadinessIssues,
		Slots:                set.Slots,
		Assumptions:          set.Assumptions,
	}
	if len(set.QuestionPlan.Questions) > 0 {
		questionPlan := set.QuestionPlan
		out.QuestionPlan = &questionPlan
	}
	return json.Marshal(out)
}

// Parser decodes an artifact into a typed draft.
type Parser[T any] interface {
	ParseIntent(ctx context.Context, artifact Artifact) (T, []Diagnostic, error)
}

// Renderer renders a typed draft into one or more artifacts.
type Renderer[T any] interface {
	RenderIntent(ctx context.Context, draft T) (ArtifactSet, []Diagnostic, error)
}

// Validator reports validation diagnostics for a typed draft.
type Validator[T any] interface {
	ValidateIntent(ctx context.Context, draft T) []Diagnostic
}

// SlotProvider reports missing caller-provided slots for a typed draft.
type SlotProvider[T any] interface {
	MissingSlots(ctx context.Context, draft T) []Slot
}

// Refiner applies transcript-driven updates to a typed draft.
type Refiner[T any] interface {
	RefineIntent(ctx context.Context, draft T, transcript []TranscriptTurn) (T, []Diagnostic, error)
}

// ChatClient is the minimal chat interface used by authoring flows.
type ChatClient interface {
	Complete(ctx context.Context, transcript []TranscriptTurn) (TranscriptTurn, error)
}

// StructuredChatClient extends ChatClient with schema-constrained responses.
type StructuredChatClient interface {
	ChatClient
	CompleteStructured(ctx context.Context, transcript []TranscriptTurn, schema any, out any) error
}

// Flow composes generic parse, refine, validate, slot, and render steps.
type Flow[T any] struct {
	Parser       Parser[T]
	Renderer     Renderer[T]
	Validator    Validator[T]
	SlotProvider SlotProvider[T]
	Refiner      Refiner[T]
}

func (flow Flow[T]) ParseValidateRender(ctx context.Context, artifact Artifact) (T, ArtifactSet, []Diagnostic, error) {
	var zero T
	if flow.Parser == nil {
		return zero, ArtifactSet{}, nil, fmt.Errorf("intent parser is required")
	}
	draft, diagnostics, err := flow.Parser.ParseIntent(ctx, artifact)
	if err != nil {
		return zero, ArtifactSet{}, diagnostics, err
	}
	diagnostics = append(diagnostics, flow.validate(ctx, draft)...)
	if HasErrors(diagnostics) {
		return draft, ArtifactSet{}, diagnostics, nil
	}
	artifacts, renderDiagnostics, err := flow.render(ctx, draft)
	diagnostics = append(diagnostics, renderDiagnostics...)
	return draft, artifacts, diagnostics, err
}

func (flow Flow[T]) RefineValidateRender(ctx context.Context, draft T, transcript []TranscriptTurn) (T, ArtifactSet, []Diagnostic, error) {
	if flow.Refiner != nil {
		refined, diagnostics, err := flow.Refiner.RefineIntent(ctx, draft, transcript)
		if err != nil {
			return refined, ArtifactSet{}, diagnostics, err
		}
		draft = refined
	}
	diagnostics := flow.validate(ctx, draft)
	if flow.SlotProvider != nil {
		for _, slot := range flow.SlotProvider.MissingSlots(ctx, draft) {
			if slot.Required {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "error",
					Code:     "missing_slot",
					Message:  fmt.Sprintf("required slot %q is missing", slot.Name),
					Path:     slot.Name,
				})
			}
		}
	}
	if HasErrors(diagnostics) {
		return draft, ArtifactSet{}, diagnostics, nil
	}
	artifacts, renderDiagnostics, err := flow.render(ctx, draft)
	diagnostics = append(diagnostics, renderDiagnostics...)
	return draft, artifacts, diagnostics, err
}

func (flow Flow[T]) validate(ctx context.Context, draft T) []Diagnostic {
	if flow.Validator == nil {
		return nil
	}
	return flow.Validator.ValidateIntent(ctx, draft)
}

func (flow Flow[T]) render(ctx context.Context, draft T) (ArtifactSet, []Diagnostic, error) {
	if flow.Renderer == nil {
		return ArtifactSet{}, nil, fmt.Errorf("intent renderer is required")
	}
	return flow.Renderer.RenderIntent(ctx, draft)
}

// HasErrors reports whether any diagnostic has error severity.
func HasErrors(diagnostics []Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			return true
		}
	}
	return false
}

// SortArtifacts orders artifacts by stable path and media type.
func SortArtifacts(set ArtifactSet) {
	sort.Slice(set.Artifacts, func(i, j int) bool {
		if set.Artifacts[i].Path != set.Artifacts[j].Path {
			return set.Artifacts[i].Path < set.Artifacts[j].Path
		}
		return strings.Compare(set.Artifacts[i].MediaType, set.Artifacts[j].MediaType) < 0
	})
}

// DiagnosticError wraps diagnostics as an error value.
type DiagnosticError struct {
	Diagnostics []Diagnostic
}

func (err DiagnosticError) Error() string {
	if len(err.Diagnostics) == 0 {
		return "diagnostics failed"
	}
	messages := make([]string, 0, len(err.Diagnostics))
	for _, diagnostic := range err.Diagnostics {
		if strings.TrimSpace(diagnostic.Message) != "" {
			messages = append(messages, diagnostic.Message)
		}
	}
	if len(messages) == 0 {
		return "diagnostics failed"
	}
	return strings.Join(messages, "; ")
}
