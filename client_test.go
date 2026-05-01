package apitools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearchAPIsGuruRanksMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"slack.com": map[string]any{
				"preferred": "1.0.0",
				"versions": map[string]any{
					"1.0.0": map[string]any{
						"swaggerUrl": "https://example.com/slack.json",
						"info": map[string]any{
							"title":                 "Slack Web API",
							"description":           "Post and search messages.",
							"x-apisguru-categories": []string{"collaboration"},
						},
					},
				},
			},
			"weather.example": map[string]any{
				"preferred": "1.0.0",
				"versions": map[string]any{
					"1.0.0": map[string]any{
						"swaggerUrl": "https://example.com/weather.json",
						"info":       map[string]any{"title": "Weather API"},
					},
				},
			},
		})
	}))
	defer server.Close()

	report, err := (&Client{APIsGuruListURL: server.URL + "/list.json", AllowUnsafeHosts: true}).Search(context.Background(), SearchOptions{
		Query:  "slack message",
		Limit:  5,
		Source: SourceAPIsGuru,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(report.Results), report.Results)
	}
	if report.Results[0].Provider != "slack.com" || report.Results[0].SpecURL != "https://example.com/slack.json" {
		t.Fatalf("unexpected result: %#v", report.Results[0])
	}
}

func TestAutoFallsBackToPublicAPIs(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/guru.json":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/entries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entries": []map[string]any{{
					"API":         "Example Mail",
					"Description": "Send transactional mail",
					"Link":        baseURL + "/docs",
					"Category":    "Communication",
				}},
			})
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Example Mail","version":"1.0.0"},"paths":{"/send":{"post":{"operationId":"sendMail","responses":{"200":{"description":"ok"}}}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{
		APIsGuruListURL:  server.URL + "/guru.json",
		PublicAPIsURL:    server.URL + "/entries",
		AllowUnsafeHosts: true,
	}).Search(context.Background(), SearchOptions{
		Query:  "mail",
		Limit:  3,
		Source: SourceAuto,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(report.Results), report.Results)
	}
	got := report.Results[0]
	if got.Source != string(SourcePublicAPIs) || !got.Validated || got.SpecURL != server.URL+"/openapi.json" {
		t.Fatalf("unexpected fallback result: %#v", got)
	}
}

func TestSearchPublicAPIsFindsValidatedSpec(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entries": []map[string]any{{
					"API":         "Example Mail",
					"Description": "Send transactional mail",
					"Link":        baseURL + "/docs",
					"Category":    "Communication",
				}},
			})
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Example Mail","version":"1.0.0"},"paths":{"/send":{"post":{"responses":{"200":{"description":"ok"}}}}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	report, err := (&Client{
		PublicAPIsURL:    server.URL + "/entries",
		AllowUnsafeHosts: true,
	}).Search(context.Background(), SearchOptions{
		Query:  "mail",
		Limit:  3,
		Source: SourcePublicAPIs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(report.Results), report.Results)
	}
	got := report.Results[0]
	if got.Source != string(SourcePublicAPIs) || !got.Validated || got.Title != "Example Mail" || got.SpecURL != server.URL+"/openapi.json" {
		t.Fatalf("unexpected public-apis result: %#v", got)
	}
}

func TestImportValidatesAndWritesUniqueFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: Support API
  version: 1.0.0
paths:
  /tickets:
    get:
      responses:
        "200":
          description: ok
`))
	}))
	defer server.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "support.yaml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	imported, err := (&Client{AllowUnsafeHosts: true}).Import(context.Background(), ImportOptions{
		URL:  server.URL + "/openapi.yaml",
		Dir:  dir,
		Name: "support",
	})
	if err != nil {
		t.Fatal(err)
	}
	if imported.Name != "support-2.yaml" {
		t.Fatalf("name = %q, want support-2.yaml", imported.Name)
	}
	if imported.Title != "Support API" {
		t.Fatalf("title = %q", imported.Title)
	}
	old, err := os.ReadFile(filepath.Join(dir, "support.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old" {
		t.Fatalf("existing file was overwritten: %q", old)
	}
}

func TestRejectsPrivateURLByDefault(t *testing.T) {
	_, err := (&Client{}).ValidateURL(context.Background(), "http://127.0.0.1/openapi.json")
	if err == nil || !strings.Contains(err.Error(), "refusing private") {
		t.Fatalf("expected private URL rejection, got %v", err)
	}
}

func TestInvalidSpecRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not":"openapi"}`))
	}))
	defer server.Close()
	_, err := (&Client{AllowUnsafeHosts: true}).ValidateURL(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "does not look like OpenAPI") {
		t.Fatalf("expected invalid OpenAPI rejection, got %v", err)
	}
}

func TestSpecValidationCases(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "valid openapi 3",
			body: `openapi: 3.0.0
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200":
          description: ok
`,
		},
		{
			name: "valid openapi 3.1",
			body: `{
  "openapi": "3.1.0",
  "info": {
    "title": "Mail",
    "version": "1.0.0"
  },
  "paths": {
    "/send": {
      "post": {
        "responses": {
          "200": {
            "description": "ok"
          }
        }
      }
    }
  }
}`,
		},
		{
			name: "valid swagger 2",
			body: `swagger: "2.0"
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200":
          description: ok
`,
		},
		{
			name: "missing info",
			body: `openapi: 3.0.0
paths: {}
`,
			wantErr: true,
		},
		{
			name: "bad version",
			body: `openapi: 9.0.0
info:
  title: Mail
  version: 1.0.0
paths: {}
`,
			wantErr: true,
		},
		{
			name: "malformed paths",
			body: `openapi: 3.0.0
info:
  title: Mail
  version: 1.0.0
paths: []
`,
			wantErr: true,
		},
		{
			name: "openapi 3 response needs description",
			body: `openapi: 3.0.0
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200": {}
`,
			wantErr: true,
		},
		{
			name: "external ref",
			body: `openapi: 3.0.0
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200":
          $ref: https://example.com/response.yaml
`,
			wantErr: true,
		},
		{
			name: "swagger 2 path must start slash",
			body: `swagger: "2.0"
info:
  title: Mail
  version: 1.0.0
paths:
  send:
    post:
      responses:
        "200":
          description: ok
`,
			wantErr: true,
		},
		{
			name: "swagger 2 rejects unknown method",
			body: `swagger: "2.0"
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    fetch:
      responses:
        "200":
          description: ok
`,
			wantErr: true,
		},
		{
			name: "swagger 2 response needs description",
			body: `swagger: "2.0"
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200": {}
`,
			wantErr: true,
		},
		{
			name: "swagger 2 rejects external ref",
			body: `swagger: "2.0"
info:
  title: Mail
  version: 1.0.0
paths:
  /send:
    post:
      responses:
        "200":
          $ref: https://example.com/response.yaml
`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			_, err := (&Client{AllowUnsafeHosts: true}).ValidateURL(context.Background(), server.URL)
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestPublicAPIsProbeBudgetStopsSlowProbes(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entries": []map[string]any{{
					"API":         "Slow Mail",
					"Description": "Send mail",
					"Link":        baseURL + "/docs",
				}},
			})
		default:
			select {
			case <-r.Context().Done():
			case <-time.After(200 * time.Millisecond):
			}
		}
	}))
	defer server.Close()
	baseURL = server.URL

	start := time.Now()
	report, err := (&Client{
		PublicAPIsURL:     server.URL + "/entries",
		AllowUnsafeHosts:  true,
		WellKnownPaths:    []string{"/slow-1", "/slow-2", "/slow-3"},
		ProbeTimeout:      20 * time.Millisecond,
		PublicProbeBudget: 35 * time.Millisecond,
	}).Search(context.Background(), SearchOptions{
		Query:  "mail",
		Source: SourcePublicAPIs,
		Limit:  1,
	})
	if !errors.Is(err, ErrProbeBudgetExceeded) {
		t.Fatalf("err = %v, want ErrProbeBudgetExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("probe budget took too long: %s", elapsed)
	}
	foundBudgetAttempt := false
	for _, attempt := range report.Attempts {
		if strings.Contains(attempt.Detail, "probe budget") {
			foundBudgetAttempt = true
		}
	}
	if !foundBudgetAttempt {
		t.Fatalf("missing budget attempt: %#v", report.Attempts)
	}
}

func TestPartialPublicAPIsProbeBudgetDoesNotCacheSearch(t *testing.T) {
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entries":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entries": []map[string]any{{
					"API":         "Slow Mail",
					"Description": "Send mail",
					"Link":        baseURL + "/docs",
				}},
			})
		case "/openapi.json":
			_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Slow Mail","version":"1.0.0"},"paths":{}}`))
		default:
			select {
			case <-r.Context().Done():
			case <-time.After(200 * time.Millisecond):
			}
		}
	}))
	defer server.Close()
	baseURL = server.URL

	storeSearchCount := 0
	cache := fakeCache{
		loadSpec: func(context.Context, string, time.Duration) (CachedSpec, bool, error) {
			return CachedSpec{}, false, nil
		},
		storeSearch: func(context.Context, SearchCacheKey, SearchReport) error {
			storeSearchCount++
			return nil
		},
	}
	report, err := (&Client{
		PublicAPIsURL:     server.URL + "/entries",
		AllowUnsafeHosts:  true,
		WellKnownPaths:    []string{"/openapi.json", "/slow-1", "/slow-2"},
		ProbeTimeout:      10 * time.Millisecond,
		PublicProbeBudget: 15 * time.Millisecond,
		Cache:             cache,
	}).Search(context.Background(), SearchOptions{
		Query:  "mail",
		Source: SourcePublicAPIs,
		Limit:  2,
	})
	if !errors.Is(err, ErrProbeBudgetExceeded) {
		t.Fatalf("err = %v, want ErrProbeBudgetExceeded", err)
	}
	if len(report.Results) != 1 {
		t.Fatalf("len = %d, want partial result: %#v", len(report.Results), report.Results)
	}
	if storeSearchCount != 0 {
		t.Fatalf("StoreSearch called %d times, want 0", storeSearchCount)
	}
}

func TestInvalidSourceRejectedBeforeOfflineCacheHandling(t *testing.T) {
	_, err := (&Client{}).Search(context.Background(), SearchOptions{
		Query:     "mail",
		Source:    Source("bad"),
		CacheMode: CacheModeOffline,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("expected unknown source error, got %v", err)
	}
}

func TestRedirectToPrivateHostRejected(t *testing.T) {
	client, err := (&Client{}).redirectSafeClient()
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/openapi.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = client.CheckRedirect(req, nil)
	if err == nil || !strings.Contains(err.Error(), "refusing private") {
		t.Fatalf("expected private redirect rejection, got %v", err)
	}
}

func TestCustomTransportRejectedInSafeMode(t *testing.T) {
	_, err := (&Client{
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("should not be called")
		})},
	}).ValidateURL(context.Background(), "http://93.184.216.34/openapi.yaml")
	if err == nil || !strings.Contains(err.Error(), "custom HTTP transport") {
		t.Fatalf("expected custom transport rejection, got %v", err)
	}
}

func TestMaxResponseSizeEnforced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 20)))
	}))
	defer server.Close()
	_, err := (&Client{AllowUnsafeHosts: true, MaxBytes: 10}).ValidateURL(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "larger than 10") {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestCachedSpecRevalidatedBeforeImport(t *testing.T) {
	dir := t.TempDir()
	rawURL := "http://93.184.216.34/openapi.yaml"
	cache := fakeCache{spec: CachedSpec{
		OriginalURL: rawURL,
		FinalURL:    rawURL,
		Content:     []byte(`{"not":"openapi"}`),
	}}
	_, err := (&Client{Cache: cache}).Import(context.Background(), ImportOptions{
		URL:       rawURL,
		Dir:       dir,
		CacheMode: CacheModeOffline,
	})
	if err == nil || !strings.Contains(err.Error(), "cached OpenAPI document") {
		t.Fatalf("expected cached validation error, got %v", err)
	}
}

func TestOfflineCachedPrivateURLRejectedByDefault(t *testing.T) {
	rawURL := "http://127.0.0.1/openapi.yaml"
	cache := fakeCache{
		loadSpec: func(context.Context, string, time.Duration) (CachedSpec, bool, error) {
			t.Fatalf("LoadSpec should not be called for unsafe URL")
			return CachedSpec{}, false, nil
		},
	}
	_, err := (&Client{Cache: cache}).Import(context.Background(), ImportOptions{
		URL:       rawURL,
		Dir:       t.TempDir(),
		CacheMode: CacheModeOffline,
	})
	if err == nil || !strings.Contains(err.Error(), "refusing private") {
		t.Fatalf("expected private URL rejection, got %v", err)
	}
}

func TestOfflineCachedPrivateURLAllowedWhenUnsafeHostsAllowed(t *testing.T) {
	rawURL := "http://127.0.0.1/openapi.yaml"
	_, err := (&Client{AllowUnsafeHosts: true, Cache: validSpecCache(rawURL)}).Import(context.Background(), ImportOptions{
		URL:       rawURL,
		Dir:       t.TempDir(),
		CacheMode: CacheModeOffline,
	})
	if err != nil {
		t.Fatalf("expected cached private URL import with AllowUnsafeHosts, got %v", err)
	}
}

func TestReadWriteCorruptCachedSpecRefreshes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: Fresh
  version: 1.0.0
paths: {}
`))
	}))
	defer server.Close()
	rawURL := server.URL + "/openapi.yaml"
	storeSpecCount := 0
	cache := fakeCache{
		loadSpec: func(context.Context, string, time.Duration) (CachedSpec, bool, error) {
			return CachedSpec{
				OriginalURL: rawURL,
				FinalURL:    rawURL,
				Content:     validOpenAPI3YAML(),
				SHA256:      "bad",
			}, true, nil
		},
		storeSpec: func(context.Context, CachedSpec) error {
			storeSpecCount++
			return nil
		},
	}
	metadata, err := (&Client{AllowUnsafeHosts: true, Cache: cache}).ValidateURL(context.Background(), rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Fresh" {
		t.Fatalf("title = %q, want Fresh", metadata.Title)
	}
	if storeSpecCount != 1 {
		t.Fatalf("StoreSpec count = %d, want 1", storeSpecCount)
	}
}

func TestOfflineCorruptCachedSpecFails(t *testing.T) {
	rawURL := "https://example.com/openapi.yaml"
	cache := fakeCache{
		loadSpec: func(context.Context, string, time.Duration) (CachedSpec, bool, error) {
			return CachedSpec{
				OriginalURL: rawURL,
				FinalURL:    rawURL,
				Content:     validOpenAPI3YAML(),
				SHA256:      "bad",
			}, true, nil
		},
	}
	_, err := (&Client{AllowUnsafeHosts: true, Cache: cache}).Import(context.Background(), ImportOptions{
		URL:       rawURL,
		Dir:       t.TempDir(),
		CacheMode: CacheModeOffline,
	})
	if !errors.Is(err, ErrCachedSpecIntegrity) {
		t.Fatalf("err = %v, want ErrCachedSpecIntegrity", err)
	}
}

func TestNilClientUsesDefaultBehavior(t *testing.T) {
	var nilClient *Client
	defaultClient := &Client{}
	_, nilErr := nilClient.ValidateURL(context.Background(), "http://127.0.0.1/openapi.json")
	_, defaultErr := defaultClient.ValidateURL(context.Background(), "http://127.0.0.1/openapi.json")
	if (nilErr == nil) != (defaultErr == nil) {
		t.Fatalf("nil err = %v, default err = %v", nilErr, defaultErr)
	}
	_, nilErr = nilClient.Search(context.Background(), SearchOptions{Query: "mail", CacheMode: CacheModeOffline})
	_, defaultErr = defaultClient.Search(context.Background(), SearchOptions{Query: "mail", CacheMode: CacheModeOffline})
	if (nilErr == nil) != (defaultErr == nil) {
		t.Fatalf("nil search err = %v, default search err = %v", nilErr, defaultErr)
	}
	_, nilErr = nilClient.Import(context.Background(), ImportOptions{URL: "http://127.0.0.1/openapi.yaml", Dir: t.TempDir(), CacheMode: CacheModeOffline})
	_, defaultErr = defaultClient.Import(context.Background(), ImportOptions{URL: "http://127.0.0.1/openapi.yaml", Dir: t.TempDir(), CacheMode: CacheModeOffline})
	if (nilErr == nil) != (defaultErr == nil) {
		t.Fatalf("nil import err = %v, default import err = %v", nilErr, defaultErr)
	}
}

func TestDefaultWellKnownPathsMutationDoesNotAffectClientDefaults(t *testing.T) {
	original := append([]string(nil), DefaultWellKnownPaths...)
	defer func() {
		DefaultWellKnownPaths = original
	}()
	DefaultWellKnownPaths = []string{"/mutated"}
	got := (&Client{}).wellKnownURLs("https://example.com/docs")
	for _, candidate := range got {
		if strings.HasSuffix(candidate, "/mutated") {
			t.Fatalf("default paths used mutated exported variable: %#v", got)
		}
	}
}

func validSpecCache(rawURL string) fakeCache {
	return fakeCache{
		loadSpec: func(context.Context, string, time.Duration) (CachedSpec, bool, error) {
			return CachedSpec{
				OriginalURL: rawURL,
				FinalURL:    rawURL,
				Content:     validOpenAPI3YAML(),
			}, true, nil
		},
	}
}

func validOpenAPI3YAML() []byte {
	return []byte(`openapi: 3.0.0
info:
  title: Mail
  version: 1.0.0
paths: {}
`)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type fakeCache struct {
	spec        CachedSpec
	loadSpec    func(context.Context, string, time.Duration) (CachedSpec, bool, error)
	storeSpec   func(context.Context, CachedSpec) error
	storeSearch func(context.Context, SearchCacheKey, SearchReport) error
}

func (f fakeCache) LoadSearch(context.Context, SearchCacheKey, time.Duration) (SearchReport, bool, error) {
	return SearchReport{}, false, nil
}

func (f fakeCache) StoreSearch(ctx context.Context, key SearchCacheKey, report SearchReport) error {
	if f.storeSearch != nil {
		return f.storeSearch(ctx, key, report)
	}
	return nil
}

func (f fakeCache) LoadSpec(ctx context.Context, rawURL string, maxAge time.Duration) (CachedSpec, bool, error) {
	if f.loadSpec != nil {
		return f.loadSpec(ctx, rawURL, maxAge)
	}
	return f.spec, true, nil
}

func (f fakeCache) StoreSpec(ctx context.Context, spec CachedSpec) error {
	if f.storeSpec != nil {
		return f.storeSpec(ctx, spec)
	}
	return nil
}
