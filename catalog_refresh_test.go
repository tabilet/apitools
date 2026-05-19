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

func TestRefreshCatalogSpecReferencesSavesParseableInvalidOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Parseable Invalid","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-parseable-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshParseableOpenAPIInvalid {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.ValidationError == "" {
		t.Fatalf("ValidationError is empty")
	}
	if result.Metadata.Title != "Parseable Invalid" || result.Metadata.OpenAPI != "3.0.0" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("saved artifact: %v", err)
	}
	if !hasRefreshFollowUp(result, "strict validation errors") {
		t.Fatalf("missing strict validation follow-up: %#v", result.ManualFollowUps)
	}
}

func TestRefreshCatalogSpecReferencesSavesParseableInvalidSwagger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"swagger":"2.0","info":{"title":"Parseable Swagger","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`))
	}))
	defer server.Close()

	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-parseable-swagger",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/swagger.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshParseableSwaggerInvalid {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.ValidationError == "" {
		t.Fatalf("ValidationError is empty")
	}
	if result.Metadata.Title != "Parseable Swagger" || result.Metadata.Swagger != "2.0" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
}

func TestRefreshCatalogSpecReferencesRejectsUnparseableOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"title":"Not OpenAPI","version":"1.0.0"},"paths":{}}`))
	}))
	defer server.Close()

	_, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-invalid",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "does not validate as OpenAPI or Swagger") {
		t.Fatalf("err = %v, want OpenAPI validation error", err)
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

func hasRefreshFollowUp(result CatalogSpecRefreshResult, text string) bool {
	for _, followUp := range result.ManualFollowUps {
		if strings.Contains(followUp, text) {
			return true
		}
	}
	return false
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
