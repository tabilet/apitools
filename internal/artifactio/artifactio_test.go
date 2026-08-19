package artifactio

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileConfinesPathsAndVerifiesIntegrity(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.json"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	good := digestBytes([]byte("source"))
	file, err := ReadFile(root, "source.json", ReadOptions{SHA256: good, Bytes: 6})
	if err != nil {
		t.Fatal(err)
	}
	if string(file.Data) != "source" || file.SHA256 != good {
		t.Fatalf("file = %#v", file)
	}
	for _, path := range []string{"../source.json", filepath.Join(root, "source.json"), "."} {
		if _, err := ReadFile(root, path, ReadOptions{}); err == nil || !strings.Contains(err.Error(), "local relative") {
			t.Errorf("path %q: expected confinement error, got %v", path, err)
		}
	}
	if _, err := ReadFile(root, "source.json", ReadOptions{SHA256: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestReadAndWriteRejectSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadFile(root, "linked", ReadOptions{}); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink read rejection, got %v", err)
	}
	if _, err := WriteFile(root, "linked", []byte("replacement"), WriteOptions{Force: true}); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink write rejection, got %v", err)
	}
	content, err := os.ReadFile(outside)
	if err != nil || string(content) != "outside" {
		t.Fatalf("outside content = %q, err %v", content, err)
	}
}

func TestArtifactRootsAndParentsRejectSymlinks(t *testing.T) {
	realRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(realRoot, "source"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	container := t.TempDir()
	linkedRoot := filepath.Join(container, "linked-root")
	if err := os.Symlink(realRoot, linkedRoot); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadFile(linkedRoot, "source", ReadOptions{}); err == nil || !strings.Contains(err.Error(), "directory component") {
		t.Fatalf("expected symlink root rejection, got %v", err)
	}

	root := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "source"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(root, "linked-parent")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(root, "linked-parent/source", ReadOptions{}); err == nil || !strings.Contains(err.Error(), "directory component") {
		t.Fatalf("expected symlink parent rejection, got %v", err)
	}
	if _, err := BeginDir(filepath.Join(root, "linked-parent", "output"), true); err == nil || !strings.Contains(err.Error(), "directory component") {
		t.Fatalf("expected transaction parent rejection, got %v", err)
	}
}

func TestWriteFileReusesIdenticalAndRequiresForceForDifference(t *testing.T) {
	root := t.TempDir()
	first, err := WriteFile(root, "nested/artifact.json", []byte("one"), WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused {
		t.Fatal("new write reported reuse")
	}
	second, err := WriteFile(root, "nested/artifact.json", []byte("one"), WriteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused {
		t.Fatal("identical write was not reused")
	}
	if _, err := WriteFile(root, "nested/artifact.json", []byte("two"), WriteOptions{}); !errors.Is(err, ErrCollision) {
		t.Fatalf("expected collision, got %v", err)
	}
	if _, err := WriteFile(root, "nested/artifact.json", []byte("two"), WriteOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "nested", "artifact.json"))
	if err != nil || string(content) != "two" {
		t.Fatalf("content = %q, err %v", content, err)
	}
}

func TestDirTransactionCollisionForceAndRollback(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "artifacts")
	tx, err := BeginDir(target, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Stage("one.txt", []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if reused, err := tx.Commit(); err != nil || reused {
		t.Fatalf("first commit reused=%v err=%v", reused, err)
	}

	tx, err = BeginDir(target, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Stage("one.txt", []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if reused, err := tx.Commit(); err != nil || !reused {
		t.Fatalf("identical commit reused=%v err=%v", reused, err)
	}

	tx, err = BeginDir(target, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Stage("two.txt", []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(); !errors.Is(err, ErrCollision) {
		t.Fatalf("expected directory collision, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "one.txt")); err != nil {
		t.Fatalf("collision mutated old tree: %v", err)
	}

	tx, err = BeginDir(target, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Stage("two.txt", []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "one.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("forced replacement retained obsolete file: %v", err)
	}

	tx, err = BeginDir(target, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Stage("three.txt", []byte("three"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "two.txt")); err != nil {
		t.Fatalf("rollback mutated target: %v", err)
	}
}
