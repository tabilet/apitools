package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProvidersWithArtifactsReportsExecutableOpenAPI(t *testing.T) {
	provider := qualityProvider("demo")
	rows, err := ResolveProvidersWithOptions(ProviderResolutionOptions{
		Catalog: Catalog{Providers: []Provider{provider}},
		ProviderKeys: []string{
			"demo",
		},
		Artifacts: []CatalogSpecArtifact{{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       "openapi/demo.json",
			Metadata:   map[string]string{"validation_status": "valid-openapi"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("resolution rows = %d, want 1", len(rows))
	}
	if rows[0].Capability != ProviderArtifactCapabilityExecutableOpenAPI {
		t.Fatalf("capability = %q, want %q", rows[0].Capability, ProviderArtifactCapabilityExecutableOpenAPI)
	}
	if len(rows[0].Artifacts) != 1 || !rows[0].Artifacts[0].Materializable {
		t.Fatalf("artifact resolution = %#v, want one materializable artifact", rows[0].Artifacts)
	}
}

func TestMaterializeProviderCopiesArtifactsAndEmitsOverlay(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	sourceRel := filepath.Join("openapi", "demo.json")
	sourcePath := filepath.Join(cacheDir, sourceRel)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "out")
	report, err := MaterializeProvider(context.Background(), MaterializeOptions{
		Catalog: Catalog{
			Providers:        []Provider{qualityProvider("demo")},
			SecurityOverlays: []SecurityOverlay{qualityOverlay("demo")},
		},
		ProviderKey:             "demo",
		TargetDir:               targetDir,
		CacheDir:                cacheDir,
		IncludeSecurityOverlays: true,
		WriteManifest:           true,
		Artifacts: []CatalogSpecArtifact{{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       filepath.ToSlash(sourceRel),
			SourceURL:  "https://example.com/openapi.json",
			Metadata:   map[string]string{"validation_status": "valid-openapi"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Artifacts) != 1 {
		t.Fatalf("materialized artifacts = %d, want 1", len(report.Artifacts))
	}
	if len(report.SecurityOverlays) != 1 {
		t.Fatalf("materialized security overlays = %d, want 1", len(report.SecurityOverlays))
	}
	if report.ManifestPath == "" {
		t.Fatalf("missing manifest path")
	}
	for _, path := range []string{report.Artifacts[0].TargetPath, report.SecurityOverlays[0].TargetPath, report.ManifestPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected materialized file %s: %v", path, err)
		}
	}
	content, err := os.ReadFile(report.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"provider_id": "demo"`) || !strings.Contains(string(content), `"source_url": "https://example.com/openapi.json"`) {
		t.Fatalf("manifest missing provenance:\n%s", string(content))
	}
}

func TestExportWorkflowArtifactsWritesAggregateManifest(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	sourceRel := filepath.Join("openapi", "demo.json")
	sourcePath := filepath.Join(cacheDir, sourceRel)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ExportWorkflowArtifacts(context.Background(), ExportWorkflowArtifactsOptions{
		Catalog: Catalog{Providers: []Provider{qualityProvider("demo")}},
		ProviderKeys: []string{
			"demo",
		},
		WorkflowDir:   filepath.Join(dir, "workflow"),
		CacheDir:      cacheDir,
		WriteManifest: true,
		Artifacts: []CatalogSpecArtifact{{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       filepath.ToSlash(sourceRel),
			Metadata:   map[string]string{"validation_status": "valid-openapi"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 1 {
		t.Fatalf("export providers = %d, want 1", len(report.Providers))
	}
	if _, err := os.Stat(report.ManifestPath); err != nil {
		t.Fatalf("expected aggregate manifest: %v", err)
	}
}
