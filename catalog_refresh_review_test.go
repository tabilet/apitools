package apitools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/apitools/catalog"
)

func TestCatalogSpecRefreshReviewReportsMissingRegistration(t *testing.T) {
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)}, CatalogSpecRefreshReviewOptions{
		CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshMissingRegistration {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshMissingRegistration)
	}
	if result.Exists {
		t.Fatalf("exists = true, want false")
	}
	if len(result.ManualFollowUps) == 0 {
		t.Fatalf("missing manual follow-ups")
	}
}

func TestCatalogSpecRefreshReviewReportsMissingRegisteredFile(t *testing.T) {
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshMissingFile {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshMissingFile)
	}
	if result.SavedPath == "" {
		t.Fatalf("saved path is empty")
	}
}

func TestCatalogSpecRefreshReviewValidOpenAPI(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshValidOpenAPI)
	}
	if result.Bytes == 0 || result.SHA256 == "" || !result.Exists {
		t.Fatalf("missing file evidence: %#v", result)
	}
}

func TestCatalogSpecRefreshReviewReportsParseableInvalidOpenAPI(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshParseableOpenAPIInvalid {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshParseableOpenAPIInvalid)
	}
	if result.ValidationError == "" {
		t.Fatalf("validation error is empty")
	}
	if result.Metadata.Title != "Demo" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if !hasRefreshReviewFollowUp(result, "strict validation errors") {
		t.Fatalf("missing strict validation follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewValidStructuredArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind catalog.SpecKind
		body string
	}{
		{name: "google", kind: catalog.SpecKindGoogleDiscovery, body: `{"title":"Google","version":"v1","resources":{}}`},
		{name: "index", kind: catalog.SpecKindOpenAPIIndex, body: `{"title":"Index","apis":{"demo":{"openapi":"demo.yaml"}}}`},
		{name: "smithy", kind: catalog.SpecKindSmithyJSON, body: `{"smithy":"2.0","shapes":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ref := refreshReviewTestRef(tc.name, tc.kind)
			ref.RegisteredArtifactPath = "openapi/" + tc.name + ".json"
			if tc.kind == catalog.SpecKindGoogleDiscovery {
				ref.RegisteredArtifactPath = "google-discovery/" + tc.name + ".json"
			}
			writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, tc.body)

			report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
			if err != nil {
				t.Fatal(err)
			}
			result := singleRefreshReviewResult(t, report)
			if result.ValidationStatus != CatalogRefreshValidStructured {
				t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshValidStructured)
			}
		})
	}
}

func TestCatalogSpecRefreshReviewNonOpenAPIMachineSpec(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("stone", catalog.SpecKindDropboxStone)
	ref.RegisteredArtifactPath = "openapi/stone.tar.gz"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, "stone module content")

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshSkippedValidation {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshSkippedValidation)
	}
}

func TestCatalogSpecRefreshReviewDiscoversUnregisteredSavedFile(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	writeRefreshReviewArtifact(t, dir, "openapi/demo.json", `{"openapi":"3.1.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RegisteredArtifactPath != "" {
		t.Fatalf("registered artifact path = %q, want empty", result.RegisteredArtifactPath)
	}
	if result.ValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshValidOpenAPI)
	}
	if result.SavedPath == "" {
		t.Fatalf("saved path is empty")
	}
}

func TestCatalogSpecRefreshReviewDiscoversAlternateSavedFilename(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("calendar-discovery-v3", catalog.SpecKindGoogleDiscovery)
	ref.ProviderID = "google-calendar"
	ref.ProviderName = "Google Calendar"
	writeRefreshReviewArtifact(t, dir, "google-discovery/google-calendar-discovery-v3.json", `{"title":"Calendar","version":"v3","resources":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshValidStructured)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.SavedPath), "google-discovery/google-calendar-discovery-v3.json") {
		t.Fatalf("saved path = %q", result.SavedPath)
	}
	if !hasRefreshReviewFollowUp(result, "Register the saved artifact path") {
		t.Fatalf("missing registration follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewReportsStaleVerificationDate(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.VerifiedAt = "2024-01-01"
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)
	asOf, err := time.Parse("2006-01-02", "2026-05-19")
	if err != nil {
		t.Fatal(err)
	}

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir:              dir,
		AsOf:                  asOf,
		StaleVerificationDays: 365,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if !result.VerificationStale {
		t.Fatalf("verification stale = false, want true")
	}
	if !hasRefreshReviewFollowUp(result, "stale verification date 2024-01-01") {
		t.Fatalf("missing stale follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewRejectsSymlinkArtifact(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"openapi":"3.0.0","info":{"title":"Outside","version":"1.0.0"},"paths":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "openapi", "demo.json")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshInvalid)
	}
	if !strings.Contains(result.ValidationError, "symlink") {
		t.Fatalf("validation error = %q, want symlink", result.ValidationError)
	}
}

func TestCatalogSpecRefreshReviewRejectsOversizedArtifact(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir: dir,
		MaxBytes: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.ValidationStatus != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", result.ValidationStatus, CatalogRefreshInvalid)
	}
	if !strings.Contains(result.ValidationError, "over limit 8") {
		t.Fatalf("validation error = %q, want size limit", result.ValidationError)
	}
	if result.Bytes <= 8 {
		t.Fatalf("bytes = %d, want original size over limit", result.Bytes)
	}
}

func TestCatalogSpecRefreshReviewDeterministicOrdering(t *testing.T) {
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{
		refreshReviewTestRef("zeta", catalog.SpecKindOpenAPI),
		refreshReviewTestRef("alpha", catalog.SpecKindOpenAPI),
	}, CatalogSpecRefreshReviewOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := report.Results[0].ProviderID+"/"+report.Results[0].SpecRefID, "alpha/alpha"; got != want {
		t.Fatalf("first result = %q, want %q", got, want)
	}
	if got, want := report.Results[1].ProviderID+"/"+report.Results[1].SpecRefID, "zeta/zeta"; got != want {
		t.Fatalf("second result = %q, want %q", got, want)
	}
}

func refreshReviewTestRef(id string, kind catalog.SpecKind) catalog.RefreshableSpecReference {
	return catalog.RefreshableSpecReference{
		ProviderID:      id,
		ProviderName:    id,
		SpecRefID:       id,
		Kind:            kind,
		URL:             "https://example.com/" + id + ".json",
		SourceAuthority: catalog.SourceAuthorityOfficialProvider,
		VerifiedAt:      "2026-05-19",
	}
}

func writeRefreshReviewArtifact(t *testing.T, cacheDir, artifactPath, content string) {
	t.Helper()
	fullPath := filepath.Join(cacheDir, filepath.FromSlash(artifactPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func singleRefreshReviewResult(t *testing.T, report CatalogSpecRefreshReviewReport) CatalogSpecRefreshReviewResult {
	t.Helper()
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("result len = %d, want %d", got, want)
	}
	return report.Results[0]
}

func hasRefreshReviewFollowUp(result CatalogSpecRefreshReviewResult, text string) bool {
	for _, followUp := range result.ManualFollowUps {
		if strings.Contains(followUp, text) {
			return true
		}
	}
	return false
}
