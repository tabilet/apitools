package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBuiltInCatalogManifestMatchesReviewedBundle(t *testing.T) {
	content, err := os.ReadFile("data/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseCatalogBundle(content)
	if err != nil {
		t.Fatal(err)
	}
	manifest := BuiltInCatalogManifest()
	digest := sha256.Sum256(content)
	specCount := 0
	for _, provider := range bundle.Providers {
		specCount += len(provider.SpecReferences)
	}
	if manifest.Version != generatedCatalogManifestVersion || manifest.CatalogVersion != bundle.Version || manifest.CatalogSHA256 != hex.EncodeToString(digest[:]) || manifest.CatalogBytes != len(content) || manifest.CandidateCount != len(bundle.Candidates) || manifest.ProviderCount != len(bundle.Providers) || manifest.SpecReferenceCount != specCount || manifest.SecurityClassificationCount != len(bundle.SecurityClassifications) || manifest.SecurityOverlayCount != len(bundle.SecurityOverlays) {
		t.Fatalf("manifest = %#v, does not match reviewed bundle", manifest)
	}
}

func TestReviewedCatalogBundleMatchesEmbeddedBuiltIns(t *testing.T) {
	content, err := os.ReadFile("data/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseCatalogBundle(content)
	if err != nil {
		t.Fatal(err)
	}
	assertCatalogJSONEqual(t, "candidates", bundle.Candidates, BuiltInCandidates())
	assertCatalogJSONEqual(t, "providers", bundle.Providers, BuiltInProviders())
	assertCatalogJSONEqual(t, "classifications", bundle.SecurityClassifications, BuiltInSecurityClassifications())
	assertCatalogJSONEqual(t, "overlays", bundle.SecurityOverlays, BuiltInSecurityOverlays())
}

func assertCatalogJSONEqual(t *testing.T, label string, got, want any) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("reviewed %s differ from built-in %s", label, label)
	}
}

func TestParseCatalogBundleRejectsUnknownFields(t *testing.T) {
	_, err := ParseCatalogBundle([]byte(`{"version":"apitools.catalog.v1","unknown":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown-field error = %v", err)
	}
}

func TestValidateCatalogBundleRejectsUnknownCandidate(t *testing.T) {
	bundle := reviewedCatalogBundleForTest(t)
	bundle.Providers[0].CandidateID = "missing-candidate"
	if err := ValidateCatalogBundle(bundle); err == nil || !strings.Contains(err.Error(), "unknown candidate") {
		t.Fatalf("unknown-candidate error = %v", err)
	}
}

func TestValidateSecurityClassificationsRejectsDuplicateScope(t *testing.T) {
	bundle := reviewedCatalogBundleForTest(t)
	classification := bundle.SecurityClassifications[0]
	classifications := append(bundle.SecurityClassifications, classification)
	if err := ValidateSecurityClassifications(classifications, bundle.Providers); err == nil || !strings.Contains(err.Error(), "duplicate provider/spec scope") {
		t.Fatalf("duplicate-scope error = %v", err)
	}
}

func reviewedCatalogBundleForTest(t *testing.T) CatalogBundle {
	t.Helper()
	content, err := os.ReadFile("data/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := ParseCatalogBundle(content)
	if err != nil {
		t.Fatal(err)
	}
	bundle.Candidates = cloneCandidates(bundle.Candidates)
	bundle.Providers = cloneProviders(bundle.Providers)
	bundle.SecurityClassifications = cloneSecurityClassifications(bundle.SecurityClassifications)
	bundle.SecurityOverlays = cloneSecurityOverlays(bundle.SecurityOverlays)
	return bundle
}
