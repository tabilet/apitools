package apitools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestBuiltInCatalogSecurityAuditClassifiesEveryProvider(t *testing.T) {
	report, err := BuiltInCatalogSecurityAuditReport(CatalogSecurityAuditOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ProviderCount != 302 {
		t.Fatalf("provider count = %d, want 302", report.Summary.ProviderCount)
	}
	if countAuditDisposition(report, SecurityAuditDispositionQueuedSourceReReview) != 0 {
		t.Fatalf("queued source re-review count = %d, want 0", countAuditDisposition(report, SecurityAuditDispositionQueuedSourceReReview))
	}
	if countAuditDisposition(report, SecurityAuditDispositionPresentIncompleteReviewed) != 37 {
		t.Fatalf("present-incomplete-reviewed count = %d, want 37", countAuditDisposition(report, SecurityAuditDispositionPresentIncompleteReviewed))
	}
	for _, disposition := range []string{
		SecurityAuditDispositionCompleteUpstream,
		SecurityAuditDispositionCompleteViaOverlay,
		SecurityAuditDispositionPresentIncompleteReviewed,
		SecurityAuditDispositionIntentionallyAnonymous,
	} {
		if countAuditDisposition(report, disposition) == 0 {
			t.Fatalf("missing disposition %q in %#v", disposition, report.Summary.Dispositions)
		}
	}
	dockerRegistry := findAuditProvider(t, report, "docker-registry")
	if dockerRegistry.AuthStatus != catalog.AuthStatusPresentIncomplete {
		t.Fatalf("docker-registry auth status = %q, want %q", dockerRegistry.AuthStatus, catalog.AuthStatusPresentIncomplete)
	}
	if dockerRegistry.Disposition != SecurityAuditDispositionPresentIncompleteReviewed {
		t.Fatalf("docker-registry disposition = %q, want %q", dockerRegistry.Disposition, SecurityAuditDispositionPresentIncompleteReviewed)
	}
	if len(dockerRegistry.OverlayIDs) == 0 {
		t.Fatalf("docker-registry should retain overlay IDs")
	}
}

func TestCatalogSecurityAuditInspectsOpenAPIArtifactSecurity(t *testing.T) {
	dir := t.TempDir()
	openAPIDir := filepath.Join(dir, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join("openapi", "example.json")
	content := []byte(`{
  "openapi": "3.0.3",
  "info": {"title": "Example", "version": "1.0.0"},
  "paths": {"/widgets": {"get": {"operationId": "listWidgets"}}},
  "components": {"securitySchemes": {"BearerAuth": {"type": "http", "scheme": "bearer"}}}
}`)
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(specPath)), content, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := BuildCatalogSecurityAuditReport(CatalogSecurityAuditOptions{
		Catalog: catalog.Catalog{
			Providers: []catalog.Provider{testSecurityAuditProvider()},
		},
		SecurityClassifications: []catalog.SecurityClassification{{
			ProviderID: "example",
			SpecRefID:  "example-openapi",
			Status:     catalog.AuthStatusPresentIncomplete,
			SourceRefs: []string{"https://example.com/docs"},
			SourceNote: "test classification",
		}},
		Artifacts: []catalog.CatalogSpecArtifact{{
			ProviderID: "example",
			SpecRefID:  "example-openapi",
			ArtifactID: "example-openapi",
			Kind:       "openapi",
			Path:       specPath,
		}},
		CacheDir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(report.Providers))
	}
	artifacts := report.Providers[0].ArtifactSecurity
	if len(artifacts) != 1 {
		t.Fatalf("artifact rows = %d, want 1", len(artifacts))
	}
	row := artifacts[0]
	if row.Status != SecurityAuditArtifactMissingSecurityRequirements {
		t.Fatalf("artifact status = %q, want %q", row.Status, SecurityAuditArtifactMissingSecurityRequirements)
	}
	if row.SecuritySchemeCount != 1 || row.OperationCount != 1 || row.OperationSecurityCount != 0 {
		t.Fatalf("artifact counts = schemes:%d ops:%d secured:%d, want 1/1/0", row.SecuritySchemeCount, row.OperationCount, row.OperationSecurityCount)
	}
	if !row.SecurityDefinitionsInspected || !row.SecurityRequirementsInspected {
		t.Fatalf("artifact security inspection flags not set: %#v", row)
	}
}

func TestCatalogSecurityAuditDetectsMissingOpenAPISecurityMetadata(t *testing.T) {
	dir := t.TempDir()
	openAPIDir := filepath.Join(dir, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join("openapi", "public.json")
	content := []byte(`{
  "openapi": "3.0.3",
  "info": {"title": "Example", "version": "1.0.0"},
  "paths": {"/status": {"get": {"operationId": "getStatus"}}}
}`)
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(specPath)), content, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := BuildCatalogSecurityAuditReport(CatalogSecurityAuditOptions{
		Catalog: catalog.Catalog{
			Providers: []catalog.Provider{testSecurityAuditProvider()},
		},
		SecurityClassifications: []catalog.SecurityClassification{{
			ProviderID: "example",
			SpecRefID:  "example-openapi",
			Status:     catalog.AuthStatusPresentIncomplete,
			SourceRefs: []string{"https://example.com/docs"},
			SourceNote: "test classification",
		}},
		Artifacts: []catalog.CatalogSpecArtifact{{
			ProviderID: "example",
			SpecRefID:  "example-openapi",
			Kind:       "openapi",
			Path:       specPath,
		}},
		CacheDir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	row := report.Providers[0].ArtifactSecurity[0]
	if row.Status != SecurityAuditArtifactMissingSecurityMetadata {
		t.Fatalf("artifact status = %q, want %q", row.Status, SecurityAuditArtifactMissingSecurityMetadata)
	}
}

func countAuditDisposition(report CatalogSecurityAuditReport, disposition string) int {
	for _, bucket := range report.Summary.Dispositions {
		if bucket.Name == disposition {
			return bucket.Count
		}
	}
	return 0
}

func findAuditProvider(t *testing.T, report CatalogSecurityAuditReport, providerID string) CatalogSecurityAuditRow {
	t.Helper()
	for _, provider := range report.Providers {
		if provider.ProviderID == providerID {
			return provider
		}
	}
	t.Fatalf("missing provider %q", providerID)
	return CatalogSecurityAuditRow{}
}

func testSecurityAuditProvider() catalog.Provider {
	return catalog.Provider{
		ID:                              "example",
		DisplayName:                     "Example",
		Category:                        "test",
		WorkflowRelevance:               "test",
		SourceHints:                     []string{"test"},
		ReviewState:                     catalog.ProviderReviewedCatalogEntry,
		CandidateID:                     "example",
		OfficialOpenAPIAvailability:     catalog.SpecAvailabilityKnown,
		OfficialMachineSpecAvailability: catalog.SpecAvailabilityUnknown,
		UserOpenAPINeed:                 catalog.UserOpenAPINeedNotExpected,
		SpecReferences: []catalog.SpecReference{{
			ID:              "example-openapi",
			Kind:            catalog.SpecKindOpenAPI,
			URL:             "https://example.com/openapi.json",
			SourceAuthority: catalog.SourceAuthorityOfficialProvider,
			SourceNote:      "test source",
			VerifiedAt:      "2026-05-21",
		}},
	}
}
