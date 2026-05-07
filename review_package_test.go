package apitools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildReviewPackageUsesBindingContractAndActions(t *testing.T) {
	pkg := BuildReviewPackage(ReviewPackageInput{
		Name:      "Ticket",
		Source:    "unit",
		Artifacts: []ReviewArtifactInput{{Path: "intent.hcl", MediaType: "text/hcl", SizeBytes: 12}},
		Diagnostics: []Diagnostic{{
			Severity: "warning",
			Code:     "review",
			Message:  "review it",
		}},
		BindingNames: []string{"runtime.api"},
		Transcript:   &Transcript{Turns: []TranscriptTurn{{Source: "user", Content: "make it"}}},
	})
	if pkg.Name != "Ticket" || len(pkg.Artifacts) != 1 || len(pkg.BindingNames) != 1 || len(pkg.TranscriptSummary) != 1 {
		t.Fatalf("package = %#v", pkg)
	}
	if len(pkg.RequiredReviewActions) == 0 {
		t.Fatal("expected review actions")
	}
}

func TestReviewHandoffInputsFromArtifactsDeduplicatesAndSorts(t *testing.T) {
	inputs := ReviewHandoffInputsFromArtifacts([]ReviewArtifactInput{
		{Path: "z.json", Purpose: "Z"},
		{Path: "a.json", Purpose: "A"},
		{Path: "z.json", Purpose: "duplicate"},
	}, ReviewHandoffInput{Path: "manifest.json", Purpose: "manifest", Required: true})
	if len(inputs) != 3 {
		t.Fatalf("inputs = %#v", inputs)
	}
	if inputs[0].Path != "a.json" || inputs[1].Path != "manifest.json" || inputs[2].Path != "z.json" {
		t.Fatalf("input order = %#v", inputs)
	}
}

func TestComputeReviewHandoffDigestIsStable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputs := []ReviewHandoffInput{
		{Path: "b.txt", Required: true},
		{Path: "a.txt", Required: true},
	}
	first, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{Root: root, Scope: "examples/unit", Version: "unit.digest.v1", Inputs: inputs})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{Root: root, Scope: "examples/unit", Version: "unit.digest.v1", Inputs: []ReviewHandoffInput{inputs[1], inputs[0]}})
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("digests = %q %q", first, second)
	}
	if _, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{Root: root, Inputs: []ReviewHandoffInput{{Path: "../escape", Required: true}}}); err == nil {
		t.Fatal("expected unsafe path failure")
	}
}
