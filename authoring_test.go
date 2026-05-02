package apitools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestHasErrors(t *testing.T) {
	if HasErrors(nil) {
		t.Error("nil should be no errors")
	}
	if HasErrors([]Diagnostic{{Severity: "warning"}, {Severity: "info"}}) {
		t.Error("warnings should not count as errors")
	}
	if !HasErrors([]Diagnostic{{Severity: "warning"}, {Severity: "error"}}) {
		t.Error("error severity should be detected")
	}
}

func TestSortArtifactsOrdersByPathThenMediaType(t *testing.T) {
	set := ArtifactSet{Artifacts: []Artifact{
		{Path: "z.md", MediaType: "text/markdown"},
		{Path: "a.md", MediaType: "text/markdown"},
		{Path: "a.md", MediaType: "application/json"},
	}}
	SortArtifacts(set)
	if set.Artifacts[0].Path != "a.md" || set.Artifacts[0].MediaType != "application/json" {
		t.Errorf("first = %#v", set.Artifacts[0])
	}
	if set.Artifacts[2].Path != "z.md" {
		t.Errorf("last = %#v", set.Artifacts[2])
	}
}

func TestDiagnosticErrorJoinsMessages(t *testing.T) {
	err := DiagnosticError{Diagnostics: []Diagnostic{
		{Severity: "error", Message: "first"},
		{Severity: "error", Message: "  "},
		{Severity: "error", Message: "second"},
	}}
	if got := err.Error(); got != "first; second" {
		t.Errorf("got %q", got)
	}
	if got := (DiagnosticError{}).Error(); got != "diagnostics failed" {
		t.Errorf("empty case: %q", got)
	}
}

func TestArtifactSetMarshalJSONOmitsEmptyQuestionPlan(t *testing.T) {
	data, err := json.Marshal(ArtifactSet{Artifacts: []Artifact{{Path: "x", MediaType: "text/plain"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "question_plan") {
		t.Errorf("empty question plan should be omitted: %s", data)
	}
}

func TestArtifactSetMarshalJSONIncludesPopulatedQuestionPlan(t *testing.T) {
	data, err := json.Marshal(ArtifactSet{
		QuestionPlan: QuestionPlan{Questions: []Question{{Prompt: "Why?"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "question_plan") || !strings.Contains(string(data), "Why?") {
		t.Errorf("populated plan missing: %s", data)
	}
}

type stubParser struct{ err error }

func (s stubParser) ParseIntent(_ context.Context, _ Artifact) (string, []Diagnostic, error) {
	return "draft", nil, s.err
}

type stubRenderer struct{ artifacts []Artifact }

func (s stubRenderer) RenderIntent(_ context.Context, _ string) (ArtifactSet, []Diagnostic, error) {
	return ArtifactSet{Artifacts: s.artifacts}, nil, nil
}

type stubValidator struct{ diagnostics []Diagnostic }

func (s stubValidator) ValidateIntent(_ context.Context, _ string) []Diagnostic {
	return s.diagnostics
}

type stubSlots struct{ slots []Slot }

func (s stubSlots) MissingSlots(_ context.Context, _ string) []Slot { return s.slots }

type stubRefiner struct {
	out string
	err error
}

func (s stubRefiner) RefineIntent(_ context.Context, _ string, _ []TranscriptTurn) (string, []Diagnostic, error) {
	return s.out, nil, s.err
}

func TestFlowParseValidateRenderHappyPath(t *testing.T) {
	flow := Flow[string]{
		Parser:    stubParser{},
		Renderer:  stubRenderer{artifacts: []Artifact{{Path: "out.txt"}}},
		Validator: stubValidator{},
	}
	draft, set, diagnostics, err := flow.ParseValidateRender(context.Background(), Artifact{Path: "in.txt"})
	if err != nil || draft != "draft" || len(set.Artifacts) != 1 || len(diagnostics) != 0 {
		t.Fatalf("draft=%q set=%#v diagnostics=%#v err=%v", draft, set, diagnostics, err)
	}
}

func TestFlowParseValidateRenderRequiresParser(t *testing.T) {
	flow := Flow[string]{}
	_, _, _, err := flow.ParseValidateRender(context.Background(), Artifact{})
	if err == nil || !strings.Contains(err.Error(), "parser") {
		t.Fatalf("expected parser error, got %v", err)
	}
}

func TestFlowParseValidateRenderShortCircuitsOnDiagnosticError(t *testing.T) {
	flow := Flow[string]{
		Parser:    stubParser{},
		Renderer:  stubRenderer{artifacts: []Artifact{{Path: "out"}}},
		Validator: stubValidator{diagnostics: []Diagnostic{{Severity: "error", Message: "bad"}}},
	}
	_, set, diagnostics, err := flow.ParseValidateRender(context.Background(), Artifact{})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Artifacts) != 0 || !HasErrors(diagnostics) {
		t.Fatalf("set should be empty when validation fails: set=%#v diagnostics=%#v", set, diagnostics)
	}
}

func TestFlowParseValidateRenderPropagatesParserError(t *testing.T) {
	flow := Flow[string]{Parser: stubParser{err: errors.New("nope")}}
	_, _, _, err := flow.ParseValidateRender(context.Background(), Artifact{})
	if err == nil || err.Error() != "nope" {
		t.Fatalf("expected parser error, got %v", err)
	}
}

func TestFlowRefineValidateRenderReportsMissingSlots(t *testing.T) {
	flow := Flow[string]{
		Renderer:     stubRenderer{},
		Validator:    stubValidator{},
		SlotProvider: stubSlots{slots: []Slot{{Name: "endpoint", Required: true}}},
		Refiner:      stubRefiner{out: "refined"},
	}
	_, _, diagnostics, err := flow.RefineValidateRender(context.Background(), "draft", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !HasErrors(diagnostics) {
		t.Fatalf("expected missing-slot error: %#v", diagnostics)
	}
}

func TestFlowRefineValidateRenderRequiresRendererWhenInvoked(t *testing.T) {
	flow := Flow[string]{Validator: stubValidator{}}
	_, _, _, err := flow.RefineValidateRender(context.Background(), "draft", nil)
	if err == nil || !strings.Contains(err.Error(), "renderer") {
		t.Fatalf("expected renderer error, got %v", err)
	}
}
