package apitools

import (
	"context"
	"strings"
	"testing"
)

func TestNeutralAuthoringCoreBuildsContextAndDraftsWithTranscript(t *testing.T) {
	core := NewAuthoringCore()
	opContext, err := core.BuildContext(context.Background(), Brief{
		Text:        "Create one support ticket from runtime inputs.",
		ProjectName: "Support Ticket Draft",
	}, []OpenAPIDoc{{
		Name:    "support",
		Path:    "openapi/support.yaml",
		Content: []byte(openAPI3InventoryFixture()),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(opContext.Inventory.Operations) != 2 {
		t.Fatalf("operations = %#v", opContext.Inventory.Operations)
	}
	if len(opContext.Transcript.Turns) < 3 {
		t.Fatalf("context transcript = %#v", opContext.Transcript)
	}

	set, err := core.Draft(context.Background(), DraftInput{
		Context:              opContext,
		SelectedOperationIDs: []string{"createTicket"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Artifacts) != 2 {
		t.Fatalf("artifacts = %#v", set.Artifacts)
	}
	if set.Transcript == nil || len(set.Transcript.Turns) <= len(opContext.Transcript.Turns) {
		t.Fatalf("draft transcript = %#v", set.Transcript)
	}
	transcriptText := transcriptContent(set.Transcript)
	for _, want := range []string{"Create one support ticket", "Loaded OpenAPI document", "createTicket", "Rendered neutral draft artifact"} {
		if !strings.Contains(transcriptText, want) {
			t.Fatalf("transcript missing %q in %q", want, transcriptText)
		}
	}
}

func TestDraftFromOpenAPIAndDraftAndRender(t *testing.T) {
	core := NewAuthoringCore()
	opContext, set, err := DraftFromOpenAPI(context.Background(), core, Brief{
		Text:        "Create a ticket.",
		ProjectName: "Ticket",
	}, []OpenAPIDoc{{
		Name:    "support",
		Content: []byte(openAPI3InventoryFixture()),
	}}, []string{"createTicket"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opContext.Inventory.Operations) != 2 || len(set.Artifacts) != 2 {
		t.Fatalf("context/artifacts = %#v %#v", opContext, set.Artifacts)
	}

	renderer := recordingLeafRenderer{}
	leaf, rendered, diagnostics, err := DraftAndRender(context.Background(), core, renderer, DraftInput{
		Context:              opContext,
		SelectedOperationIDs: []string{"createTicket"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if !strings.Contains(artifactText(leaf.ArtifactSet, "project.md"), "# Ticket") || !strings.Contains(artifactText(rendered, "intent.copy.hcl"), `operation = "createTicket"`) {
		t.Fatalf("leaf/rendered artifacts:\n%s\n%s", artifactText(leaf.ArtifactSet, "project.md"), artifactText(rendered, "intent.copy.hcl"))
	}
}

func transcriptContent(transcript *Transcript) string {
	if transcript == nil {
		return ""
	}
	var b strings.Builder
	for _, turn := range transcript.Turns {
		b.WriteString(turn.Content)
		b.WriteByte('\n')
	}
	return b.String()
}

type recordingLeafRenderer struct{}

func (renderer recordingLeafRenderer) RenderLeaf(ctx context.Context, leaf LeafAdapter) (ArtifactSet, []Diagnostic, error) {
	_ = ctx
	intent := artifactText(leaf.ArtifactSet, "intent.hcl")
	return ArtifactSet{Artifacts: []Artifact{{
		Path:      "intent.copy.hcl",
		MediaType: "text/plain",
		Content:   []byte(intent),
	}}}, nil, nil
}

func artifactText(set ArtifactSet, path string) string {
	for _, artifact := range set.Artifacts {
		if artifact.Path == path {
			return string(artifact.Content)
		}
	}
	return ""
}
