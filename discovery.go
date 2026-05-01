package apitools

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	discoveryImportMaxBytes = 20 * 1024 * 1024
	discoveryImportTimeout  = 30 * time.Second
)

// DiscoveryCandidate is a local or imported OpenAPI document candidate.
type DiscoveryCandidate struct {
	Path         string `json:"path,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	Title        string `json:"title,omitempty"`
	Description  string `json:"description,omitempty"`
	Source       string `json:"source,omitempty"`
	Score        int    `json:"score,omitempty"`
}

// DiscoveryReport records discovery attempts for audit and prompting.
type DiscoveryReport struct {
	Attempts []DiscoveryAttempt `json:"attempts,omitempty"`
}

// DiscoveryAttempt describes one local, URL, or catalog discovery attempt.
type DiscoveryAttempt struct {
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Discoverer discovers OpenAPI documents from local files, project URLs, and
// APIs.guru fallback search.
type Discoverer struct {
	HTTPClient      *http.Client
	APIsGuruListURL string
}

func (d *Discoverer) Discover(ctx context.Context, exampleDir, projectText string) ([]DiscoveryCandidate, error) {
	candidates, _, err := d.DiscoverWithReport(ctx, exampleDir, projectText)
	return candidates, err
}

func (d *Discoverer) DiscoverWithReport(ctx context.Context, exampleDir, projectText string) ([]DiscoveryCandidate, DiscoveryReport, error) {
	openAPIDir := filepath.Join(exampleDir, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		return nil, DiscoveryReport{}, err
	}

	var candidates []DiscoveryCandidate
	var report DiscoveryReport
	local, err := DiscoverOpenAPI(ctx, openAPIDir, exampleDir, projectText)
	if err != nil {
		return nil, report, err
	}
	report.Attempts = append(report.Attempts, DiscoveryAttempt{
		Kind:   "local",
		Source: filepath.ToSlash(openAPIDir),
		Status: "pass",
		Detail: fmt.Sprintf("%d local OpenAPI document(s)", len(local)),
	})
	candidates = append(candidates, local...)

	imported, attempts := d.ImportProjectURLsWithReport(ctx, openAPIDir, exampleDir, projectText)
	report.Attempts = append(report.Attempts, attempts...)
	candidates = append(candidates, imported...)

	if len(candidates) == 0 {
		fromGuru, err := d.ImportBestAPIsGuruMatch(ctx, openAPIDir, exampleDir, projectText)
		if err != nil {
			report.Attempts = append(report.Attempts, DiscoveryAttempt{Kind: "apis.guru", Status: "fail", Detail: err.Error()})
			return nil, report, err
		}
		if fromGuru.Path != "" {
			report.Attempts = append(report.Attempts, DiscoveryAttempt{Kind: "apis.guru", Source: fromGuru.Source, Status: "pass", Detail: fromGuru.RelativePath})
			candidates = append(candidates, fromGuru)
		}
	}

	sortDiscoveryCandidates(candidates)
	return candidates, report, nil
}

func (d *Discoverer) ImportProjectURLs(ctx context.Context, openAPIDir, baseDir, projectText string) ([]DiscoveryCandidate, error) {
	out, _ := d.ImportProjectURLsWithReport(ctx, openAPIDir, baseDir, projectText)
	return out, nil
}

func (d *Discoverer) ImportProjectURLsWithReport(ctx context.Context, openAPIDir, baseDir, projectText string) ([]DiscoveryCandidate, []DiscoveryAttempt) {
	var out []DiscoveryCandidate
	var attempts []DiscoveryAttempt
	seen := map[string]bool{}
	for _, rawURL := range ExtractURLs(projectText) {
		if seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		candidate, err := d.ImportURL(ctx, openAPIDir, baseDir, rawURL, "")
		if err != nil {
			attempts = append(attempts, DiscoveryAttempt{Kind: "url", Source: rawURL, Status: "fail", Detail: err.Error()})
			continue
		}
		candidate.Source = "url:" + rawURL
		candidate.Score = ScoreText(projectText, candidate.Title+" "+candidate.Description+" "+candidate.RelativePath)
		attempts = append(attempts, DiscoveryAttempt{Kind: "url", Source: rawURL, Status: "pass", Detail: candidate.RelativePath})
		out = append(out, candidate)
	}
	return out, attempts
}

// DiscoverOpenAPI returns local OpenAPI document candidates under openAPIDir.
func DiscoverOpenAPI(ctx context.Context, openAPIDir, baseDir, projectText string) ([]DiscoveryCandidate, error) {
	results, err := LocalFiles(ctx, LocalOptions{
		Dir:     openAPIDir,
		BaseDir: baseDir,
		Query:   projectText,
	})
	if err != nil {
		return nil, err
	}
	candidates := make([]DiscoveryCandidate, 0, len(results))
	for _, result := range results {
		candidates = append(candidates, DiscoveryCandidate{
			Path:         result.Path,
			RelativePath: result.RelativePath,
			Title:        result.Title,
			Description:  result.Description,
			Source:       "local",
			Score:        result.Score,
		})
	}
	return candidates, nil
}

func (d *Discoverer) ImportBestAPIsGuruMatch(ctx context.Context, openAPIDir, baseDir, projectText string) (DiscoveryCandidate, error) {
	report, err := d.searchClient().Search(ctx, SearchOptions{
		Query:  projectText,
		Limit:  1,
		Source: SourceAPIsGuru,
	})
	if err != nil {
		return DiscoveryCandidate{}, err
	}
	if len(report.Results) == 0 {
		return DiscoveryCandidate{}, fmt.Errorf("no APIs.guru match found for project brief")
	}

	best := report.Results[0]
	candidate, err := d.ImportURL(ctx, openAPIDir, baseDir, best.SpecURL, best.Provider)
	if err != nil {
		return DiscoveryCandidate{}, err
	}
	candidate.Source = "apis.guru:" + best.Provider
	candidate.Score = best.Score
	return candidate, nil
}

func (d *Discoverer) ImportURL(ctx context.Context, openAPIDir, baseDir, rawURL, suggestedName string) (DiscoveryCandidate, error) {
	imported, err := d.searchClient().Import(ctx, ImportOptions{
		URL:  rawURL,
		Dir:  openAPIDir,
		Name: suggestedName,
	})
	if err != nil {
		return DiscoveryCandidate{}, err
	}
	rel, err := filepath.Rel(baseDir, imported.Path)
	if err != nil {
		return DiscoveryCandidate{}, err
	}
	return DiscoveryCandidate{
		Path:         imported.Path,
		RelativePath: filepath.ToSlash(rel),
		Title:        imported.Title,
		Description:  imported.Description,
	}, nil
}

func (d *Discoverer) searchClient() *Client {
	client := &Client{
		Timeout:  discoveryImportTimeout,
		MaxBytes: discoveryImportMaxBytes,
	}
	if d != nil {
		client.HTTPClient = d.HTTPClient
		client.APIsGuruListURL = d.APIsGuruListURL
	}
	return client
}

func SelectPrimaryDiscoveryCandidate(candidates []DiscoveryCandidate) (DiscoveryCandidate, error) {
	if len(candidates) == 0 {
		return DiscoveryCandidate{}, fmt.Errorf("no OpenAPI documents discovered")
	}
	sortDiscoveryCandidates(candidates)
	return candidates[0], nil
}

func sortDiscoveryCandidates(candidates []DiscoveryCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].RelativePath < candidates[j].RelativePath
	})
}

var discoveryURLPattern = regexp.MustCompile(`https?://[^\s<>"')]+`)

// ExtractURLs returns HTTP(S) URLs found in free text, trimmed of trailing
// punctuation common in prose.
func ExtractURLs(text string) []string {
	matches := discoveryURLPattern.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimRight(match, ".,;:"))
	}
	return out
}
