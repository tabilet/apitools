package sqlitecache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
)

func sha256Hex(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func TestRegisterCatalogRefreshResults(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "catalog-openapi-cache", "cache.sqlite")
	artifactPath := filepath.Join(dir, "catalog-openapi-cache", "openapi", "demo.yaml")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"openapi":"3.0.0","info":{"title":"Demo","version":"1"},"paths":{}}`)
	if err := os.WriteFile(artifactPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	report := apitools.CatalogSpecRefreshReport{Results: []apitools.CatalogSpecRefreshResult{{
		ProviderID:          "demo",
		SpecRefID:           "demo-openapi",
		Kind:                catalog.SpecKindOpenAPI,
		URL:                 "https://example.com/openapi.yaml",
		FinalURL:            "https://cdn.example.com/openapi.yaml",
		RawValidationStatus: apitools.CatalogRefreshValidOpenAPI,
		ArtifactPath:        "openapi/demo.yaml",
		SavedPath:           artifactPath,
		SHA256:              sha256Hex(content),
		Bytes:               int64(len(content)),
		RawMetadata:         apitools.SpecMetadata{Title: "Demo", OpenAPI: "3.0.0"},
	}}}
	if err := RegisterCatalogRefreshResults(context.Background(), cachePath, "unused", report); err != nil {
		t.Fatal(err)
	}
	cache, err := Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	artifacts, err := cache.ListCatalogArtifacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(artifacts), 1; got != want {
		t.Fatalf("artifacts len = %d, want %d", got, want)
	}
	if got, want := artifacts[0].Path, "openapi/demo.yaml"; got != want {
		t.Fatalf("artifact path = %q, want %q", got, want)
	}
	if got, want := artifacts[0].SHA256, sha256Hex(content); got != want {
		t.Fatalf("artifact SHA256 = %q, want %q", got, want)
	}
	stored, ok, err := cache.LoadSpec(context.Background(), report.Results[0].URL, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || stored.ContentPath != "openapi/demo.yaml" || stored.Metadata.Title != "Demo" {
		t.Fatalf("stored spec = %#v, ok = %v", stored, ok)
	}
}

func TestRegisterCatalogRefreshResultsRejectsUnsafePaths(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "catalog-openapi-cache", "cache.sqlite")
	outside := filepath.Join(dir, "outside.yaml")
	if err := os.WriteFile(outside, []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := apitools.CatalogSpecRefreshReport{Results: []apitools.CatalogSpecRefreshResult{{
		ProviderID:   "demo",
		SpecRefID:    "demo-openapi",
		Kind:         catalog.SpecKindOpenAPI,
		URL:          "https://example.com/openapi.yaml",
		ArtifactPath: "openapi/demo.yaml",
		SavedPath:    outside,
	}}}
	if err := RegisterCatalogRefreshResults(context.Background(), cachePath, "unused", report); err == nil || !strings.Contains(err.Error(), "must stay under SQLite cache directory") {
		t.Fatalf("outside registration error = %v", err)
	}
	if err := RegisterCatalogRefreshResults(context.Background(), "", "unused", report); err == nil || !strings.Contains(err.Error(), "cache path is required") {
		t.Fatalf("missing cache path error = %v", err)
	}
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

func TestSearchReportRFC9727ProviderURLIsolatesCacheKey(t *testing.T) {
	cache, err := Open(filepath.Join(t.TempDir(), "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	key := apitools.SearchCacheKey{Query: "mail", Source: apitools.SourceRFC9727, Limit: 3, ProviderURL: "https://mail.example"}
	report := apitools.SearchReport{
		Query:  "mail",
		Source: apitools.SourceRFC9727,
		Results: []apitools.Result{{
			ID:           "rfc9727:mail.example:one",
			Source:       string(apitools.SourceRFC9727),
			Title:        "Mail API",
			SpecURL:      "https://mail.example/openapi.yaml",
			Provenance:   "test",
			MediaType:    "application/yaml",
			Experimental: true,
		}},
	}
	if err := cache.StoreSearch(context.Background(), key, report); err != nil {
		t.Fatal(err)
	}
	got, ok, err := cache.LoadSearch(context.Background(), key, time.Hour)
	if err != nil || !ok {
		t.Fatalf("matching provider cache lookup: ok=%t err=%v", ok, err)
	}
	if len(got.Results) != 1 || !got.Results[0].Experimental || got.Results[0].MediaType != "application/yaml" {
		t.Fatalf("cached RFC 9727 trust metadata = %#v", got.Results)
	}
	other := key
	other.ProviderURL = "https://other.example"
	if _, ok, err := cache.LoadSearch(context.Background(), other, time.Hour); err != nil || ok {
		t.Fatalf("different provider cache lookup: ok=%t err=%v", ok, err)
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
	for _, table := range []string{"search_results", "search_queries", "spec_documents", "catalog_artifacts"} {
		columns, err := cache.tableColumns(context.Background(), table)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := columns["accessed_at"]; !ok {
			t.Fatalf("%s accessed_at column missing after migration", table)
		}
	}
	var version int
	if err := cache.db.QueryRowContext(context.Background(), `SELECT version FROM schema_meta`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != schemaVersion {
		t.Fatalf("schema version = %d, want %d", version, schemaVersion)
	}
}

func TestStoreSpecRejectsCallerSHA256Mismatch(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	rawURL := "https://example.com/openapi.yaml"
	err = cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		SHA256:      "not-the-real-digest",
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	})
	if err == nil {
		t.Fatal("expected caller digest mismatch")
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

func TestSpecLoadRejectsMissingDigestAndByteMismatch(t *testing.T) {
	for _, test := range []struct {
		name   string
		column string
		value  any
	}{
		{name: "missing digest", column: "sha256", value: ""},
		{name: "byte mismatch", column: "bytes", value: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.db.ExecContext(context.Background(), `UPDATE spec_documents SET `+test.column+` = ? WHERE url = ?`, test.value, rawURL); err != nil {
				t.Fatal(err)
			}
			_, ok, err := cache.LoadSpec(context.Background(), rawURL, time.Hour)
			if !errors.Is(err, apitools.ErrCachedSpecIntegrity) || ok {
				t.Fatalf("LoadSpec() = ok %t, err %v; want integrity error", ok, err)
			}
		})
	}
}

func TestPathBackedCacheRejectsUnsafeArtifactPaths(t *testing.T) {
	dir := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.yaml")
	content := []byte("openapi: 3.0.0\ninfo:\n  title: Outside\n  version: 1\npaths: {}\n")
	if err := os.WriteFile(outsidePath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := Open(filepath.Join(dir, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	for _, path := range []string{outsidePath, "../outside.yaml"} {
		err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
			OriginalURL: "https://example.com/" + filepath.Base(path),
			ContentPath: path,
		})
		if err == nil {
			t.Fatalf("StoreSpec(%q) succeeded", path)
		}
		err = cache.StoreCatalogArtifact(context.Background(), CatalogArtifact{
			ProviderID: "provider", ArtifactID: filepath.Base(path), Kind: "openapi", Path: path,
		})
		if err == nil {
			t.Fatalf("StoreCatalogArtifact(%q) succeeded", path)
		}
	}

	linkPath := filepath.Join(dir, "linked.yaml")
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: "https://example.com/linked", ContentPath: "linked.yaml",
	}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("StoreSpec(symlink) error = %v", err)
	}
	if got, err := os.ReadFile(outsidePath); err != nil || string(got) != string(content) {
		t.Fatalf("outside file changed: content=%q err=%v", got, err)
	}
}

func TestSpecLoadRejectsTamperedPathEscape(t *testing.T) {
	dir := t.TempDir()
	outsidePath := filepath.Join(filepath.Dir(dir), "sqlitecache-outside-"+filepath.Base(dir)+".yaml")
	if err := os.WriteFile(outsidePath, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outsidePath) })
	cache, err := Open(filepath.Join(dir, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	rawURL := "https://example.com/openapi.yaml"
	content := []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1\npaths: {}\n")
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{OriginalURL: rawURL, Content: content}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../" + filepath.Base(outsidePath), outsidePath} {
		if _, err := cache.db.ExecContext(context.Background(), `UPDATE spec_documents SET content_path = ?, content = NULL WHERE url = ?`, path, rawURL); err != nil {
			t.Fatal(err)
		}
		_, ok, err := cache.LoadSpec(context.Background(), rawURL, time.Hour)
		if err == nil || ok {
			t.Fatalf("LoadSpec(%q) = ok %t, err %v; want rejection", path, ok, err)
		}
	}
}

func TestCatalogArtifactRejectsCallerIntegrityMismatch(t *testing.T) {
	dir := t.TempDir()
	cache, err := Open(filepath.Join(dir, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if err := os.WriteFile(filepath.Join(dir, "artifact.json"), []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []CatalogArtifact{
		{ProviderID: "p", ArtifactID: "bad-sha", Kind: "openapi", Path: "artifact.json", SHA256: strings.Repeat("0", 64)},
		{ProviderID: "p", ArtifactID: "bad-bytes", Kind: "openapi", Path: "artifact.json", Bytes: 1},
	} {
		if err := cache.StoreCatalogArtifact(context.Background(), artifact); err == nil {
			t.Fatalf("StoreCatalogArtifact(%s) succeeded", artifact.ArtifactID)
		}
	}
}

func TestOpenRejectsSymlinkCachePath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.sqlite")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "cache.sqlite")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if cache, err := Open(link); err == nil {
		_ = cache.Close()
		t.Fatal("Open(symlink) succeeded")
	}
}

func TestConfiguredByteBudgetsRejectOversizedRecords(t *testing.T) {
	cache, err := OpenWithOptions(":memory:", Options{MaxArtifactBytes: 4, MaxSearchReportBytes: 4, MaxMetadataBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: "https://example.com/openapi.yaml", Content: []byte("12345"),
	}); err == nil {
		t.Fatal("oversized spec was cached")
	}
	if err := cache.StoreSearch(context.Background(), apitools.SearchCacheKey{Query: "mail"}, apitools.SearchReport{Query: "mail"}); err == nil {
		t.Fatal("oversized search report was cached")
	}
	if err := cache.StoreSearch(context.Background(), apitools.SearchCacheKey{Query: "metadata-over-budget"}, apitools.SearchReport{}); err == nil {
		t.Fatal("oversized search key was cached")
	}
}

func TestCatalogListRejectsTamperedPathsAndIntegrityMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		column string
		value  any
	}{
		{name: "artifact path escape", column: "path", value: "../outside.json"},
		{name: "overlay path escape", column: "overlay_path", value: "../outside.json"},
		{name: "builder path absolute", column: "builder_path", value: "/tmp/builder.go"},
		{name: "missing digest", column: "sha256", value: ""},
		{name: "missing bytes", column: "bytes", value: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			cache, err := Open(filepath.Join(dir, "cache.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer cache.Close()
			if err := os.WriteFile(filepath.Join(dir, "artifact.json"), []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := cache.StoreCatalogArtifact(context.Background(), CatalogArtifact{
				ProviderID: "p", ArtifactID: "a", Kind: "openapi", Path: "artifact.json",
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.db.ExecContext(context.Background(), `UPDATE catalog_artifacts SET `+test.column+` = ?`, test.value); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.ListCatalogArtifacts(context.Background()); err == nil {
				t.Fatal("tampered catalog row was accepted")
			}
		})
	}
}

func TestConfiguredSearchLRUEvictsLeastRecentlyUsedQueryAndResult(t *testing.T) {
	cache, err := OpenWithOptions(":memory:", Options{MaxSearchQueries: 2, MaxSearchResults: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	ctx := context.Background()
	store := func(id string) apitools.SearchCacheKey {
		key := apitools.SearchCacheKey{Query: id, Source: apitools.SourceAuto, Limit: 1}
		report := apitools.SearchReport{Query: id, Results: []apitools.Result{{ID: id, Title: id, SpecURL: "https://example.com/" + id}}}
		if err := cache.StoreSearch(ctx, key, report); err != nil {
			t.Fatal(err)
		}
		return key
	}
	a := store("a")
	b := store("b")
	if _, ok, err := cache.LoadSearch(ctx, a, time.Hour); err != nil || !ok {
		t.Fatalf("touch a: ok=%t err=%v", ok, err)
	}
	c := store("c")
	for _, check := range []struct {
		key  apitools.SearchCacheKey
		want bool
	}{{a, true}, {b, false}, {c, true}} {
		if _, ok, err := cache.LoadSearch(ctx, check.key, time.Hour); err != nil || ok != check.want {
			t.Fatalf("LoadSearch(%q) = ok %t, err %v; want %t", check.key.Query, ok, err, check.want)
		}
	}
	var resultIDs string
	if err := cache.db.QueryRowContext(ctx, `SELECT group_concat(id, '') FROM (SELECT id FROM search_results ORDER BY id)`).Scan(&resultIDs); err != nil {
		t.Fatal(err)
	}
	if resultIDs != "ac" {
		t.Fatalf("remaining result ids = %q, want ac", resultIDs)
	}
}

func TestConfiguredSpecLRUEvictsLeastRecentlyUsedDocument(t *testing.T) {
	cache, err := OpenWithOptions(":memory:", Options{MaxSpecDocuments: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	ctx := context.Background()
	store := func(id string) string {
		rawURL := "https://example.com/" + id
		if err := cache.StoreSpec(ctx, apitools.CachedSpec{OriginalURL: rawURL, Content: []byte(id)}); err != nil {
			t.Fatal(err)
		}
		return rawURL
	}
	a := store("a")
	b := store("b")
	if _, ok, err := cache.LoadSpec(ctx, a, time.Hour); err != nil || !ok {
		t.Fatalf("touch a: ok=%t err=%v", ok, err)
	}
	c := store("c")
	for _, check := range []struct {
		url  string
		want bool
	}{{a, true}, {b, false}, {c, true}} {
		if _, ok, err := cache.LoadSpec(ctx, check.url, time.Hour); err != nil || ok != check.want {
			t.Fatalf("LoadSpec(%q) = ok %t, err %v; want %t", check.url, ok, err, check.want)
		}
	}
}

func TestConfiguredCatalogBoundAndExplicitPruneReport(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenWithOptions(filepath.Join(dir, "cache.sqlite"), Options{MaxCatalogArtifacts: 3})
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		path := id + ".json"
		if err := os.WriteFile(filepath.Join(dir, path), []byte(id), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := cache.StoreCatalogArtifact(ctx, CatalogArtifact{ProviderID: "p", ArtifactID: id, Kind: "openapi", Path: path}); err != nil {
			t.Fatal(err)
		}
	}
	cache.options.MaxCatalogArtifacts = 1
	report, err := cache.Prune(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.CatalogArtifacts != 2 {
		t.Fatalf("catalog prune report = %#v, want 2 removals", report)
	}
	artifacts, err := cache.ListCatalogArtifacts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].ArtifactID != "c" {
		t.Fatalf("remaining artifacts = %#v, want c", artifacts)
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
