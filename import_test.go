package apitools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"Hello World":   "hello-world",
		"  Foo__Bar  ":  "foo-bar",
		"slack.com/v1":  "slack-com-v1",
		"!!!":           "",
		"OpenAPI v3.0!": "openapi-v3-0",
		"already-good":  "already-good",
	}
	for input, want := range cases {
		if got := sanitizeName(input); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFileNameForImportPrefersSuggestedName(t *testing.T) {
	parsed, _ := url.Parse("https://example.com/spec.json")
	got := fileNameForImport("My API.yaml", parsed, []byte(`{"a":1}`))
	if got != "my-api.yaml" {
		t.Errorf("got %q", got)
	}
}

func TestFileNameForImportFallsBackToURLPath(t *testing.T) {
	parsed, _ := url.Parse("https://example.com/path/openapi.json")
	got := fileNameForImport("", parsed, []byte(`{"a":1}`))
	if got != "openapi.json" {
		t.Errorf("got %q", got)
	}
}

func TestFileNameForImportDetectsJSONBody(t *testing.T) {
	parsed, _ := url.Parse("https://example.com/api")
	got := fileNameForImport("api", parsed, []byte(`{"openapi":"3.0.0"}`))
	if !strings.HasSuffix(got, ".json") {
		t.Errorf("expected .json suffix, got %q", got)
	}
}

func TestFileNameForImportFallsBackToHashedStem(t *testing.T) {
	got := fileNameForImport("", nil, []byte(`openapi: 3.0.0`))
	if !strings.HasPrefix(got, "openapi-") {
		t.Errorf("got %q", got)
	}
}

func TestWriteUniqueFileGeneratesNewSuffix(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	name, path, err := writeUniqueFile(dir, "spec.yaml", []byte("second"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "spec-2.yaml" || filepath.Base(path) != "spec-2.yaml" {
		t.Errorf("name = %q path = %q", name, path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "second" {
		t.Errorf("body = %q", body)
	}
}

func TestImportDownloadsAndStoresSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validOpenAPISpec))
	}))
	defer server.Close()

	dir := t.TempDir()
	c := &Client{AllowUnsafeHosts: true}
	got, err := c.Import(context.Background(), ImportOptions{URL: server.URL + "/openapi.json", Dir: dir, Name: "Example"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "example.json" {
		t.Errorf("name = %q", got.Name)
	}
	if got.Title != "Example" || got.Bytes == 0 || got.SHA256 == "" {
		t.Errorf("imported = %#v", got)
	}
	if _, err := os.Stat(got.Path); err != nil {
		t.Errorf("file missing: %v", err)
	}
}

func TestImportRejectsBlankDir(t *testing.T) {
	_, err := (&Client{}).Import(context.Background(), ImportOptions{URL: "https://example.com/x", Dir: "  "})
	if err == nil || !strings.Contains(err.Error(), "directory is required") {
		t.Errorf("got %v", err)
	}
}

func TestImportSurfacesDownloadFailures(t *testing.T) {
	dir := t.TempDir()
	_, err := (&Client{}).Import(context.Background(), ImportOptions{URL: "http://localhost/spec", Dir: dir})
	if err == nil {
		t.Errorf("expected localhost rejection")
	}
}
