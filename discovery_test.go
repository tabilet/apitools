package apitools

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryExtractURLsImportsAndSelects(t *testing.T) {
	urls := ExtractURLs("Use https://api.example.test/openapi.yaml, then https://api.example.test/openapi.yaml.")
	if len(urls) != 2 || urls[0] != "https://api.example.test/openapi.yaml" {
		t.Fatalf("urls = %#v", urls)
	}

	base := t.TempDir()
	openAPIDir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openAPIDir, "weather.yaml"), []byte(`openapi: 3.0.0
info:
  title: Weather API
  version: 1.0.0
  description: Forecasts and observations.
paths: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err := DiscoverOpenAPI(context.Background(), openAPIDir, base, "weather forecast")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].RelativePath != "openapi/weather.yaml" || candidates[0].Score == 0 {
		t.Fatalf("candidates = %#v", candidates)
	}
	primary, err := SelectPrimaryDiscoveryCandidate([]DiscoveryCandidate{
		{RelativePath: "openapi/low.yaml", Score: 1},
		{RelativePath: "openapi/high.yaml", Score: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if primary.RelativePath != "openapi/high.yaml" {
		t.Fatalf("primary = %#v", primary)
	}
}

func TestDiscoveryAPIsGuruRejectsPrivateListURLBeforeRequest(t *testing.T) {
	var called bool
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}
	_, err := (&Discoverer{
		HTTPClient:      client,
		APIsGuruListURL: "http://127.0.0.1/list.json",
	}).ImportBestAPIsGuruMatch(context.Background(), t.TempDir(), t.TempDir(), "weather")
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatalf("HTTP client was called before private host rejection")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
