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
	for _, expected := range []string{"Usage: apitools catalog", "check", "list", "specs", "inspect", "overlay-view", "security-report"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
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
	for _, expected := range []string{"PROVIDER", "slack-web-openapi-v2", "openapi", "official-github"} {
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
	if !strings.Contains(out.String(), `"spec_ref_id": "slack-web-openapi-v2"`) {
		t.Fatalf("catalog specs json missing slack row:\n%s", out.String())
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

func TestCatalogRefreshCommandRegistersSelectedSpec(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "catalog-openapi-cache")
	cachePath := filepath.Join(cacheDir, "cache.sqlite")
	var selected []catalog.RefreshableSpecReference
	refresh := func(ctx context.Context, rows []catalog.RefreshableSpecReference, opts apitools.CatalogSpecRefreshOptions) (apitools.CatalogSpecRefreshReport, error) {
		selected = append([]catalog.RefreshableSpecReference(nil), rows...)
		artifactPath := "openapi/slack-web-openapi-v2.json"
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
	if !strings.Contains(out.String(), "Manual follow-ups") {
		t.Fatalf("refresh output missing follow-ups:\n%s", out.String())
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
	if artifacts[0].Path != "openapi/slack-web-openapi-v2.json" {
		t.Fatalf("artifact path = %q", artifacts[0].Path)
	}
}

func TestCatalogRefreshRejectsProviderWithoutRefreshableSpecs(t *testing.T) {
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
		"slack-web-openapi-v2",
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
