package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/internal/artifactio"
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

func TestResolveProvidersReportsOpenRPCOnlyCapability(t *testing.T) {
	provider := Provider{
		ID:                              "pet-rpc",
		DisplayName:                     "Pet RPC",
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     "pet-rpc",
		OfficialOpenAPIAvailability:     SpecAvailabilityUnavailable,
		OfficialMachineSpecAvailability: SpecAvailabilityKnown,
		UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
		SpecReferences: []SpecReference{{
			ID:              "pet-openrpc",
			Kind:            SpecKindOpenRPC,
			URL:             "https://example.com/openrpc.json",
			SourceAuthority: SourceAuthorityOfficialProvider,
			SourceNote:      "Official OpenRPC document.",
		}},
	}
	rows, err := ResolveProvidersWithOptions(ProviderResolutionOptions{
		Catalog:      Catalog{Providers: []Provider{provider}},
		ProviderKeys: []string{"pet-rpc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("resolution rows = %d, want 1", len(rows))
	}
	if rows[0].Capability != ProviderArtifactCapabilityOpenRPCOnly {
		t.Fatalf("capability = %q, want %q", rows[0].Capability, ProviderArtifactCapabilityOpenRPCOnly)
	}
	if rows[0].SpecReferences[0].UWSSourceType != "openrpc" {
		t.Fatalf("uws source type = %q, want openrpc", rows[0].SpecReferences[0].UWSSourceType)
	}
}

func TestResolveProvidersReportsGraphQLOnlyCapability(t *testing.T) {
	provider := Provider{
		ID:                              "issue-graphql",
		DisplayName:                     "Issue GraphQL",
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     "issue-graphql",
		OfficialOpenAPIAvailability:     SpecAvailabilityUnavailable,
		OfficialMachineSpecAvailability: SpecAvailabilityKnown,
		UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
		SpecReferences: []SpecReference{{
			ID:              "issue-graphql-schema",
			Kind:            SpecKindGraphQL,
			URL:             "https://example.com/schema.graphql",
			SourceAuthority: SourceAuthorityOfficialProvider,
			SourceNote:      "Official GraphQL SDL schema.",
		}},
	}
	rows, err := ResolveProvidersWithOptions(ProviderResolutionOptions{
		Catalog:      Catalog{Providers: []Provider{provider}},
		ProviderKeys: []string{"issue-graphql"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("resolution rows = %d, want 1", len(rows))
	}
	if rows[0].Capability != ProviderArtifactCapabilityGraphQLOnly {
		t.Fatalf("capability = %q, want %q", rows[0].Capability, ProviderArtifactCapabilityGraphQLOnly)
	}
	if rows[0].SpecReferences[0].UWSSourceType != "graphql" {
		t.Fatalf("uws source type = %q, want graphql", rows[0].SpecReferences[0].UWSSourceType)
	}
}

func TestResolveProvidersReportsGRPCProtobufOnlyCapability(t *testing.T) {
	provider := Provider{
		ID:                              "issue-grpc",
		DisplayName:                     "Issue gRPC",
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     "issue-grpc",
		OfficialOpenAPIAvailability:     SpecAvailabilityUnavailable,
		OfficialMachineSpecAvailability: SpecAvailabilityKnown,
		UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
		SpecReferences: []SpecReference{{
			ID:              "issue-proto",
			Kind:            SpecKindGRPCProtobuf,
			URL:             "https://example.com/issues.proto",
			SourceAuthority: SourceAuthorityOfficialProvider,
			SourceNote:      "Official gRPC protobuf source.",
		}},
	}
	rows, err := ResolveProvidersWithOptions(ProviderResolutionOptions{
		Catalog:      Catalog{Providers: []Provider{provider}},
		ProviderKeys: []string{"issue-grpc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("resolution rows = %d, want 1", len(rows))
	}
	if rows[0].Capability != ProviderArtifactCapabilityGRPCProtobufOnly {
		t.Fatalf("capability = %q, want %q", rows[0].Capability, ProviderArtifactCapabilityGRPCProtobufOnly)
	}
	if rows[0].SpecReferences[0].UWSSourceType != "grpc-protobuf" {
		t.Fatalf("uws source type = %q, want grpc-protobuf", rows[0].SpecReferences[0].UWSSourceType)
	}
}

func TestResolveProvidersReportsODataOnlyCapability(t *testing.T) {
	provider := Provider{
		ID:                              "products-odata",
		DisplayName:                     "Products OData",
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     "products-odata",
		OfficialOpenAPIAvailability:     SpecAvailabilityUnavailable,
		OfficialMachineSpecAvailability: SpecAvailabilityKnown,
		UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
		SpecReferences: []SpecReference{{
			ID:              "products-metadata",
			Kind:            SpecKindOData,
			URL:             "https://example.com/$metadata",
			SourceAuthority: SourceAuthorityOfficialProvider,
			SourceNote:      "Official OData CSDL metadata.",
		}},
	}
	rows, err := ResolveProvidersWithOptions(ProviderResolutionOptions{
		Catalog:      Catalog{Providers: []Provider{provider}},
		ProviderKeys: []string{"products-odata"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("resolution rows = %d, want 1", len(rows))
	}
	if rows[0].Capability != ProviderArtifactCapabilityODataOnly {
		t.Fatalf("capability = %q, want %q", rows[0].Capability, ProviderArtifactCapabilityODataOnly)
	}
	if rows[0].SpecReferences[0].UWSSourceType != "odata" {
		t.Fatalf("uws source type = %q, want odata", rows[0].SpecReferences[0].UWSSourceType)
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
		Artifacts: []CatalogSpecArtifact{withMaterializeIntegrity(t, cacheDir, CatalogSpecArtifact{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       filepath.ToSlash(sourceRel),
			SourceURL:  "https://example.com/openapi.json",
			Metadata:   map[string]string{"validation_status": "valid-openapi"},
		})},
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
		Artifacts: []CatalogSpecArtifact{withMaterializeIntegrity(t, cacheDir, CatalogSpecArtifact{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       filepath.ToSlash(sourceRel),
			Metadata:   map[string]string{"validation_status": "valid-openapi"},
		})},
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
		filepath.Join("openrpc", "pet-rpc.json"):           `{"openrpc":"1.3.2","info":{"title":"Pet RPC","version":"1.0.0"},"methods":[]}`,
		filepath.Join("graphql", "issues.graphql"):         `type Query { issue(id: ID!): Issue }`,
		filepath.Join("grpc-protobuf", "issues.proto"):     `syntax = "proto3"; service IssueService { rpc GetIssue(GetIssueRequest) returns (Issue); } message GetIssueRequest { string id = 1; }`,
		filepath.Join("odata", "metadata.xml"):             `<Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm"><EntityContainer Name="Container"><EntitySet Name="Products" EntityType="Demo.Product"/></EntityContainer></Schema>`,
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

	artifacts := []CatalogSpecArtifact{
		{ProviderID: "demo", SpecRefID: "gmail", ArtifactID: "gmail", Kind: "google-discovery", Path: "google-discovery/gmail.json"},
		{ProviderID: "demo", SpecRefID: "aws-s3", ArtifactID: "aws-s3", Kind: "smithy-json", Path: "aws-smithy/aws-s3-smithy.json"},
		{ProviderID: "demo", SpecRefID: "events", ArtifactID: "events", Kind: "asyncapi", Path: "asyncapi/events.yaml"},
		{ProviderID: "demo", SpecRefID: "pet-rpc", ArtifactID: "pet-rpc", Kind: "openrpc", Path: "openrpc/pet-rpc.json"},
		{ProviderID: "demo", SpecRefID: "issues", ArtifactID: "issues", Kind: "graphql", Path: "graphql/issues.graphql"},
		{ProviderID: "demo", SpecRefID: "issues-grpc", ArtifactID: "issues-grpc", Kind: "grpc-protobuf", Path: "grpc-protobuf/issues.proto"},
		{ProviderID: "demo", SpecRefID: "metadata", ArtifactID: "metadata", Kind: "odata", Path: "odata/metadata.xml"},
		{ProviderID: "demo", SpecRefID: "docs", ArtifactID: "docs", Kind: "advisory-overlay", Path: "openapi/docs-overlay.json"},
		{ProviderID: "demo", SpecRefID: "stone", ArtifactID: "stone", Kind: "dropbox-stone", Path: "artifacts/dropbox-stone.tar.gz"},
	}
	for index := range artifacts {
		artifacts[index] = withMaterializeIntegrity(t, cacheDir, artifacts[index])
	}
	report, err := MaterializeProvider(context.Background(), MaterializeOptions{
		Catalog:     Catalog{Providers: []Provider{qualityProvider("demo")}},
		ProviderKey: "demo",
		TargetDir:   filepath.Join(dir, "out"),
		CacheDir:    cacheDir,
		Artifacts:   artifacts,
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
		"demo/openrpc/pet-rpc.json",
		"demo/graphql/issues.graphql",
		"demo/grpc-protobuf/issues-grpc.proto",
		"demo/odata/metadata.xml",
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
		"gmail":       {protocol: SpecProtocolGoogleDiscovery, uwsSourceType: "google-discovery"},
		"aws-s3":      {protocol: SpecProtocolSmithy, uwsSourceType: "aws-smithy"},
		"events":      {protocol: SpecProtocolAsyncAPI, uwsSourceType: "asyncapi"},
		"pet-rpc":     {protocol: SpecProtocolOpenRPC, uwsSourceType: "openrpc"},
		"issues":      {protocol: SpecProtocolGraphQL, uwsSourceType: "graphql"},
		"issues-grpc": {protocol: SpecProtocolGRPCProtobuf, uwsSourceType: "grpc-protobuf"},
		"metadata":    {protocol: SpecProtocolOData, uwsSourceType: "odata"},
		"docs":        {protocol: SpecProtocolOpenAPI, uwsSourceType: "openapi"},
		"stone":       {protocol: SpecProtocolDropboxStone},
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

func TestMaterializeProviderVerifiesDigestAndRollsBack(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "source.json"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(dir, "out")
	providerDir := filepath.Join(targetDir, "demo")
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(providerDir, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := MaterializeProvider(context.Background(), MaterializeOptions{
		Catalog:     Catalog{Providers: []Provider{qualityProvider("demo")}},
		ProviderKey: "demo",
		TargetDir:   targetDir,
		CacheDir:    cacheDir,
		Force:       true,
		Artifacts: []CatalogSpecArtifact{{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			ArtifactID: "demo-openapi",
			Kind:       "openapi",
			Path:       "source.json",
			SHA256:     strings.Repeat("0", 64),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
	content, readErr := os.ReadFile(sentinel)
	if readErr != nil || string(content) != "keep" {
		t.Fatalf("rollback changed destination: content=%q err=%v", content, readErr)
	}
}

func TestMaterializeProviderRejectsSymlinkSource(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside.json")
	content := []byte("outside")
	if err := os.WriteFile(outside, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(cacheDir, "source.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	digest := sha256.Sum256(content)
	_, err := MaterializeProvider(context.Background(), MaterializeOptions{
		Catalog:     Catalog{Providers: []Provider{qualityProvider("demo")}},
		ProviderKey: "demo",
		TargetDir:   filepath.Join(dir, "out"),
		CacheDir:    cacheDir,
		Artifacts: []CatalogSpecArtifact{{
			ProviderID: "demo", ArtifactID: "demo", Kind: "openapi", Path: "source.json", SHA256: hex.EncodeToString(digest[:]),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "out", "demo")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed materialization published output: %v", statErr)
	}
}

func TestMaterializeProviderCollisionRequiresForce(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	path := filepath.Join(cacheDir, "source.json")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := withMaterializeIntegrity(t, cacheDir, CatalogSpecArtifact{ProviderID: "demo", ArtifactID: "demo", Kind: "openapi", Path: "source.json"})
	opts := MaterializeOptions{Catalog: Catalog{Providers: []Provider{qualityProvider("demo")}}, ProviderKey: "demo", TargetDir: filepath.Join(dir, "out"), CacheDir: cacheDir, Artifacts: []CatalogSpecArtifact{artifact}}
	if _, err := MaterializeProvider(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts.Artifacts[0] = withMaterializeIntegrity(t, cacheDir, opts.Artifacts[0])
	if _, err := MaterializeProvider(context.Background(), opts); !errors.Is(err, artifactio.ErrCollision) {
		t.Fatalf("expected collision, got %v", err)
	}
	target := filepath.Join(dir, "out", "demo", "openapi", "demo.json")
	if content, err := os.ReadFile(target); err != nil || string(content) != "one" {
		t.Fatalf("collision changed destination: content=%q err=%v", content, err)
	}
	opts.Force = true
	if _, err := MaterializeProvider(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != "two" {
		t.Fatalf("forced transaction did not replace destination: content=%q err=%v", content, err)
	}
}

func TestExportWorkflowArtifactsConfinesRootAndRollsBackAllProviders(t *testing.T) {
	dir := t.TempDir()
	for _, artifactDir := range []string{"../outside", filepath.Join(dir, "absolute"), "."} {
		_, err := ExportWorkflowArtifacts(context.Background(), ExportWorkflowArtifactsOptions{
			Catalog:      Catalog{Providers: []Provider{qualityProvider("demo")}},
			ProviderKeys: []string{"demo"},
			WorkflowDir:  filepath.Join(dir, "workflow"),
			ArtifactDir:  artifactDir,
		})
		if err == nil || !strings.Contains(err.Error(), "under the workflow") {
			t.Errorf("artifact dir %q: expected confinement error, got %v", artifactDir, err)
		}
	}

	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "one.json"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	one := withMaterializeIntegrity(t, cacheDir, CatalogSpecArtifact{ProviderID: "one", ArtifactID: "one", Kind: "openapi", Path: "one.json"})
	two := CatalogSpecArtifact{ProviderID: "two", ArtifactID: "two", Kind: "openapi", Path: "missing.json", SHA256: strings.Repeat("0", 64)}
	workflowDir := filepath.Join(dir, "transaction-workflow")
	_, err := ExportWorkflowArtifacts(context.Background(), ExportWorkflowArtifactsOptions{
		Catalog:      Catalog{Providers: []Provider{qualityProvider("one"), qualityProvider("two")}},
		ProviderKeys: []string{"one", "two"},
		WorkflowDir:  workflowDir,
		CacheDir:     cacheDir,
		Artifacts:    []CatalogSpecArtifact{one, two},
	})
	if err == nil {
		t.Fatal("expected second-provider failure")
	}
	if _, statErr := os.Stat(filepath.Join(workflowDir, "api-artifacts")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed export published a partial tree: %v", statErr)
	}
}

func withMaterializeIntegrity(t *testing.T, cacheDir string, artifact CatalogSpecArtifact) CatalogSpecArtifact {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(cacheDir, filepath.FromSlash(artifact.Path)))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	artifact.SHA256 = hex.EncodeToString(digest[:])
	artifact.Bytes = int64(len(content))
	return artifact
}
