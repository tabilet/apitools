package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tabilet/apitools"
	"github.com/tabilet/apitools/sqlitecache"
)

func TestSearchHelpDocumentsFlags(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"search", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	text := out.String()
	for _, expected := range []string{"Usage: openapisearch search", "-query", "-limit", "-source", "-public-probe", "-probe-timeout", "-probe-budget", "-cache", "-cache-mode", "-cache-ttl", "-offline", "-json"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestImportHelpDocumentsFlags(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"import", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	text := out.String()
	for _, expected := range []string{"Usage: openapisearch import", "-url", "-dir", "-name", "-cache", "-cache-mode", "-cache-ttl", "-offline", "-json"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestSearchParseErrorsUseStderr(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"search", "--limit", "bad"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty stdout, got:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "invalid value") {
		t.Fatalf("expected parse error on stderr, got:\n%s", errOut.String())
	}
}

func TestSearchInvalidSourceExitsNonzero(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"search", "--query", "mail", "--source", "bad", "--offline"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "unknown source") {
		t.Fatalf("expected source error on stderr, got:\n%s", errOut.String())
	}
}

func TestSearchOfflineUsesCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.sqlite")
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	key := openapisearch.SearchCacheKey{Query: "mail", Source: openapisearch.SourceAuto, Limit: 10}
	report := openapisearch.SearchReport{
		Query:  "mail",
		Source: openapisearch.SourceAuto,
		Results: []openapisearch.Result{{
			ID:      "apis-guru:mail:v1",
			Source:  string(openapisearch.SourceAPIsGuru),
			Title:   "Mail API",
			SpecURL: "https://example.com/openapi.yaml",
		}},
	}
	if err := cache.StoreSearch(context.Background(), key, report); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"search", "--query", "mail", "--cache", cachePath, "--offline"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Mail API") {
		t.Fatalf("offline search output missing cached result:\n%s", out.String())
	}
}

func TestSearchPublicProbeFlagUsesCacheKey(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.sqlite")
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	key := openapisearch.SearchCacheKey{Query: "mail", Source: openapisearch.SourceAuto, Limit: 10, PublicProbe: 2}
	report := openapisearch.SearchReport{
		Query:  "mail",
		Source: openapisearch.SourceAuto,
		Results: []openapisearch.Result{{
			ID:      "public-apis:mail:https://example.com/openapi.yaml",
			Source:  string(openapisearch.SourcePublicAPIs),
			Title:   "Mail API",
			SpecURL: "https://example.com/openapi.yaml",
		}},
	}
	if err := cache.StoreSearch(context.Background(), key, report); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"search", "--query", "mail", "--cache", cachePath, "--offline", "--public-probe", "2"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Mail API") {
		t.Fatalf("offline search output missing cached result keyed by public probe:\n%s", out.String())
	}
}

func TestSearchProbeTimeoutAndBudgetFlagsAreWired(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entries":
			_, _ = w.Write([]byte(`{"entries":[{"API":"Slow Mail","Description":"Send mail","Link":"` + baseURL + `/docs","Category":"Communication"}]}`))
		default:
			select {
			case <-r.Context().Done():
			case <-time.After(200 * time.Millisecond):
			}
		}
	}))
	defer server.Close()
	baseURL = server.URL

	var client *openapisearch.Client
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runSearchWithClient([]string{
		"--query", "mail",
		"--source", string(openapisearch.SourcePublicAPIs),
		"--limit", "1",
		"--public-probe", "3",
		"--probe-timeout", "10ms",
		"--probe-budget", "15ms",
	}, &out, &errOut, func(string) (*openapisearch.Client, func(), error) {
		client = &openapisearch.Client{
			PublicAPIsURL:    server.URL + "/entries",
			AllowUnsafeHosts: true,
			WellKnownPaths:   []string{"/slow-1", "/slow-2", "/slow-3"},
		}
		return client, func() {}, nil
	})
	if code != 1 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if client.ProbeTimeout != 10*time.Millisecond || client.PublicProbeBudget != 15*time.Millisecond {
		t.Fatalf("probe durations not wired: timeout=%s budget=%s", client.ProbeTimeout, client.PublicProbeBudget)
	}
	if !strings.Contains(errOut.String(), openapisearch.ErrProbeBudgetExceeded.Error()) {
		t.Fatalf("expected probe budget error on stderr, got:\n%s", errOut.String())
	}
}

func TestImportOfflineUsesCachedSpec(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.sqlite")
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	rawURL := "http://93.184.216.34/openapi.yaml"
	if err := cache.StoreSpec(context.Background(), openapisearch.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    openapisearch.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"import", "--url", rawURL, "--dir", outDir, "--cache", cachePath, "--offline"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	content, err := os.ReadFile(filepath.Join(outDir, "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "title: Mail") {
		t.Fatalf("unexpected imported content:\n%s", content)
	}
}
