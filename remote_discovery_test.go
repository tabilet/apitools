package apitools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSearchLAPRegistryUsesOriginalSourceURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("q"); got != "slack message" {
			t.Fatalf("q = %q, want slack message", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want 2", got)
		}
		w.Header().Set("Content-Type", "text/lap")
		_, _ = w.Write([]byte(`@search_query slack message
@total 1
@limit 2
@offset 0
@has_more false

@result
@name slack-web
@url https://registry.lap.sh/v1/apis/slack-web
@description Slack Web API
@version 1.7.0
@base https://slack.com/api
@endpoints 174
@source_url https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json
@updated 2026-08-14 03:06:55
@provider slack-com | Slack | slack.com
@endresult
`))
	}))
	defer server.Close()

	report, err := (&Client{
		LAPSearchURL:     server.URL + "/v1/search",
		AllowUnsafeHosts: true,
	}).Search(context.Background(), SearchOptions{
		Query:  "slack message",
		Limit:  2,
		Source: SourceLAPRegistry,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(report.Results), report.Results)
	}
	got := report.Results[0]
	if got.ID != "lap-registry:slack-web" || got.Source != string(SourceLAPRegistry) {
		t.Fatalf("unexpected LAP identity: %#v", got)
	}
	if got.Provider != "Slack" || got.Title != "Slack Web API" || got.Version != "1.7.0" {
		t.Fatalf("unexpected LAP metadata: %#v", got)
	}
	if got.SpecURL != "https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json" {
		t.Fatalf("SpecURL = %q", got.SpecURL)
	}
	if got.Validated || !got.Experimental {
		t.Fatalf("LAP candidate trust flags = validated:%t experimental:%t", got.Validated, got.Experimental)
	}
	if !strings.Contains(got.Provenance, "reported original source") {
		t.Fatalf("provenance = %q", got.Provenance)
	}
}

func TestSearchLAPRegistryRejectsMalformedSources(t *testing.T) {
	tests := []struct {
		name      string
		result    string
		wantError string
	}{
		{name: "missing name", result: "@source_url https://example.com/openapi.yaml\n", wantError: "result name is required"},
		{name: "missing source", result: "@name example\n", wantError: "source_url is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/lap")
				fmt.Fprintf(w, "@search_query example\n@total 1\n@result\n%s@endresult\n", tt.result)
			}))
			defer server.Close()

			client := &Client{LAPSearchURL: server.URL, AllowUnsafeHosts: true}
			report, err := client.Search(context.Background(), SearchOptions{Query: "example", Source: SourceLAPRegistry})
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("err = %v, want %q", err, tt.wantError)
			}
			if len(report.Results) != 0 || len(report.Attempts) != 1 || report.Attempts[0].Status != "fail" {
				t.Fatalf("partial LAP report = %#v", report)
			}
		})
	}
}

func TestSearchAutoUsesLAPBeforePublicAPIs(t *testing.T) {
	publicCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guru.json":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/v1/search":
			w.Header().Set("Content-Type", "text/lap")
			_, _ = w.Write([]byte(`@search_query mail
@total 1
@result
@name mail-api
@description Mail API
@source_url https://example.com/openapi.yaml
@provider example-com | Example | example.com
@endresult
`))
		case "/entries":
			publicCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := (&Client{
		APIsGuruListURL:  server.URL + "/guru.json",
		LAPSearchURL:     server.URL + "/v1/search",
		PublicAPIsURL:    server.URL + "/entries",
		AllowUnsafeHosts: true,
	}).Search(context.Background(), SearchOptions{Query: "mail", Source: SourceAuto})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].Source != string(SourceLAPRegistry) {
		t.Fatalf("results = %#v, want LAP fallback", report.Results)
	}
	if publicCalled {
		t.Fatal("public-apis fallback was called after LAP returned a result")
	}
}

func TestSearchAutoUsesRFC9727WhenGlobalSourcesMiss(t *testing.T) {
	publicCalled := false
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guru.json":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/v1/search":
			w.Header().Set("Content-Type", "text/lap")
			_, _ = w.Write([]byte("@search_query mail\n@total 0\n"))
		case "/.well-known/api-catalog":
			w.Header().Set("Content-Type", "application/linkset+json")
			_ = json.NewEncoder(w).Encode(map[string]any{"linkset": []any{map[string]any{
				"anchor": baseURL + "/apis/mail",
				"service-desc": []any{map[string]any{
					"href": baseURL + "/mail.yaml",
					"type": "application/yaml",
				}},
			}}})
		case "/entries":
			publicCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{
		APIsGuruListURL:  server.URL + "/guru.json",
		LAPSearchURL:     server.URL + "/v1/search",
		PublicAPIsURL:    server.URL + "/entries",
		AllowUnsafeHosts: true,
	}).Search(context.Background(), SearchOptions{
		Query:       "mail",
		Source:      SourceAuto,
		ProviderURL: baseURL + "/docs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].Source != string(SourceRFC9727) {
		t.Fatalf("results = %#v, want RFC 9727 fallback", report.Results)
	}
	if publicCalled {
		t.Fatal("public-apis fallback was called after RFC 9727 returned a result")
	}
}

func TestSearchRFC9727FindsServiceDescriptions(t *testing.T) {
	requestCount := 0
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/.well-known/api-catalog" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Accept"); got != "application/linkset+json" {
			t.Fatalf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", `application/linkset+json; profile="https://www.rfc-editor.org/info/rfc9727"`)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"linkset": []any{
				map[string]any{
					"anchor": baseURL + "/apis/billing",
					"service-desc": []any{
						map[string]any{"href": "/specs/billing.yaml", "type": "application/yaml"},
						map[string]any{"href": "/specs/billing.n3", "type": "text/n3"},
					},
				},
				map[string]any{
					"anchor":      baseURL + "/.well-known/api-catalog",
					"api-catalog": []any{map[string]any{"href": baseURL + "/nested-catalog"}},
				},
			},
		})
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{AllowUnsafeHosts: true}).Search(context.Background(), SearchOptions{
		Query:       "billing",
		Limit:       3,
		Source:      SourceRFC9727,
		ProviderURL: server.URL + "/developer/docs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestCount != 1 {
		t.Fatalf("requests = %d, want exactly one non-recursive catalog fetch", requestCount)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want 1 OpenAPI-like service description: %#v", len(report.Results), report.Results)
	}
	got := report.Results[0]
	if got.Source != string(SourceRFC9727) || got.SpecURL != server.URL+"/specs/billing.yaml" {
		t.Fatalf("unexpected RFC 9727 result: %#v", got)
	}
	if got.Title != "billing" || got.MediaType != "application/yaml" || got.Validated || !got.Experimental {
		t.Fatalf("unexpected RFC 9727 metadata: %#v", got)
	}
	if got.LandingURL != server.URL+"/apis/billing" || !strings.Contains(got.Provenance, "/.well-known/api-catalog") {
		t.Fatalf("unexpected RFC 9727 provenance: %#v", got)
	}
}

func TestSearchRFC9727RequiresProviderURL(t *testing.T) {
	_, err := (&Client{}).Search(context.Background(), SearchOptions{
		Query:  "billing",
		Source: SourceRFC9727,
	})
	if err == nil || !strings.Contains(err.Error(), "provider URL") {
		t.Fatalf("err = %v, want provider URL requirement", err)
	}
}

func TestRFC9727CatalogURLNormalizesProvider(t *testing.T) {
	got, err := rfc9727CatalogURL("api.example.com/developer?format=html#docs")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.example.com/.well-known/api-catalog" {
		t.Fatalf("catalog URL = %q", got)
	}
	if _, err := rfc9727CatalogURL("https://user@example.com"); err == nil || !strings.Contains(err.Error(), "user information") {
		t.Fatalf("userinfo err = %v", err)
	}
}

func TestParseRFC9727DocumentRejectsNonLinksetMembers(t *testing.T) {
	_, err := parseRFC9727Document([]byte(`{"linkset":[],"partial":true}`))
	if err == nil || !strings.Contains(err.Error(), "sole top-level member") {
		t.Fatalf("err = %v, want sole-member rejection", err)
	}
	if _, err := parseRFC9727Targets(json.RawMessage(`{"href":"https://example.com/openapi.yaml"}`)); err == nil || !strings.Contains(err.Error(), "must be an array") {
		t.Fatalf("target err = %v, want array rejection", err)
	}
}

func TestSearchRFC9727ResolvesRelativeHrefAgainstCatalog(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/linkset+json")
		_ = json.NewEncoder(w).Encode(map[string]any{"linkset": []any{map[string]any{
			"anchor": baseURL + "/apis/mail/",
			"service-desc": []any{map[string]any{
				"href": "specs/mail.yaml",
				"type": "application/yaml",
			}},
		}}})
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{AllowUnsafeHosts: true}).Search(context.Background(), SearchOptions{
		Query:       "mail",
		Source:      SourceRFC9727,
		ProviderURL: baseURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].SpecURL != baseURL+"/.well-known/specs/mail.yaml" {
		t.Fatalf("relative href result = %#v", report.Results)
	}
}

func TestSearchRFC9727UsesRedirectedCatalogAsRelativeBase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/api-catalog":
			http.Redirect(w, r, "/catalogs/current.json", http.StatusFound)
		case "/catalogs/current.json":
			w.Header().Set("Content-Type", "application/linkset+json")
			_, _ = w.Write([]byte(`{"linkset":[{"service-desc":[{"href":"mail.yaml","type":"application/yaml"}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	report, err := (&Client{AllowUnsafeHosts: true}).Search(context.Background(), SearchOptions{
		Query:       "mail",
		Source:      SourceRFC9727,
		ProviderURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].SpecURL != server.URL+"/catalogs/mail.yaml" {
		t.Fatalf("redirect-relative result = %#v", report.Results)
	}
	if !strings.Contains(report.Results[0].Provenance, server.URL+"/catalogs/current.json") {
		t.Fatalf("redirect provenance = %q", report.Results[0].Provenance)
	}
}

func TestSearchRFC9727RejectsWrongMediaType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"linkset":[]}`))
	}))
	defer server.Close()

	_, err := (&Client{AllowUnsafeHosts: true}).Search(context.Background(), SearchOptions{
		Query:       "billing",
		Source:      SourceRFC9727,
		ProviderURL: server.URL,
	})
	if err == nil || !strings.Contains(err.Error(), "application/linkset+json") {
		t.Fatalf("err = %v, want media-type rejection", err)
	}
}

func TestSearchRFC9727RejectsUnsafeProviderByDefault(t *testing.T) {
	_, err := (&Client{}).Search(context.Background(), SearchOptions{
		Query:       "billing",
		Source:      SourceRFC9727,
		ProviderURL: "http://127.0.0.1:8080",
	})
	if err == nil || !strings.Contains(err.Error(), "refusing private") {
		t.Fatalf("err = %v, want unsafe-host rejection", err)
	}
}

func TestRFC9727RejectsUnsafeServiceDescriptionTarget(t *testing.T) {
	base, err := url.Parse("https://example.com/.well-known/api-catalog")
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&Client{}).resolveSafeRFC9727URL(context.Background(), base, "http://127.0.0.1/openapi.yaml")
	if err == nil || !strings.Contains(err.Error(), "refusing private") {
		t.Fatalf("err = %v, want unsafe service-desc rejection", err)
	}
}

func TestSearchRFC9727RejectsServiceDescriptionOverflow(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/linkset+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"linkset": []any{map[string]any{
				"anchor": baseURL + "/apis",
				"service-desc": []any{
					map[string]any{"href": baseURL + "/one.yaml", "type": "application/yaml"},
					map[string]any{"href": baseURL + "/two.yaml", "type": "application/yaml"},
				},
			}},
		})
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{AllowUnsafeHosts: true, RFC9727LinkLimit: 1}).Search(context.Background(), SearchOptions{
		Query:       "billing",
		Source:      SourceRFC9727,
		ProviderURL: baseURL,
	})
	if err == nil || !strings.Contains(err.Error(), "service-desc link limit") {
		t.Fatalf("err = %v, want service-desc link limit", err)
	}
	if len(report.Results) != 0 || len(report.Attempts) != 1 || report.Attempts[0].Status != "fail" {
		t.Fatalf("partial RFC 9727 report was not rejected: %#v", report)
	}
}

func TestSearchRFC9727EnforcesResponseLimitAndCancellation(t *testing.T) {
	t.Run("response limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/linkset+json")
			_, _ = w.Write([]byte(`{"linkset":[]}` + strings.Repeat(" ", 64)))
		}))
		defer server.Close()

		report, err := (&Client{AllowUnsafeHosts: true, MaxBytes: 32}).Search(context.Background(), SearchOptions{
			Query:       "mail",
			Source:      SourceRFC9727,
			ProviderURL: server.URL,
		})
		if err == nil || !strings.Contains(err.Error(), "larger than 32 bytes") {
			t.Fatalf("err = %v, want response-size rejection", err)
		}
		if len(report.Results) != 0 || len(report.Attempts) != 1 || report.Attempts[0].Status != "fail" {
			t.Fatalf("oversized RFC 9727 report = %#v", report)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			called = true
		}))
		defer server.Close()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := (&Client{AllowUnsafeHosts: true}).Search(ctx, SearchOptions{
			Query:       "mail",
			Source:      SourceRFC9727,
			ProviderURL: server.URL,
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context cancellation", err)
		}
		if called {
			t.Fatal("cancelled RFC 9727 lookup reached server")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(200 * time.Millisecond):
				w.Header().Set("Content-Type", "application/linkset+json")
				_, _ = w.Write([]byte(`{"linkset":[]}`))
			}
		}))
		defer server.Close()

		_, err := (&Client{AllowUnsafeHosts: true, Timeout: 10 * time.Millisecond}).Search(context.Background(), SearchOptions{
			Query:       "mail",
			Source:      SourceRFC9727,
			ProviderURL: server.URL,
		})
		if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
			t.Fatalf("err = %v, want timeout", err)
		}
	})
}
