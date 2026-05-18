package apitools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestRefreshCatalogSpecReferencesDownloadsValidOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Refresh Test","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{"description":"ok"}}}}}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("len(results) = %d, want %d", got, want)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Metadata.Title != "Refresh Test" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if result.ArtifactPath != "openapi/test-openapi.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("saved artifact: %v", err)
	}
	if result.SHA256 == "" || result.Bytes == 0 {
		t.Fatalf("missing hash/bytes: %#v", result)
	}
	if len(result.ManualFollowUps) == 0 {
		t.Fatalf("missing manual follow-ups")
	}
}

func TestRefreshCatalogSpecReferencesRejectsUnsafeHost(t *testing.T) {
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := (&Client{HTTPClient: server.Client()}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "refusing private URL host") {
		t.Fatalf("err = %v, want unsafe host rejection", err)
	}
	if requested {
		t.Fatal("unsafe refresh reached server")
	}
}

func TestRefreshCatalogSpecReferencesValidatesStructuredNonOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"discoveryVersion":"v1","title":"Discovery Test","version":"v1"}`))
	}))
	defer server.Close()

	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "google-test",
		SpecRefID:  "google-test-discovery",
		Kind:       catalog.SpecKindGoogleDiscovery,
		URL:        server.URL + "/discovery/rest",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.ArtifactPath != "google-discovery/google-test-discovery.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if result.Metadata.Title != "Discovery Test" {
		t.Fatalf("Metadata.Title = %q", result.Metadata.Title)
	}
}
