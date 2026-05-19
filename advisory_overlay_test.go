package apitools

import (
	"path/filepath"
	"testing"
)

func TestTrackedAdvisoryOverlayArtifactsLoad(t *testing.T) {
	paths, err := filepath.Glob("catalog-openapi-cache/advisory-overlays/*-overlay.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no advisory overlay artifacts found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			index, err := LoadOperationIndex(path)
			if err != nil {
				t.Fatalf("LoadOperationIndex(%q) error = %v", path, err)
			}
			if len(index.OperationIDs) == 0 {
				t.Fatalf("%s has no indexed operations", path)
			}
		})
	}
}
