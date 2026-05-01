package apitools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeafAdapterBuildsReviewMetadata(t *testing.T) {
	set := ArtifactSet{
		Artifacts: []Artifact{{
			Path:      "intent.hcl",
			MediaType: "text/plain",
			Content:   []byte("binding = \"runtime.api_key\"\n"),
		}},
		Diagnostics:      []Diagnostic{{Severity: "warning", Code: "draft.review", Message: "review it"}},
		ReadinessIssues:  []ReadinessIssue{{Severity: "error", Code: "schema.ref_unresolved", Message: "missing ref"}},
		SymbolicBindings: []SymbolicBinding{{Name: "runtime.api_key", Kind: "apiKey"}},
		Assumptions:      []Assumption{{Text: "review-only", Source: "test"}},
		QuestionPlan:     QuestionPlan{Questions: []Question{{Prompt: "Which binding?"}}},
		Transcript:       &Transcript{Turns: []TranscriptTurn{{Source: "test", Content: "drafted"}}},
	}
	leaf := NewLeafAdapter(set, LeafOptions{Name: "Ticket", Source: "unit"})

	if got := leaf.ArtifactPaths(); len(got) != 1 || got[0] != "intent.hcl" {
		t.Fatalf("artifact paths = %#v", got)
	}
	if !leaf.HasBlockingIssues() {
		t.Fatal("expected blocking readiness issue")
	}
	if got := leaf.BindingNames(); len(got) != 1 || got[0] != "runtime.api_key" {
		t.Fatalf("binding names = %#v", got)
	}
	pkg := leaf.MinimumReviewPackage()
	if pkg.Name != "Ticket" || pkg.Source != "unit" || len(pkg.Artifacts) != 1 || len(pkg.TranscriptSummary) != 1 {
		t.Fatalf("review package = %#v", pkg)
	}
	leaf = leaf.WithReviewArtifact(ArtifactReview{Path: "review/summary.md", MediaType: "text/markdown", SizeBytes: 42})
	if _, ok := leaf.ReviewArtifact("review/summary.md"); ok {
		t.Fatal("ReviewArtifact should only read concrete artifacts")
	}
	if len(leaf.MinimumReviewPackage().Artifacts) != 2 {
		t.Fatalf("review artifacts = %#v", leaf.MinimumReviewPackage().Artifacts)
	}
}

func TestLeafAdapterReviewMarkdownIsDeferredAndSymbolic(t *testing.T) {
	leaf := NewLeafAdapter(ArtifactSet{
		Artifacts:        []Artifact{{Path: "project.md", MediaType: "text/markdown", Content: []byte("# Project\n")}},
		SymbolicBindings: []SymbolicBinding{{Name: "support_api", Kind: "apiKey"}},
		ReadinessIssues:  []ReadinessIssue{{Severity: "warning", Code: "draft.review", Message: "needs review"}},
		Diagnostics:      []Diagnostic{{Severity: "info", Code: "draft.info", Message: "info"}},
	}, LeafOptions{Name: "Review"})

	md := leaf.ReviewMarkdown()
	for _, want := range []string{"# Review", "review-only", "runtime binding", "`support_api`", "needs review", "No assumptions recorded"} {
		if !strings.Contains(md, want) {
			t.Fatalf("review markdown missing %q:\n%s", want, md)
		}
	}
	if strings.Contains(strings.ToLower(md), "execute now") || strings.Contains(strings.ToLower(md), "direct execution allowed") {
		t.Fatalf("review markdown contains direct execution language:\n%s", md)
	}
}

func TestLeafAdapterCredentialAuditFlagsLiteralSecrets(t *testing.T) {
	leaf := NewLeafAdapter(ArtifactSet{
		Artifacts: []Artifact{
			{Path: "symbolic.hcl", Content: []byte(`api_key = "runtime.support_api"`)},
			{Path: "secret.hcl", Content: []byte(`api_key = "sk-proj-abcdefghijklmnopqrst1234567890"`)},
		},
		SymbolicBindings: []SymbolicBinding{{Name: "runtime.support_api"}},
	}, LeafOptions{})

	audit := leaf.BindingAudit()
	if len(audit.DeclaredSymbolicBindings) != 1 || audit.DeclaredSymbolicBindings[0] != "runtime.support_api" {
		t.Fatalf("audit bindings = %#v", audit.DeclaredSymbolicBindings)
	}
	if len(audit.LiteralCredentialDiagnostics) != 1 || audit.LiteralCredentialDiagnostics[0].Path != "secret.hcl" {
		t.Fatalf("literal diagnostics = %#v", audit.LiteralCredentialDiagnostics)
	}
}

func TestArtifactOutputPathSafety(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{name: "normal", path: "dir/file.txt", ok: true},
		{name: "empty", path: "", ok: false},
		{name: "absolute", path: "/tmp/file.txt", ok: false},
		{name: "dot", path: ".", ok: false},
		{name: "traversal", path: "../file.txt", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ArtifactOutputPath(root, tt.path)
			if tt.ok && err != nil {
				t.Fatalf("ArtifactOutputPath error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("ArtifactOutputPath succeeded: %s", got)
			}
			if tt.ok && got != filepath.Join(root, filepath.FromSlash(tt.path)) {
				t.Fatalf("path = %q", got)
			}
		})
	}
}

func TestWriteArtifactSet(t *testing.T) {
	root := t.TempDir()
	written, err := WriteArtifactSet(context.Background(), root, ArtifactSet{Artifacts: []Artifact{
		{Path: "b.txt", Content: []byte("b")},
		{Path: "dir/a.txt", Content: []byte("a")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 || written[0] != filepath.Join(root, "b.txt") || written[1] != filepath.Join(root, "dir", "a.txt") {
		t.Fatalf("written = %#v", written)
	}
	if data, err := os.ReadFile(filepath.Join(root, "dir", "a.txt")); err != nil || string(data) != "a" {
		t.Fatalf("read written file = %q, %v", string(data), err)
	}
	if _, err := WriteArtifactSet(context.Background(), root, ArtifactSet{Artifacts: []Artifact{{Path: "../escape.txt"}}}); err == nil {
		t.Fatal("WriteArtifactSet accepted escaping path")
	}
}
