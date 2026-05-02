package apitools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWellKnownURLsReplacesPath(t *testing.T) {
	c := &Client{WellKnownPaths: []string{"/openapi.json", "/swagger.json"}}
	got := c.wellKnownURLs("https://example.com/docs?v=1#frag")
	want := []string{"https://example.com/openapi.json", "https://example.com/swagger.json"}
	if len(got) != len(want) {
		t.Fatalf("got %#v", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWellKnownURLsRejectsInvalidBase(t *testing.T) {
	if got := (&Client{}).wellKnownURLs("not a url"); got != nil {
		t.Errorf("got %#v", got)
	}
	if got := (&Client{}).wellKnownURLs("/no-host"); got != nil {
		t.Errorf("got %#v", got)
	}
}

func TestWellKnownURLsUsesDefaultPathsWhenNotConfigured(t *testing.T) {
	got := (&Client{}).wellKnownURLs("https://example.com")
	if len(got) != len(WellKnownPaths()) {
		t.Errorf("got %d paths, want %d", len(got), len(WellKnownPaths()))
	}
}

func TestApisGuruAPIPreferredFallsBackToHighestKey(t *testing.T) {
	api := apisGuruAPI{
		Versions: map[string]apisGuruVersion{
			"1.0.0": {SwaggerURL: "u1"},
			"2.0.0": {SwaggerURL: "u2"},
		},
	}
	id, version := api.preferred()
	if id != "2.0.0" || version.SwaggerURL != "u2" {
		t.Errorf("preferred = %q %#v", id, version)
	}
}

func TestApisGuruAPIPreferredHonorsExplicitField(t *testing.T) {
	api := apisGuruAPI{
		Preferred: "1.0.0",
		Versions: map[string]apisGuruVersion{
			"1.0.0": {SwaggerURL: "older"},
			"2.0.0": {SwaggerURL: "newer"},
		},
	}
	id, version := api.preferred()
	if id != "1.0.0" || version.SwaggerURL != "older" {
		t.Errorf("preferred = %q %#v", id, version)
	}
}

func TestApisGuruAPIPreferredEmpty(t *testing.T) {
	id, version := apisGuruAPI{}.preferred()
	if id != "" || version.SwaggerURL != "" {
		t.Errorf("empty preferred = %q %#v", id, version)
	}
}

func TestSearchPublicAPIsTimesOutOnProbeBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/entries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entries": []map[string]any{{
					"API":         "Slow API",
					"Description": "always slow",
					"Link":        "http://" + r.Host + "/docs",
					"Category":    "test",
				}},
			})
		default:
			time.Sleep(200 * time.Millisecond)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := &Client{
		PublicAPIsURL:     server.URL + "/entries",
		AllowUnsafeHosts:  true,
		ProbeTimeout:      30 * time.Millisecond,
		PublicProbeBudget: 50 * time.Millisecond,
	}
	report, err := c.Search(context.Background(), SearchOptions{
		Query:  "slow",
		Limit:  3,
		Source: SourcePublicAPIs,
	})
	if err == nil || !errors.Is(err, ErrProbeBudgetExceeded) {
		t.Fatalf("expected probe-budget exceeded, got %v", err)
	}
	if len(report.Attempts) == 0 {
		t.Fatalf("expected attempts to be recorded")
	}
}

func TestSearchSurfacesAuthoritativeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &Client{
		APIsGuruListURL:  server.URL + "/list.json",
		PublicAPIsURL:    server.URL + "/entries",
		AllowUnsafeHosts: true,
	}
	_, err := c.Search(context.Background(), SearchOptions{Query: "anything", Source: SourceAPIsGuru})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 surfaced, got %v", err)
	}
}
