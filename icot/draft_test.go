// Copyright (c) Greetingland LLC

package icot

import (
	"os"
	"path/filepath"
	"testing"
)

type session struct {
	Workflow string   `yaml:"workflow"`
	Steps    []string `yaml:"steps,omitempty"`
}

func TestDraftPath(t *testing.T) {
	got := DraftPath("/tmp/example")
	want := filepath.Join("/tmp/example", ".icot", "session.yaml")
	if got != want {
		t.Fatalf("DraftPath = %q, want %q", got, want)
	}
}

func TestLoadDraftMissingReturnsZero(t *testing.T) {
	got, ok, err := LoadDraft[session](filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("ok = true on missing file")
	}
	if got.Workflow != "" || len(got.Steps) != 0 {
		t.Fatalf("expected zero value, got %+v", got)
	}
}

func TestSaveAndLoadDraftRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := DraftPath(dir)
	want := session{Workflow: "demo", Steps: []string{"a", "b"}}
	if err := SaveDraft(path, want); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	got, ok, err := LoadDraft[session](path)
	if err != nil {
		t.Fatalf("LoadDraft: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false after SaveDraft")
	}
	if got.Workflow != want.Workflow || len(got.Steps) != len(want.Steps) {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestSaveDraftEmptyPathIsNoop(t *testing.T) {
	if err := SaveDraft("", session{Workflow: "x"}); err != nil {
		t.Fatalf("SaveDraft(\"\"): %v", err)
	}
}

func TestSaveDraftIsAtomicOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := DraftPath(dir)
	if err := SaveDraft(path, session{Workflow: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveDraft(path, session{Workflow: "second"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadDraft[session](path)
	if err != nil || !ok {
		t.Fatalf("LoadDraft: ok=%v err=%v", ok, err)
	}
	if got.Workflow != "second" {
		t.Fatalf("workflow = %q, want %q", got.Workflow, "second")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestDeleteDraftRemovesFileAndPrunesIcotDir(t *testing.T) {
	dir := t.TempDir()
	path := DraftPath(dir)
	if err := SaveDraft(path, session{Workflow: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteDraft(path); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".icot")); !os.IsNotExist(err) {
		t.Fatalf(".icot/ should have been pruned: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("example dir should survive draft deletion: %v", err)
	}
}

func TestDeleteDraftMissingIsNoError(t *testing.T) {
	if err := DeleteDraft(filepath.Join(t.TempDir(), "missing.yaml")); err != nil {
		t.Fatalf("DeleteDraft on missing: %v", err)
	}
}
