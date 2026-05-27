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
	if !strings.HasSuffix(filepath.ToSlash(report.Artifacts[0].TargetPath), "demo/openapi/demo-openapi.json") {
		t.Fatalf("materialized target path = %q, want source-aligned openapi path", report.Artifacts[0].TargetPath)
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

func TestMaterializeProviderUsesSourceAlignedDirs(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	for rel, content := range map[string]string{
		filepath.Join("google-discovery", "gmail.json"):    `{"title":"Gmail","version":"v1","resources":{}}`,
		filepath.Join("aws-smithy", "aws-s3-smithy.json"):  `{"smithy":"2.0","shapes":{}}`,
		filepath.Join("asyncapi", "events.yaml"):           `{"asyncapi":"3.0.0","info":{"title":"Events","version":"1.0.0"},"operations":{}}`,
		filepath.Join("openapi", "docs-overlay.json"):      `{"openapi":"3.0.3","info":{"title":"Docs","version":"1"},"paths":{}}`,
		filepath.Join("artifacts", "dropbox-stone.tar.gz"): `stone module`,
	} {
		path := filepath.Join(cacheDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	report, err := MaterializeProvider(context.Background(), MaterializeOptions{
		Catalog:     Catalog{Providers: []Provider{qualityProvider("demo")}},
		ProviderKey: "demo",
		TargetDir:   filepath.Join(dir, "out"),
		CacheDir:    cacheDir,
		Artifacts: []CatalogSpecArtifact{
			{ProviderID: "demo", SpecRefID: "gmail", ArtifactID: "gmail", Kind: "google-discovery", Path: "google-discovery/gmail.json"},
			{ProviderID: "demo", SpecRefID: "aws-s3", ArtifactID: "aws-s3", Kind: "smithy-json", Path: "aws-smithy/aws-s3-smithy.json"},
			{ProviderID: "demo", SpecRefID: "events", ArtifactID: "events", Kind: "asyncapi", Path: "asyncapi/events.yaml"},
			{ProviderID: "demo", SpecRefID: "docs", ArtifactID: "docs", Kind: "advisory-overlay", Path: "openapi/docs-overlay.json"},
			{ProviderID: "demo", SpecRefID: "stone", ArtifactID: "stone", Kind: "dropbox-stone", Path: "artifacts/dropbox-stone.tar.gz"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	artifactsByID := map[string]MaterializedArtifact{}
	for _, artifact := range report.Artifacts {
		targets[filepath.ToSlash(artifact.TargetPath)] = true
		artifactsByID[artifact.ArtifactID] = artifact
	}
	for _, wantSuffix := range []string{
		"demo/google-discovery/gmail.json",
		"demo/aws-smithy/aws-s3.json",
		"demo/asyncapi/events.yaml",
		"demo/openapi/docs.json",
		"demo/artifacts/stone.tar.gz",
	} {
		var found bool
		for target := range targets {
			if strings.HasSuffix(target, wantSuffix) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing materialized target suffix %q in %#v", wantSuffix, targets)
		}
	}
	for id, want := range map[string]struct {
		protocol      SpecProtocol
		uwsSourceType string
	}{
		"gmail":  {protocol: SpecProtocolGoogleDiscovery, uwsSourceType: "google-discovery"},
		"aws-s3": {protocol: SpecProtocolSmithy, uwsSourceType: "aws-smithy"},
		"events": {protocol: SpecProtocolAsyncAPI, uwsSourceType: "asyncapi"},
		"docs":   {protocol: SpecProtocolOpenAPI, uwsSourceType: "openapi"},
		"stone":  {protocol: SpecProtocolDropboxStone},
	} {
		artifact, ok := artifactsByID[id]
		if !ok {
			t.Fatalf("missing materialized artifact %q", id)
		}
		if artifact.Protocol != want.protocol {
			t.Fatalf("%s protocol = %q, want %q", id, artifact.Protocol, want.protocol)
		}
		if artifact.UWSSourceType != want.uwsSourceType {
			t.Fatalf("%s UWS source type = %q, want %q", id, artifact.UWSSourceType, want.uwsSourceType)
		}
	}
}
