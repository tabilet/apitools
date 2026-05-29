package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/sqlitecache"
)

func TestSearchHelpDocumentsFlags(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"search", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	text := out.String()
	for _, expected := range []string{"Usage: apitools search", "-query", "-limit", "-source", "-public-probe", "-probe-timeout", "-probe-budget", "-cache", "-cache-mode", "-cache-ttl", "-offline", "-json"} {
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
	for _, expected := range []string{"Usage: apitools import", "-url", "-dir", "-name", "-cache", "-cache-mode", "-cache-ttl", "-offline", "-json"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestCatalogHelpDocumentsSubcommands(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"catalog", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	text := out.String()
	for _, expected := range []string{"Usage: apitools catalog", "advisory", "check", "list", "specs", "stats", "refresh-report", "inspect", "overlay-view", "security-audit", "security-report"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestOAuthHelpDocumentsSubcommands(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"oauth", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "google") {
		t.Fatalf("oauth help missing google command:\n%s", out.String())
	}

	out.Reset()
	code = run([]string{"oauth", "google", "login", "--help"}, &out, &out)
	if code != 0 {
		t.Fatalf("code = %d\n%s", code, out.String())
	}
	for _, expected := range []string{"-client-id", "-client-secret-env", "-scope", "-listen", "-code"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("oauth google login help missing %q:\n%s", expected, out.String())
		}
	}
}

func TestOAuthGoogleLoginCodeExchangePrintsEnvMarkerHCL(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_SECRET_TEST", "client-secret")
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "auth-code" {
			t.Fatalf("unexpected token form: %#v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"access-token","refresh_token":"refresh-token","token_type":"Bearer","expires_in":3600}`)
	}))
	defer tokenServer.Close()

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{
		"oauth", "google", "login",
		"--client-id", "client-id",
		"--client-secret-env", "GOOGLE_CLIENT_SECRET_TEST",
		"--scope", "https://www.googleapis.com/auth/gmail.send",
		"--code", "auth-code",
		"--redirect-url", "http://127.0.0.1:8765/oauth2/callback",
		"--token-url", tokenServer.URL,
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{
		"export GOOGLE_REFRESH_TOKEN='refresh-token'",
		"credentials {",
		"googleOAuth2 {",
		`client_secret = "ENVIRONMENT:GOOGLE_CLIENT_SECRET_TEST"`,
		`refresh_token = "ENVIRONMENT:GOOGLE_REFRESH_TOKEN"`,
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("oauth output missing %q:\n%s", expected, out.String())
		}
	}
	if strings.Contains(out.String(), "client-secret") {
		t.Fatalf("oauth output leaked client secret:\n%s", out.String())
	}
}

func TestCatalogAdvisoryOutputAndJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "advisory", "grafana"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		"Provider: Grafana (grafana)",
		"OpenAPI availability: known",
		"Auth status: complete",
		"Resolved OpenAPI: built-in-spec-reference",
		"grafana-http-api-openapi-v3",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog advisory output missing %q:\n%s", expected, text)
		}
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "advisory", "--json", "grafana"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"provider_id": "grafana"`, `"auth_status": "complete"`, `"spec_ref_id": "grafana-http-api-openapi-v3"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog advisory json missing %q:\n%s", expected, out.String())
		}
	}
}

func TestCatalogAdvisoryShowsRegisteredArtifactPath(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.sqlite")
	artifactPath := filepath.Join("openapi", "slack-web-openapi-v2.json")
	if err := os.MkdirAll(filepath.Join(dir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(artifactPath)), []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       artifactPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "advisory", "slack", "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "artifact: "+artifactPath) {
		t.Fatalf("catalog advisory output missing registered artifact path:\n%s", out.String())
	}
}

func TestCatalogAdvisoryShowsRegisteredEndpointOverlay(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.sqlite")
	overlayPath := filepath.Join("advisory-overlays", "activecampaign-api-v3-overlay.json")
	builderPath := filepath.Join("overlay-builders", "build_m21_human_docs_overlays.go")
	if err := os.MkdirAll(filepath.Join(dir, "advisory-overlays"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(overlayPath)), []byte(`{"openapi":"3.0.3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID:  "activecampaign",
		ArtifactID:  "activecampaign-api-v3-overlay",
		Kind:        "advisory-overlay",
		Path:        overlayPath,
		OverlayPath: overlayPath,
		BuilderPath: builderPath,
		Metadata: map[string]string{
			"official_openapi":  "false",
			"derived_from_docs": "true",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "advisory", "activecampaign", "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{
		"Endpoint overlays:",
		"activecampaign-api-v3-overlay",
		overlayPath,
		"builder: " + builderPath,
		"metadata: derived_from_docs=true, official_openapi=false",
		"Review registered advisory endpoint overlay metadata",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog advisory output missing %q:\n%s", expected, out.String())
		}
	}
	if strings.Contains(out.String(), "OpenAPI-only workflows likely need a user-provided or generated OpenAPI document before import.") {
		t.Fatalf("catalog advisory output kept generic OpenAPI follow-up despite endpoint overlay:\n%s", out.String())
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "advisory", "activecampaign", "--cache", cachePath, "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"endpoint_overlays"`, `"artifact_id": "activecampaign-api-v3-overlay"`, `"builder_path": "` + builderPath + `"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog advisory json missing %q:\n%s", expected, out.String())
		}
	}
}

func TestCatalogAdvisoryMissingCacheDoesNotCreateFile(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "missing.sqlite")
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "advisory", "slack", "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache path stat err = %v, want not exist", err)
	}
}

func TestCatalogAdvisoryUnknownProvider(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "advisory", "missing"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), `unknown provider "missing"`) {
		t.Fatalf("stderr missing unknown provider error:\n%s", errOut.String())
	}
}

func TestCatalogSpecsOutputAndJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "specs"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{"PROVIDER", "PROTOCOL", "slack-web-openapi-v2", "openapi", "swagger 2.0", "smithy", "google-discovery", "dropbox-stone", "official-github"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog specs output missing %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "human-docs") {
		t.Fatalf("catalog specs output should not include human docs:\n%s", text)
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "specs", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"spec_ref_id": "slack-web-openapi-v2"`, `"protocol": "swagger"`, `"protocol_version": "2.0"`, `"uws_source_type": "openapi"`, `"uws_source_type": "aws-smithy"`, `"uws_source_type": "google-discovery"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog specs json missing %q:\n%s", expected, out.String())
		}
	}
}

func TestCatalogSpecsShowsRegisteredArtifactPath(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.sqlite")
	artifactPath := filepath.Join("openapi", "slack-web-openapi-v2.json")
	if err := os.MkdirAll(filepath.Join(dir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(artifactPath)), []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       artifactPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "specs", "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), artifactPath) {
		t.Fatalf("catalog specs output missing registered artifact path:\n%s", out.String())
	}
}

func TestCatalogResolveMaterializeAndExport(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "catalog-openapi-cache")
	cachePath := filepath.Join(cacheDir, "cache.sqlite")
	artifactPath := filepath.Join("openapi", "slack-web-openapi-v2.json")
	if err := os.MkdirAll(filepath.Join(cacheDir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, filepath.FromSlash(artifactPath)), []byte(`{"swagger":"2.0","info":{"title":"Slack","version":"1.0.0"},"paths":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       artifactPath,
		SourceURL:  "https://example.com/slack.json",
		Metadata:   map[string]string{"validation_status": "valid-swagger"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "resolve", "slack", "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("resolve code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{"Provider resolutions: 1 provider(s)", "Slack (slack): executable-openapi", artifactPath} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("resolve output missing %q:\n%s", expected, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	materializeDir := filepath.Join(dir, "materialized")
	code = run([]string{"catalog", "materialize", "slack", "--out", materializeDir, "--cache-dir", cacheDir, "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("materialize code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Materialized provider: Slack (slack)") || !strings.Contains(out.String(), "slack-web-openapi-v2.json") {
		t.Fatalf("materialize output missing expected text:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(materializeDir, "slack", "openapi", "slack-web-openapi-v2.json")); err != nil {
		t.Fatalf("missing materialized artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(materializeDir, "slack", "provenance.json")); err != nil {
		t.Fatalf("missing materialized provenance: %v", err)
	}

	out.Reset()
	errOut.Reset()
	workflowDir := filepath.Join(dir, "workflow")
	code = run([]string{"catalog", "export", "slack", "--workflow-dir", workflowDir, "--cache-dir", cacheDir, "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("export code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Exported workflow artifacts: 1 provider(s)") {
		t.Fatalf("export output missing summary:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(workflowDir, "api-artifacts", "provenance.json")); err != nil {
		t.Fatalf("missing export provenance: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workflowDir, "api-artifacts", "slack", "openapi", "slack-web-openapi-v2.json")); err != nil {
		t.Fatalf("missing exported artifact: %v", err)
	}
}

func TestCatalogStatsOutputAndJSON(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "catalog-openapi-cache")
	cachePath := filepath.Join(cacheDir, "cache.sqlite")
	artifactPath := filepath.Join("openapi", "slack-web-openapi-v2.json")
	if err := os.MkdirAll(filepath.Join(cacheDir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, filepath.FromSlash(artifactPath)), []byte(`{"swagger":"2.0","info":{"title":"Slack","version":"1.0.0"},"paths":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       artifactPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "stats", "--cache-dir", cacheDir, "--cache", cachePath, "--as-of", "2026-05-19"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{"Provider protocols: 316 provider(s)", "OpenAPI", "94", "Swagger", "13", "Smithy", "29", "Google Discovery", "21", "Human docs", "157", "Artifact registry", "openapi", "Refresh artifacts", "valid-swagger"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog stats output missing %q:\n%s", expected, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "stats", "--cache-dir", cacheDir, "--cache", cachePath, "--as-of", "2026-05-19", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"provider_count": 316`, `"protocol": "openapi"`, `"count": 94`, `"protocol": "swagger"`, `"count": 13`, `"protocol": "smithy"`, `"count": 29`, `"protocol": "google-discovery"`, `"count": 21`, `"artifact_registry"`, `"kind": "openapi"`, `"status": "valid-swagger"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("catalog stats json missing %q:\n%s", expected, out.String())
		}
	}
}

func TestCatalogRefreshCommandRegistersSelectedSpec(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "catalog-openapi-cache")
	cachePath := filepath.Join(cacheDir, "cache.sqlite")
	var selected []catalog.RefreshableSpecReference
	refresh := func(ctx context.Context, rows []catalog.RefreshableSpecReference, opts apitools.CatalogSpecRefreshOptions) (apitools.CatalogSpecRefreshReport, error) {
		selected = append([]catalog.RefreshableSpecReference(nil), rows...)
		artifactPath := "openapi/slack-web-openapi-v2.json"
		if rows[0].RegisteredArtifactPath != "" {
			artifactPath = rows[0].RegisteredArtifactPath
		}
		fullPath := filepath.Join(opts.CacheDir, filepath.FromSlash(artifactPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return apitools.CatalogSpecRefreshReport{}, err
		}
		if err := os.WriteFile(fullPath, []byte(`{"openapi":"3.0.0","info":{"title":"Slack","version":"1.0.0"},"paths":{}}`), 0o644); err != nil {
			return apitools.CatalogSpecRefreshReport{}, err
		}
		return apitools.CatalogSpecRefreshReport{Results: []apitools.CatalogSpecRefreshResult{{
			ProviderID:       rows[0].ProviderID,
			SpecRefID:        rows[0].SpecRefID,
			Kind:             rows[0].Kind,
			Protocol:         catalog.SpecProtocolSwagger,
			ProtocolVersion:  "2.0",
			URL:              rows[0].URL,
			FinalURL:         rows[0].URL,
			DownloadStatus:   apitools.CatalogRefreshDownloaded,
			ValidationStatus: apitools.CatalogRefreshValidSwagger,
			ArtifactPath:     artifactPath,
			SavedPath:        fullPath,
			SHA256:           "abc",
			Bytes:            72,
			Metadata:         apitools.SpecMetadata{Title: "Slack", OpenAPI: "3.0.0"},
			ManualFollowUps:  []string{"review manually"},
		}}}, nil
	}
	var out bytes.Buffer
	var errOut bytes.Buffer
	priorCache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "openapi", "registered-slack.json"), []byte(`{"openapi":"3.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := priorCache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       "openapi/registered-slack.json",
	}); err != nil {
		t.Fatal(err)
	}
	if err := priorCache.Close(); err != nil {
		t.Fatal(err)
	}

	code := runCatalogRefreshWithClient([]string{
		"--provider", "slack",
		"--cache-dir", cacheDir,
		"--cache", cachePath,
	}, &out, &errOut, func(*apitools.Client) catalogRefreshFunc { return refresh })
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if got, want := len(selected), 1; got != want {
		t.Fatalf("selected len = %d, want %d", got, want)
	}
	if selected[0].SpecRefID != "slack-web-openapi-v2" {
		t.Fatalf("selected spec = %q", selected[0].SpecRefID)
	}
	if selected[0].RegisteredArtifactPath != "openapi/registered-slack.json" {
		t.Fatalf("selected registered artifact path = %q", selected[0].RegisteredArtifactPath)
	}
	for _, expected := range []string{"PROTOCOL", "swagger 2.0", "Manual follow-ups"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("refresh output missing %q:\n%s", expected, out.String())
		}
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	artifacts, err := cache.ListCatalogArtifacts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(artifacts), 1; got != want {
		t.Fatalf("artifacts len = %d, want %d: %#v", got, want, artifacts)
	}
	if artifacts[0].Path != "openapi/registered-slack.json" {
		t.Fatalf("artifact path = %q", artifacts[0].Path)
	}
}

func TestCatalogRefreshRejectsProviderWithoutRefreshableSpecs(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCatalogRefreshWithClient([]string{"--provider", "mailchimp"}, &out, &errOut, func(*apitools.Client) catalogRefreshFunc {
		return func(context.Context, []catalog.RefreshableSpecReference, apitools.CatalogSpecRefreshOptions) (apitools.CatalogSpecRefreshReport, error) {
			t.Fatal("refresh should not run")
			return apitools.CatalogSpecRefreshReport{}, nil
		}
	})
	if code != 2 {
		t.Fatalf("code = %d, want 2\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "has no refreshable spec references") {
		t.Fatalf("stderr missing no refreshable specs message:\n%s", errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "catalog-openapi-cache", "cache.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("selection failure created default cache: %v", err)
	}
}

func TestCatalogRefreshReportOutputAndJSON(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "catalog-openapi-cache")
	cachePath := filepath.Join(cacheDir, "cache.sqlite")
	artifactPath := "openapi/slack-web-openapi-v2.json"
	if err := os.MkdirAll(filepath.Join(cacheDir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, filepath.FromSlash(artifactPath)), []byte(`{"openapi":"3.0.0","info":{"title":"Slack","version":"1.0.0"},"paths":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.StoreCatalogArtifact(context.Background(), sqlitecache.CatalogArtifact{
		ProviderID: "slack",
		ArtifactID: "slack-web-openapi-v2",
		Kind:       "openapi",
		Path:       artifactPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "refresh-report", "--cache-dir", cacheDir, "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{"slack-web-openapi-v2", "openapi 3.0.0", "valid-openapi", artifactPath, "Manual follow-ups"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("refresh report output missing %q:\n%s", expected, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "refresh-report", "--cache-dir", cacheDir, "--cache", cachePath, "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"provider_id": "slack"`, `"spec_ref_id": "slack-web-openapi-v2"`, `"protocol": "openapi"`, `"protocol_version": "3.0.0"`, `"validation_status": "valid-openapi"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("refresh report json missing %q:\n%s", expected, out.String())
		}
	}
}

func TestCatalogRefreshReportMissingCacheDoesNotCreateFile(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "missing.sqlite")
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "refresh-report", "--cache-dir", dir, "--cache", cachePath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("cache path stat err = %v, want not exist", err)
	}
	if !strings.Contains(out.String(), apitools.CatalogRefreshMissingRegistration) {
		t.Fatalf("refresh report missing unregistered status:\n%s", out.String())
	}
}

func TestCatalogCheckOutputAndExitCode(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "check", "--as-of", "2026-05-18"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Catalog quality: 0 error(s), 0 warning(s)") {
		t.Fatalf("catalog check output missing clean summary:\n%s", out.String())
	}
}

func TestCatalogCheckJSONOutput(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "check", "--as-of", "2026-05-18", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("catalog check json = %s, want empty report object", out.String())
	}
}

func TestCatalogCheckExitCodeAllowsWarnings(t *testing.T) {
	report := catalog.CatalogQualityReport{Findings: []catalog.CatalogQualityFinding{{
		Severity: catalog.CatalogQualityWarning,
		Code:     "stale-verification-date",
		Message:  "stale",
	}}}
	if code := catalogCheckExitCode(report); code != 0 {
		t.Fatalf("warning-only exit code = %d, want 0", code)
	}
}

func TestCatalogCheckExitCodeFailsErrors(t *testing.T) {
	report := catalog.CatalogQualityReport{Findings: []catalog.CatalogQualityFinding{{
		Severity: catalog.CatalogQualityError,
		Code:     "missing-security-status",
		Message:  "missing",
	}}}
	if code := catalogCheckExitCode(report); code != 1 {
		t.Fatalf("error exit code = %d, want 1", code)
	}
}

func TestCatalogCheckCommandFailsErrorReport(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCatalogCheckWithReport(nil, &out, &errOut, func(catalog.CatalogQualityOptions) catalog.CatalogQualityReport {
		return catalog.CatalogQualityReport{Findings: []catalog.CatalogQualityFinding{{
			Severity: catalog.CatalogQualityError,
			Code:     "missing-security-status",
			Message:  "missing",
		}}}
	})
	if code != 1 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Catalog quality: 1 error(s), 0 warning(s)") {
		t.Fatalf("catalog check output missing error summary:\n%s", out.String())
	}
}

func TestCatalogListOutputIsDeterministic(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	airtable := strings.Index(text, "airtable")
	gmail := strings.Index(text, "gmail")
	if airtable < 0 || gmail < 0 {
		t.Fatalf("catalog list missing expected providers:\n%s", text)
	}
	if airtable > gmail {
		t.Fatalf("catalog list not deterministic by provider id:\n%s", text)
	}
	if !strings.Contains(text, "overlay-required") || !strings.Contains(text, "complete") {
		t.Fatalf("catalog list missing auth status:\n%s", text)
	}
}

func TestCatalogInspectShowsResolutionAndNotes(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "inspect", "slack"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		"Provider: Slack (slack)",
		"Resolved OpenAPI: built-in-spec-reference",
		"Resolved security: built-in-security-overlay",
		"Auth status: present-incomplete",
		"slack-web-openapi-v2 [openapi/swagger 2.0]",
		"slack-web-api-auth-review",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("inspect output missing %q:\n%s", expected, text)
		}
	}
}

func TestCatalogInspectUserOpenAPIOverridesBuiltInSecurity(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "inspect", "slack", "--openapi", "./openapi/slack.yaml"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		"Resolved OpenAPI: user-openapi",
		"./openapi/slack.yaml",
		"Resolved security: none",
		"Auth status: unknown",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("inspect override output missing %q:\n%s", expected, text)
		}
	}
}

func TestCatalogInspectAcceptsFlagsBeforeProvider(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "inspect", "--json", "slack"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), `"id": "slack"`) {
		t.Fatalf("inspect json missing provider:\n%s", out.String())
	}
}

func TestCatalogSecurityReportJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "security-report", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, `"provider_id": "airtable"`) || !strings.Contains(text, `"status": "overlay-required"`) {
		t.Fatalf("security report json missing expected metadata:\n%s", text)
	}
}

func TestCatalogSecurityAuditOutputAndJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "security-audit"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		"Catalog security audit: 316 provider(s)",
		"Disposition:",
		"Auth status:",
		"Queued source re-review: none",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("security audit output missing %q:\n%s", expected, text)
		}
	}

	out.Reset()
	errOut.Reset()
	code = run([]string{"catalog", "security-audit", "--json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("json code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	for _, expected := range []string{`"provider_count": 316`, `"disposition": "complete-via-overlay"`, `"provider_id": "github"`} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("security audit json missing %q:\n%s", expected, out.String())
		}
	}
	if strings.Contains(out.String(), `"disposition": "queued-source-re-review"`) {
		t.Fatalf("security audit json has queued source re-review rows:\n%s", out.String())
	}
}

func TestCatalogOverlayViewShowsProvenanceAndConflicts(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "overlay-view", "github"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		"Provider: GitHub (github)",
		"Auth status: overlay-required",
		"Classification: classification spec=github-rest-api-openapi status=overlay-required",
		"githubBearer [overlay] overlay=github-rest-api-auth-overlay",
		"overlay-only-addition scheme=githubBearer overlay=github-rest-api-auth-overlay",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("overlay-view output missing %q:\n%s", expected, text)
		}
	}
}

func TestCatalogOverlayViewJSON(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "overlay-view", "--json", "github"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	text := out.String()
	for _, expected := range []string{
		`"provider_id": "github"`,
		`"provenance": "overlay"`,
		`"type": "overlay-only-addition"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("overlay-view json missing %q:\n%s", expected, text)
		}
	}
}

func TestCatalogListRejectsUnexpectedArgs(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"catalog", "list", "slack"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("code = %d\nstdout:\n%s\nstderr:\n%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "unexpected argument") {
		t.Fatalf("expected unexpected argument error, got:\n%s", errOut.String())
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
	key := apitools.SearchCacheKey{Query: "mail", Source: apitools.SourceAuto, Limit: 10}
	report := apitools.SearchReport{
		Query:  "mail",
		Source: apitools.SourceAuto,
		Results: []apitools.Result{{
			ID:      "apis-guru:mail:v1",
			Source:  string(apitools.SourceAPIsGuru),
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
	key := apitools.SearchCacheKey{Query: "mail", Source: apitools.SourceAuto, Limit: 10, PublicProbe: 2}
	report := apitools.SearchReport{
		Query:  "mail",
		Source: apitools.SourceAuto,
		Results: []apitools.Result{{
			ID:      "public-apis:mail:https://example.com/openapi.yaml",
			Source:  string(apitools.SourcePublicAPIs),
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

	var client *apitools.Client
	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runSearchWithClient([]string{
		"--query", "mail",
		"--source", string(apitools.SourcePublicAPIs),
		"--limit", "1",
		"--public-probe", "3",
		"--probe-timeout", "10ms",
		"--probe-budget", "15ms",
	}, &out, &errOut, func(string) (*apitools.Client, func(), error) {
		client = &apitools.Client{
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
	if !strings.Contains(errOut.String(), apitools.ErrProbeBudgetExceeded.Error()) {
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
	if err := cache.StoreSpec(context.Background(), apitools.CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte("openapi: 3.0.0\ninfo:\n  title: Mail\n  version: 1.0.0\npaths: {}\n"),
		Metadata:    apitools.SpecMetadata{Title: "Mail", OpenAPI: "3.0.0"},
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
