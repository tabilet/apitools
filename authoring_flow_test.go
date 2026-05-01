package apitools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFlowParseValidateRenderAndDiagnosticError(t *testing.T) {
	flow := Flow[string]{
		Parser:    testAuthoringAdapter{},
		Renderer:  testAuthoringAdapter{},
		Validator: testAuthoringAdapter{},
	}
	draft, artifacts, diagnostics, err := flow.ParseValidateRender(context.Background(), Artifact{
		Path:    "intent.txt",
		Content: []byte("ready"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft != "ready" || HasErrors(diagnostics) {
		t.Fatalf("draft/diagnostics = %q %#v", draft, diagnostics)
	}
	if len(artifacts.Artifacts) != 1 || string(artifacts.Artifacts[0].Content) != "ready" {
		t.Fatalf("artifacts = %#v", artifacts)
	}

	_, _, diagnostics, err = flow.ParseValidateRender(context.Background(), Artifact{Content: []byte("bad")})
	if err != nil {
		t.Fatal(err)
	}
	if !HasErrors(diagnostics) {
		t.Fatalf("expected error diagnostics: %#v", diagnostics)
	}
	if got := (DiagnosticError{Diagnostics: diagnostics}).Error(); !strings.Contains(got, "bad draft") {
		t.Fatalf("diagnostic error = %q", got)
	}
}

func TestFlowRefineValidateRenderReportsRequiredSlots(t *testing.T) {
	flow := Flow[string]{
		Renderer:     testAuthoringAdapter{},
		Validator:    testAuthoringAdapter{},
		Refiner:      testAuthoringAdapter{},
		SlotProvider: testAuthoringAdapter{},
	}
	draft, artifacts, diagnostics, err := flow.RefineValidateRender(context.Background(), "needs-slot", []TranscriptTurn{{Role: "user", Content: "fill"}})
	if err != nil {
		t.Fatal(err)
	}
	if draft != "needs-slot refined" {
		t.Fatalf("draft = %q", draft)
	}
	if len(artifacts.Artifacts) != 0 || !HasErrors(diagnostics) {
		t.Fatalf("artifacts/diagnostics = %#v %#v", artifacts, diagnostics)
	}
}

type testAuthoringAdapter struct{}

func (testAuthoringAdapter) ParseIntent(ctx context.Context, artifact Artifact) (string, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if len(artifact.Content) == 0 {
		return "", nil, errors.New("content required")
	}
	return string(artifact.Content), nil, nil
}

func (testAuthoringAdapter) RenderIntent(ctx context.Context, draft string) (ArtifactSet, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactSet{}, nil, err
	}
	return ArtifactSet{Artifacts: []Artifact{{Path: "intent.txt", MediaType: "text/plain", Content: []byte(draft)}}}, nil, nil
}

func (testAuthoringAdapter) ValidateIntent(ctx context.Context, draft string) []Diagnostic {
	if err := ctx.Err(); err != nil {
		return []Diagnostic{{Severity: "error", Message: err.Error()}}
	}
	if draft == "bad" {
		return []Diagnostic{{Severity: "error", Code: "bad", Message: "bad draft"}}
	}
	return nil
}

func (testAuthoringAdapter) MissingSlots(ctx context.Context, draft string) []Slot {
	_ = ctx
	if strings.Contains(draft, "needs-slot") {
		return []Slot{{Name: "account_id", Required: true, BindingRef: BindingRef{Name: "account"}}}
	}
	return nil
}

func (testAuthoringAdapter) RefineIntent(ctx context.Context, draft string, transcript []TranscriptTurn) (string, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if len(transcript) > 0 {
		return draft + " refined", nil, nil
	}
	return draft, nil, nil
}
