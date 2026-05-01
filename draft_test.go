package apitools

import (
	"context"
	"strings"
	"testing"
)

func TestDraftArtifactsRendersNeutralArtifactsAndMetadata(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Query: "create ticket",
		Documents: []InventoryDocument{{
			Name:    "support",
			Path:    "openapi/support.yaml",
			Content: []byte(openAPI3InventoryFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	set, err := DraftArtifacts(context.Background(), DraftOptions{
		Brief:                "Create one support ticket from runtime inputs.",
		ProjectName:          "Support Ticket Draft",
		Inventory:            inventory,
		SelectedOperationIDs: []string{"createTicket"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Artifacts) != 2 {
		t.Fatalf("artifacts = %#v", set.Artifacts)
	}
	project := artifactContent(t, set, "project.md")
	intent := artifactContent(t, set, "intent.hcl")
	for _, want := range []string{"# Support Ticket Draft", "`subject`", "`priority`", "`tenant_id`", "apiKeyAuth", "review-only draft"} {
		if !strings.Contains(project+intent, want) {
			t.Fatalf("draft missing %q\nproject:\n%s\nintent:\n%s", want, project, intent)
		}
	}
	if !strings.Contains(intent, `operation = "createTicket"`) || !strings.Contains(intent, `binding "apikeyauth"`) {
		t.Fatalf("intent missing operation or binding:\n%s", intent)
	}
	if len(set.Slots) != 3 {
		t.Fatalf("slots = %#v", set.Slots)
	}
	if len(set.SymbolicBindings) != 1 || set.SymbolicBindings[0].Name != "apikeyauth" {
		t.Fatalf("bindings = %#v", set.SymbolicBindings)
	}
	if len(set.QuestionPlan.Questions) < 2 {
		t.Fatalf("question plan = %#v", set.QuestionPlan)
	}
}

func TestDraftArtifactsReportsUnknownSelection(t *testing.T) {
	set, err := DraftArtifacts(context.Background(), DraftOptions{
		ProjectName:          "Missing",
		SelectedOperationIDs: []string{"missingOperation"},
		Inventory: OperationInventory{
			Operations: []OperationSummary{{
				ID:          "known",
				OperationID: "known",
				Method:      "GET",
				Path:        "/known",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.ReadinessIssues) == 0 {
		t.Fatalf("expected readiness issues")
	}
	found := false
	for _, issue := range set.ReadinessIssues {
		if issue.Code == "draft.operation_not_found" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %#v", set.ReadinessIssues)
	}
	intent := artifactContent(t, set, "intent.hcl")
	if strings.Contains(intent, "known") {
		t.Fatalf("unknown selection should not draft known operation:\n%s", intent)
	}
}

func TestDraftArtifactsUsesRecursiveRequestFields(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name:    "nested",
			Content: []byte(openAPI3NestedRequestFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	set, err := DraftArtifacts(context.Background(), DraftOptions{
		ProjectName:          "Nested Draft",
		Inventory:            inventory,
		SelectedOperationIDs: []string{"createUser"},
	})
	if err != nil {
		t.Fatal(err)
	}
	project := artifactContent(t, set, "project.md")
	intent := artifactContent(t, set, "intent.hcl")
	for _, want := range []string{"`user_email`", "`groups_name`", `input "user_email"`, `input "groups_name"`} {
		if !strings.Contains(project+intent, want) {
			t.Fatalf("draft missing nested input %q\nproject:\n%s\nintent:\n%s", want, project, intent)
		}
	}
	for _, forbidden := range []string{"password", "api_key", "token"} {
		if strings.Contains(project+intent, forbidden) {
			t.Fatalf("draft leaked secret-like field %q\nproject:\n%s\nintent:\n%s", forbidden, project, intent)
		}
	}
	if !hasSlot(set.Slots, "user_email") || !hasSlot(set.Slots, "groups_name") {
		t.Fatalf("slots = %#v", set.Slots)
	}
}

func hasSlot(slots []Slot, name string) bool {
	for _, slot := range slots {
		if slot.Name == name {
			return true
		}
	}
	return false
}

func artifactContent(t *testing.T, set ArtifactSet, path string) string {
	t.Helper()
	for _, artifact := range set.Artifacts {
		if artifact.Path == path {
			return string(artifact.Content)
		}
	}
	t.Fatalf("artifact %s not found in %#v", path, set.Artifacts)
	return ""
}
