package sqlitecache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/apitools"
)

func sha256Hex(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func TestSearchReportRoundTrip(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	key := apitools.SearchCacheKey{Query: "mail", Source: apitools.SourceAuto, Limit: 3}
	report := apitools.SearchReport{
		Query:  "mail",
		Source: apitools.SourceAuto,
		Results: []apitools.Result{{
			ID:          "apis-guru:mail:v1",
			Source:      string(apitools.SourceAPIsGuru),
			Provider:    "mail",
			Title:       "Mail API",
			Description: "Send mail",
			Version:     "v1",
			SpecURL:     "https://example.com/openapi.yaml",
			Score:       14,
			Provenance:  "test",
		}},
	}
	if err := cache.StoreSearch(context.Background(), key, report); err != nil {
		t.Fatal(err)
	}
	got, ok, err := cache.LoadSearch(context.Background(), key, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected cached search report")
	}
	if got.Results[0].Title != "Mail API" || got.Results[0].SpecURL != "https://example.com/openapi.yaml" {
		t.Fatalf("unexpected report: %#v", got)
	}
}

func TestSearchReportHonorsTTL(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	key := apitools.SearchCacheKey{Query: "mail", Source: apitools.SourceAuto, Limit: 3}
	report := apitools.SearchReport{Query: "mail", Source: apitools.SourceAuto}
	if err := cache.StoreSearch(context.Background(), key, report); err != nil {
		t.Fatal(err)
	}
	_, ok, err := cache.LoadSearch(context.Background(), key, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected expired cached search report")
	}
}

func TestSpecRoundTripByOriginalAndFinalURL(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	spec := apitools.CachedSpec{
		OriginalURL: "https://example.com/spec",
		FinalURL:    "https://cdn.example.com/openapi.yaml",
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	}
	if err := cache.StoreSpec(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	for _, urlValue := range []string{spec.OriginalURL, spec.FinalURL} {
		got, ok, err := cache.LoadSpec(context.Background(), urlValue, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("expected cached spec for %s", urlValue)
		}
		if got.Metadata.Title != "Mail" || string(got.Content) == "" {
			t.Fatalf("unexpected spec: %#v", got)
		}
	}
}

func TestSpecRoundTripByContentPath(t *testing.T) {
	dir := t.TempDir()
	cache, err := Open(filepath.Join(dir, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	specPath := filepath.Join(dir, "openapi", "mail.yaml")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n")
	if err := os.WriteFile(specPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	rawURL := "https://example.com/openapi.yaml"
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		ContentPath: "openapi/mail.yaml",
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := cache.LoadSpec(context.Background(), rawURL, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected cached spec")
	}
	if got.ContentPath != "openapi/mail.yaml" || string(got.Content) != string(content) {
		t.Fatalf("unexpected cached spec: %#v content=%q", got, string(got.Content))
	}
	var contentPath string
	var contentIsNull bool
	if err := cache.db.QueryRowContext(context.Background(), `SELECT content_path, content IS NULL FROM spec_documents WHERE url = ?`, rawURL).Scan(&contentPath, &contentIsNull); err != nil {
		t.Fatal(err)
	}
	if contentPath != "openapi/mail.yaml" || !contentIsNull {
		t.Fatalf("content_path = %q contentIsNull = %v, want path with no blob", contentPath, contentIsNull)
	}
}

func TestCatalogArtifactRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cache, err := Open(filepath.Join(dir, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	overlayPath := filepath.Join(dir, "advisory-overlays", "airtable.json")
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"openapi":"3.0.3","info":{"title":"Airtable","version":"advisory"},"paths":{}}`)
	if err := os.WriteFile(overlayPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), CatalogArtifact{
		ProviderID:  "airtable",
		ArtifactID:  "airtable-web-api-overlay",
		Kind:        "advisory-overlay",
		Path:        "advisory-overlays/airtable.json",
		OverlayPath: "advisory-overlays/airtable.json",
		BuilderPath: "overlay-builders/build_airtable_overlay.go",
		Metadata:    map[string]string{"source": "docs-derived"},
	}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := cache.ListCatalogArtifacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("len = %d, want 1", len(artifacts))
	}
	got := artifacts[0]
	if got.ProviderID != "airtable" || got.OverlayPath != "advisory-overlays/airtable.json" || got.BuilderPath == "" {
		t.Fatalf("unexpected artifact: %#v", got)
	}
	if got.Bytes != int64(len(content)) || got.SHA256 == "" || got.Metadata["source"] != "docs-derived" {
		t.Fatalf("unexpected artifact integrity metadata: %#v", got)
	}
}

func TestOpenMigratesBlobSpecDocumentsToPathCapableSchema(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.sqlite")
	db, err := sql.Open("sqlite", cachePath)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n")
	digest := sha256Hex(content)
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE spec_documents (
	url TEXT PRIMARY KEY,
	original_url TEXT NOT NULL,
	final_url TEXT NOT NULL,
	sha256 TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	metadata_json TEXT NOT NULL,
	content BLOB NOT NULL,
	first_seen_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO spec_documents (
	url, original_url, final_url, sha256, bytes, metadata_json, content, first_seen_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"https://example.com/openapi.yaml",
		"https://example.com/openapi.yaml",
		"https://example.com/openapi.yaml",
		digest,
		len(content),
		`{"title":"Mail","openapi":"3.0.0"}`,
		content,
		time.Now().UTC().Unix(),
		time.Now().UTC().Unix(),
	); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	cache, err := Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	got, ok, err := cache.LoadSpec(context.Background(), "https://example.com/openapi.yaml", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || string(got.Content) != string(content) {
		t.Fatalf("unexpected migrated cached spec: ok=%v spec=%#v", ok, got)
	}
	columns, err := cache.tableColumns(context.Background(), "spec_documents")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := columns["content_path"]; !ok {
		t.Fatalf("content_path column missing after migration: %#v", columns)
	}
	if columns["content"].notNull {
		t.Fatalf("content column is still NOT NULL after migration")
	}
}

func TestStoreSpecRecomputesSHA256(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	rawURL := "https://example.com/openapi.yaml"
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		SHA256:      "not-the-real-digest",
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := cache.LoadSpec(context.Background(), rawURL, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected cached spec")
	}
	if got.SHA256 == "not-the-real-digest" {
		t.Fatalf("StoreSpec preserved caller-supplied bad digest")
	}
}

func TestSpecLoadRejectsCorruptStoredSHA256(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	rawURL := "https://example.com/openapi.yaml"
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.db.ExecContext(context.Background(), `UPDATE spec_documents SET sha256 = ? WHERE url = ?`, "bad", rawURL); err != nil {
		t.Fatal(err)
	}
	_, ok, err := cache.LoadSpec(context.Background(), rawURL, time.Hour)
	if err == nil {
		t.Fatalf("expected SHA256 mismatch error")
	}
	if ok {
		t.Fatalf("expected cache miss on SHA256 mismatch")
	}
}

func TestOpenConfiguresBusyTimeout(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	var timeoutMS int
	if err := cache.db.QueryRowContext(context.Background(), `PRAGMA busy_timeout`).Scan(&timeoutMS); err != nil {
		t.Fatal(err)
	}
	if timeoutMS < 5000 {
		t.Fatalf("busy_timeout = %d, want at least 5000", timeoutMS)
	}
}
