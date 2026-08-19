package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOutputsIsDeterministic(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "catalog", "data", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := buildOutputs(content)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildOutputs(first.catalog)
	if err != nil {
		t.Fatal(err)
	}
	for label, values := range map[string][][]byte{
		"catalog":  {first.catalog, second.catalog},
		"manifest": {first.manifest, second.manifest},
		"go":       {first.goSource, second.goSource},
	} {
		if !bytes.Equal(values[0], values[1]) {
			t.Fatalf("%s generation is not deterministic", label)
		}
	}
}

func TestBuildOutputsRejectsInvalidBundle(t *testing.T) {
	_, err := buildOutputs([]byte(`{"version":"wrong"}`))
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("invalid bundle error = %v", err)
	}
}

func TestCheckedInCatalogOutputsAreCurrent(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-check", "-root", filepath.Join("..", "..")}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "current") {
		t.Fatalf("check output = %q", out.String())
	}
}

func TestCheckRejectsStaleGeneratedOutput(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "catalog", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "catalog", "data", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "catalog.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "catalog", "catalog_gen.go"), []byte("package catalog\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err = run([]string{"-check", "-root", root}, &out)
	if err == nil || !strings.Contains(err.Error(), "is stale") {
		t.Fatalf("stale-output error = %v", err)
	}
}
