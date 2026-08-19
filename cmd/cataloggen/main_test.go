package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
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

func TestBuildOutputsRejectsConflictingScopedDispositions(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "catalog", "data", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := catalog.ParseCatalogBundle(content)
	if err != nil {
		t.Fatal(err)
	}
	conflict := bundle.SecurityOverlays[0]
	conflict.ID += "-conflict"
	if conflict.Status == catalog.AuthStatusComplete {
		conflict.Status = catalog.AuthStatusAbsent
	} else {
		conflict.Status = catalog.AuthStatusComplete
	}
	bundle.SecurityOverlays = append(bundle.SecurityOverlays, conflict)
	sort.Slice(bundle.SecurityOverlays, func(i, j int) bool {
		left, right := bundle.SecurityOverlays[i], bundle.SecurityOverlays[j]
		if left.ProviderID == right.ProviderID {
			return left.ID < right.ID
		}
		return left.ProviderID < right.ProviderID
	})
	content, err = json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	_, err = buildOutputs(content)
	if err == nil || !strings.Contains(err.Error(), "conflicting security dispositions") {
		t.Fatalf("conflicting-disposition error = %v", err)
	}
}

func TestBuildOutputsRejectsInvalidCatalogReferences(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*catalog.CatalogBundle)
		want   string
	}{
		{
			name: "invalid id",
			mutate: func(bundle *catalog.CatalogBundle) {
				bundle.Candidates[0].ID = "INVALID ID"
			},
			want: "invalid id",
		},
		{
			name: "missing candidate reference",
			mutate: func(bundle *catalog.CatalogBundle) {
				bundle.Providers[0].CandidateID = "missing-candidate"
			},
			want: "unknown candidate",
		},
		{
			name: "missing overlay scope",
			mutate: func(bundle *catalog.CatalogBundle) {
				bundle.SecurityOverlays[0].SpecRefID = "missing-spec"
			},
			want: "unknown spec ref",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("..", "..", "catalog", "data", "catalog.json"))
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := catalog.ParseCatalogBundle(content)
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(&bundle)
			content, err = json.Marshal(bundle)
			if err != nil {
				t.Fatal(err)
			}
			_, err = buildOutputs(content)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("generation error = %v, want %q", err, test.want)
			}
		})
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
