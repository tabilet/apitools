package sqlitecache

import (
	"context"
	"testing"
	"time"

	"github.com/tabilet/apitools"
)

func TestSearchReportRoundTrip(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	key := openapisearch.SearchCacheKey{Query: "mail", Source: openapisearch.SourceAuto, Limit: 3}
	report := openapisearch.SearchReport{
		Query:  "mail",
		Source: openapisearch.SourceAuto,
		Results: []openapisearch.Result{{
			ID:          "apis-guru:mail:v1",
			Source:      string(openapisearch.SourceAPIsGuru),
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
	key := openapisearch.SearchCacheKey{Query: "mail", Source: openapisearch.SourceAuto, Limit: 3}
	report := openapisearch.SearchReport{Query: "mail", Source: openapisearch.SourceAuto}
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
	spec := openapisearch.CachedSpec{
		OriginalURL: "https://example.com/spec",
		FinalURL:    "https://cdn.example.com/openapi.yaml",
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    openapisearch.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
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

func TestStoreSpecRecomputesSHA256(t *testing.T) {
	cache, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	rawURL := "https://example.com/openapi.yaml"
	if err := cache.StoreSpec(context.Background(), openapisearch.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		SHA256:      "not-the-real-digest",
		Metadata:    openapisearch.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
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
	if err := cache.StoreSpec(context.Background(), openapisearch.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    openapisearch.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
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
