package apitools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultAPIsGuruListURL   = "https://api.apis.guru/v2/list.json"
	DefaultLAPSearchURL      = "https://registry.lap.sh/v1/search"
	DefaultPublicAPIsURL     = "https://api.publicapis.org/entries"
	DefaultAPICatalogPath    = "/.well-known/api-catalog"
	DefaultRFC9727LinkLimit  = 100
	DefaultTimeout           = 30 * time.Second
	DefaultMaxBytes          = 20 * 1024 * 1024
	DefaultCacheMaxAge       = 24 * time.Hour
	DefaultProbeTimeout      = 5 * time.Second
	DefaultPublicProbeBudget = 30 * time.Second
)

var defaultWellKnownPaths = [...]string{
	"/openapi.json",
	"/openapi.yaml",
	"/swagger.json",
	"/swagger.yaml",
	"/v3/api-docs",
	"/api-docs",
	"/.well-known/openapi",
}

// DefaultWellKnownPaths is kept for source compatibility. Prefer setting
// Client.WellKnownPaths or calling WellKnownPaths; clients copy defaults before
// use so caller mutation of this variable does not affect discovery behavior.
var DefaultWellKnownPaths = WellKnownPaths()

// WellKnownPaths returns a fresh copy of the default public-apis probe paths.
func WellKnownPaths() []string {
	paths := make([]string, len(defaultWellKnownPaths))
	copy(paths, defaultWellKnownPaths[:])
	return paths
}

var (
	ErrProbeBudgetExceeded = errors.New("public-apis probe budget exceeded")
	ErrCachedSpecIntegrity = errors.New("cached OpenAPI document failed integrity validation")
)

type Source string

const (
	SourceAuto        Source = "auto"
	SourceAPIsGuru    Source = "apis-guru"
	SourceLAPRegistry Source = "lap-registry"
	SourceRFC9727     Source = "rfc9727"
	SourcePublicAPIs  Source = "public-apis"
)

type CacheMode string

const (
	CacheModeReadWrite CacheMode = "read-write"
	CacheModeRefresh   CacheMode = "refresh"
	CacheModeOffline   CacheMode = "offline"
	CacheModeBypass    CacheMode = "bypass"
)

type Client struct {
	HTTPClient      *http.Client
	APIsGuruListURL string
	LAPSearchURL    string
	PublicAPIsURL   string
	Timeout         time.Duration
	MaxBytes        int64
	// AllowedPorts adds reviewed destination ports to the default HTTP port 80
	// and HTTPS port 443 policy. It does not relax hostname or IP checks.
	AllowedPorts      []int
	WellKnownPaths    []string
	AllowUnsafeHosts  bool
	Cache             Cache
	ProbeTimeout      time.Duration
	PublicProbeBudget time.Duration
	RFC9727LinkLimit  int
}

type SearchOptions struct {
	Query       string
	Limit       int
	Source      Source
	PublicProbe int
	CacheMode   CacheMode
	CacheMaxAge time.Duration
	// ProviderURL enables provider-scoped RFC 9727 discovery. A bare hostname
	// defaults to HTTPS; any supplied path is replaced by /.well-known/api-catalog.
	ProviderURL string
}

type SearchReport struct {
	Query    string          `json:"query"`
	Source   Source          `json:"source"`
	Results  []Result        `json:"results"`
	Attempts []SearchAttempt `json:"attempts,omitempty"`
}

type SearchAttempt struct {
	Source string `json:"source"`
	URL    string `json:"url,omitempty"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type Result struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Provider     string   `json:"provider,omitempty"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Version      string   `json:"version,omitempty"`
	Categories   []string `json:"categories,omitempty"`
	SpecURL      string   `json:"spec_url"`
	LandingURL   string   `json:"landing_url,omitempty"`
	Score        int      `json:"score"`
	Validated    bool     `json:"validated"`
	Provenance   string   `json:"provenance"`
	MediaType    string   `json:"media_type,omitempty"`
	Experimental bool     `json:"experimental,omitempty"`
}

type ImportOptions struct {
	URL         string
	Dir         string
	Name        string
	CacheMode   CacheMode
	CacheMaxAge time.Duration
}

type ImportedSpec struct {
	Name        string       `json:"name"`
	Path        string       `json:"path"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url"`
	SHA256      string       `json:"sha256"`
	Bytes       int64        `json:"bytes"`
	Metadata    SpecMetadata `json:"metadata"`
}

type SpecMetadata struct {
	Title          string `json:"title,omitempty"`
	Description    string `json:"description,omitempty"`
	OpenAPI        string `json:"openapi,omitempty"`
	Swagger        string `json:"swagger,omitempty"`
	OperationCount int    `json:"operation_count,omitempty"`
}

type SearchCacheKey struct {
	Query       string `json:"query"`
	Source      Source `json:"source"`
	Limit       int    `json:"limit"`
	PublicProbe int    `json:"public_probe,omitempty"`
	ProviderURL string `json:"provider_url,omitempty"`
}

type CachedSpec struct {
	OriginalURL string       `json:"original_url"`
	FinalURL    string       `json:"final_url"`
	ContentPath string       `json:"content_path,omitempty"`
	Content     []byte       `json:"-"`
	SHA256      string       `json:"sha256"`
	Bytes       int64        `json:"bytes"`
	Metadata    SpecMetadata `json:"metadata"`
	StoredAt    time.Time    `json:"stored_at,omitempty"`
}

type Cache interface {
	LoadSearch(ctx context.Context, key SearchCacheKey, maxAge time.Duration) (SearchReport, bool, error)
	StoreSearch(ctx context.Context, key SearchCacheKey, report SearchReport) error
	LoadSpec(ctx context.Context, rawURL string, maxAge time.Duration) (CachedSpec, bool, error)
	StoreSpec(ctx context.Context, spec CachedSpec) error
}

func (c *Client) Search(ctx context.Context, opts SearchOptions) (SearchReport, error) {
	c = c.effective()
	query := strings.TrimSpace(opts.Query)
	if len(query) < 2 {
		return SearchReport{}, fmt.Errorf("query must be at least two characters")
	}
	source := opts.Source
	if source == "" {
		source = SourceAuto
	}
	if err := validateSource(source); err != nil {
		return SearchReport{}, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	mode, err := normalizeCacheMode(opts.CacheMode)
	if err != nil {
		return SearchReport{}, err
	}
	maxAge := normalizeCacheMaxAge(opts.CacheMaxAge)
	providerURL := strings.TrimSpace(opts.ProviderURL)
	key := SearchCacheKey{Query: query, Source: source, Limit: limit, PublicProbe: opts.PublicProbe, ProviderURL: providerURL}
	if c.Cache != nil && mode != CacheModeRefresh && mode != CacheModeBypass {
		report, ok, err := c.Cache.LoadSearch(ctx, key, maxAge)
		if err != nil {
			return SearchReport{}, err
		}
		if ok {
			report.Attempts = append([]SearchAttempt{{Source: "cache", Status: "pass", Detail: "search results loaded from SQLite cache"}}, report.Attempts...)
			return report, nil
		}
	}
	if mode == CacheModeOffline {
		if c.Cache == nil {
			return SearchReport{}, fmt.Errorf("cache is required for offline search")
		}
		return SearchReport{
			Query:    query,
			Source:   source,
			Attempts: []SearchAttempt{{Source: "cache", Status: "miss", Detail: "no cached search results"}},
		}, nil
	}
	report := SearchReport{Query: query, Source: source}
	switch source {
	case SourceAPIsGuru:
		results, attempts, err := c.searchAPIsGuru(ctx, query, limit)
		report.Results = results
		report.Attempts = append(report.Attempts, attempts...)
		return c.storeSearch(ctx, key, report, mode, err)
	case SourceLAPRegistry:
		results, attempts, err := c.searchLAPRegistry(ctx, query, limit)
		report.Results = results
		report.Attempts = append(report.Attempts, attempts...)
		return c.storeSearch(ctx, key, report, mode, err)
	case SourceRFC9727:
		results, attempts, err := c.searchRFC9727(ctx, query, providerURL, limit)
		report.Results = results
		report.Attempts = append(report.Attempts, attempts...)
		return c.storeSearch(ctx, key, report, mode, err)
	case SourcePublicAPIs:
		results, attempts, err := c.searchPublicAPIs(ctx, query, limit, opts.PublicProbe, mode, maxAge)
		report.Results = results
		report.Attempts = append(report.Attempts, attempts...)
		return c.storeSearch(ctx, key, report, mode, err)
	case SourceAuto:
		results, attempts, err := c.searchAPIsGuru(ctx, query, limit)
		report.Attempts = append(report.Attempts, attempts...)
		if err == nil && len(results) > 0 {
			report.Results = results
			return c.storeSearch(ctx, key, report, mode, nil)
		}
		var fallbackErrors []string
		if err != nil {
			fallbackErrors = append(fallbackErrors, "apis-guru: "+err.Error())
		}
		results, attempts, lapErr := c.searchLAPRegistry(ctx, query, limit)
		report.Attempts = append(report.Attempts, attempts...)
		if lapErr == nil && len(results) > 0 {
			report.Results = results
			return c.storeSearch(ctx, key, report, mode, nil)
		}
		if lapErr != nil {
			fallbackErrors = append(fallbackErrors, "lap-registry: "+lapErr.Error())
		}
		if providerURL != "" {
			results, attempts, catalogErr := c.searchRFC9727(ctx, query, providerURL, limit)
			report.Attempts = append(report.Attempts, attempts...)
			if catalogErr == nil && len(results) > 0 {
				report.Results = results
				return c.storeSearch(ctx, key, report, mode, nil)
			}
			if catalogErr != nil {
				fallbackErrors = append(fallbackErrors, "rfc9727: "+catalogErr.Error())
			}
		}
		results, attempts, publicErr := c.searchPublicAPIs(ctx, query, limit, opts.PublicProbe, mode, maxAge)
		report.Attempts = append(report.Attempts, attempts...)
		report.Results = results
		if publicErr != nil {
			fallbackErrors = append(fallbackErrors, "public-apis: "+publicErr.Error())
			return c.storeSearch(ctx, key, report, mode, errors.New(strings.Join(fallbackErrors, "; ")))
		}
		return c.storeSearch(ctx, key, report, mode, nil)
	default:
		return SearchReport{}, fmt.Errorf("unknown source %q", source)
	}
}

func (c *Client) ValidateURL(ctx context.Context, rawURL string) (SpecMetadata, error) {
	c = c.effective()
	_, _, metadata, err := c.downloadSpecWithCache(ctx, rawURL, CacheModeReadWrite, DefaultCacheMaxAge)
	return metadata, err
}

func (c *Client) effective() *Client {
	if c == nil {
		return &Client{}
	}
	return c
}

func validateSource(source Source) error {
	switch source {
	case SourceAuto, SourceAPIsGuru, SourceLAPRegistry, SourceRFC9727, SourcePublicAPIs:
		return nil
	default:
		return fmt.Errorf("unknown source %q", source)
	}
}

func (c *Client) storeSearch(ctx context.Context, key SearchCacheKey, report SearchReport, mode CacheMode, searchErr error) (SearchReport, error) {
	c = c.effective()
	if searchErr == nil && c.Cache != nil && mode != CacheModeBypass {
		if err := c.Cache.StoreSearch(ctx, key, report); err != nil {
			return report, err
		}
	}
	return report, searchErr
}

func normalizeCacheMode(mode CacheMode) (CacheMode, error) {
	if mode == "" {
		return CacheModeReadWrite, nil
	}
	switch mode {
	case CacheModeReadWrite, CacheModeRefresh, CacheModeOffline, CacheModeBypass:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown cache mode %q", mode)
	}
}

func normalizeCacheMaxAge(maxAge time.Duration) time.Duration {
	if maxAge <= 0 {
		return DefaultCacheMaxAge
	}
	return maxAge
}
