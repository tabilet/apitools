package apitools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFilesFindsValidOpenAPIDocuments(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLocalFile(t, filepath.Join(dir, "support.yaml"), `openapi: 3.0.0
info:
  title: Support Ticket API
  version: 1.0.0
  description: Fetch and update support tickets.
paths:
  /tickets:
    get:
      responses:
        "200":
          description: ok
`)
	writeLocalFile(t, filepath.Join(dir, "notes.yaml"), `not: openapi`)

	got, err := LocalFiles(context.Background(), LocalOptions{
		Dir:     dir,
		BaseDir: base,
		Query:   "support ticket",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].RelativePath != "openapi/support.yaml" {
		t.Fatalf("relative path = %q", got[0].RelativePath)
	}
	if got[0].Title != "Support Ticket API" || got[0].Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", got[0])
	}
	if got[0].Score == 0 {
		t.Fatalf("score should be positive")
	}
}

func TestLocalFilesFindsDraftOpenAPIDocuments(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLocalFile(t, filepath.Join(dir, "draft.yaml"), `openapi: 3.0.0
info:
  title: Draft API
  version: 1.0.0
paths:
  /items:
    get:
      operationId: listItems
`)

	got, err := LocalFiles(context.Background(), LocalOptions{Dir: dir, BaseDir: base, Query: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Draft API" || got[0].Metadata.OpenAPI != "3.0.0" {
		t.Fatalf("results = %#v", got)
	}
}

func TestLocalFilesSortsByScoreThenPath(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLocalFile(t, filepath.Join(dir, "z.yaml"), validLocalSpec("Billing API", "Invoices"))
	writeLocalFile(t, filepath.Join(dir, "a.yaml"), validLocalSpec("Mail API", "Send mail"))

	got, err := LocalFiles(context.Background(), LocalOptions{Dir: dir, BaseDir: base, Query: "mail"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].RelativePath != "openapi/a.yaml" {
		t.Fatalf("first = %#v", got)
	}
}

func TestLocalFilesRejectsMissingDir(t *testing.T) {
	_, err := LocalFiles(context.Background(), LocalOptions{})
	if err == nil || !strings.Contains(err.Error(), "directory is required") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func TestLocalFilesRejectsSymlinkedRoot(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real-openapi")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(base, "openapi")
	symlinkOrSkip(t, realDir, linkDir)

	_, err := LocalFiles(context.Background(), LocalOptions{Dir: linkDir, BaseDir: base})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestLocalFilesRejectsSymlinkedCandidates(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target.yaml")
	writeLocalFile(t, target, validLocalSpec("Target API", "Target"))
	symlinkOrSkip(t, target, filepath.Join(dir, "support.yaml"))

	_, err := LocalFiles(context.Background(), LocalOptions{Dir: dir, BaseDir: base})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestLocalFilesRejectsOversizedCandidates(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLocalFile(t, filepath.Join(dir, "support.yaml"), validLocalSpec("Support API", "Support"))

	_, err := LocalFiles(context.Background(), LocalOptions{Dir: dir, BaseDir: base, MaxBytes: 8})
	if err == nil || !strings.Contains(err.Error(), "larger than 8 bytes") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func writeLocalFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
}

func validLocalSpec(title, description string) string {
	return `openapi: 3.0.0
info:
  title: ` + title + `
  version: 1.0.0
  description: ` + description + `
paths:
  /items:
    get:
      responses:
        "200":
          description: ok
`
}
