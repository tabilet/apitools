package apitools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

func (c *Client) searchAPIsGuru(ctx context.Context, query string, limit int) ([]Result, []SearchAttempt, error) {
	c = c.effective()
	listURL := strings.TrimSpace(c.APIsGuruListURL)
	if listURL == "" {
		listURL = DefaultAPIsGuruListURL
	}
	body, finalURL, err := c.downloadBounded(ctx, listURL)
	attempts := []SearchAttempt{{Source: string(SourceAPIsGuru), URL: listURL, Status: "pass"}}
	if err != nil {
		return nil, []SearchAttempt{{Source: string(SourceAPIsGuru), URL: listURL, Status: "fail", Detail: err.Error()}}, err
	}
	if finalURL != nil {
		attempts[0].URL = finalURL.String()
	}
	var raw map[string]apisGuruAPI
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, attempts, fmt.Errorf("parse APIs.guru list: %w", err)
	}
	results := make([]Result, 0, limit)
	for provider, entry := range raw {
		versionID, version := entry.preferred()
		specURL := firstNonEmpty(version.SwaggerYAMLURL, version.SwaggerURL)
		if specURL == "" {
			continue
		}
		title := firstNonEmpty(strings.TrimSpace(version.Info.Title), provider)
		description := strings.TrimSpace(version.Info.Description)
		categories := append([]string(nil), version.Info.Categories...)
		score := scoreText(query, strings.Join(append([]string{provider, title, description}, categories...), " "))
		if score == 0 {
			continue
		}
		results = append(results, Result{
			ID:          "apis-guru:" + provider + ":" + versionID,
			Source:      string(SourceAPIsGuru),
			Provider:    provider,
			Title:       title,
			Description: description,
			Version:     versionID,
			Categories:  categories,
			SpecURL:     specURL,
			Score:       score,
			Validated:   false,
			Provenance:  "APIs.guru OpenAPI Directory",
		})
	}
	sortResults(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, attempts, nil
}

func (c *Client) searchPublicAPIs(ctx context.Context, query string, limit, probeLimit int, mode CacheMode, maxAge time.Duration) ([]Result, []SearchAttempt, error) {
	c = c.effective()
	sourceURL := strings.TrimSpace(c.PublicAPIsURL)
	if sourceURL == "" {
		sourceURL = DefaultPublicAPIsURL
	}
	body, finalURL, err := c.downloadBounded(ctx, sourceURL)
	attempts := []SearchAttempt{{Source: string(SourcePublicAPIs), URL: sourceURL, Status: "pass"}}
	if err != nil {
		return nil, []SearchAttempt{{Source: string(SourcePublicAPIs), URL: sourceURL, Status: "fail", Detail: err.Error()}}, err
	}
	if finalURL != nil {
		attempts[0].URL = finalURL.String()
	}
	var catalog publicAPIsResponse
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, attempts, fmt.Errorf("parse public-apis catalog: %w", err)
	}
	matches := make([]publicAPIEntry, 0, len(catalog.Entries))
	for _, entry := range catalog.Entries {
		score := scoreText(query, entry.API+" "+entry.Description+" "+entry.Category+" "+entry.Link)
		if score == 0 || strings.TrimSpace(entry.Link) == "" {
			continue
		}
		entry.score = score
		matches = append(matches, entry)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].API < matches[j].API
	})
	if probeLimit <= 0 {
		probeLimit = limit * 5
	}
	if probeLimit > 50 {
		probeLimit = 50
	}
	if len(matches) > probeLimit {
		matches = matches[:probeLimit]
	}
	probeTimeout := c.ProbeTimeout
	if probeTimeout <= 0 {
		probeTimeout = DefaultProbeTimeout
	}
	probeBudget := c.PublicProbeBudget
	if probeBudget <= 0 {
		probeBudget = DefaultPublicProbeBudget
	}
	probeCtx, cancelBudget := context.WithTimeout(ctx, probeBudget)
	defer cancelBudget()
	results := make([]Result, 0, limit)
	seen := map[string]bool{}
	for _, entry := range matches {
		for _, candidateURL := range c.wellKnownURLs(entry.Link) {
			if err := probeCtx.Err(); err != nil {
				attempts = append(attempts, SearchAttempt{Source: string(SourcePublicAPIs), Status: "fail", Detail: "public-apis probe budget exceeded"})
				sortResults(results)
				return results, attempts, ErrProbeBudgetExceeded
			}
			if seen[candidateURL] {
				continue
			}
			seen[candidateURL] = true
			candidateCtx, cancelCandidate := context.WithTimeout(probeCtx, probeTimeout)
			_, _, metadata, err := c.downloadSpecWithCache(candidateCtx, candidateURL, mode, maxAge)
			cancelCandidate()
			if err != nil {
				attempts = append(attempts, SearchAttempt{Source: string(SourcePublicAPIs), URL: candidateURL, Status: "fail", Detail: err.Error()})
				if probeCtx.Err() != nil {
					attempts = append(attempts, SearchAttempt{Source: string(SourcePublicAPIs), Status: "fail", Detail: "public-apis probe budget exceeded"})
					sortResults(results)
					return results, attempts, ErrProbeBudgetExceeded
				}
				continue
			}
			attempts = append(attempts, SearchAttempt{Source: string(SourcePublicAPIs), URL: candidateURL, Status: "pass"})
			title := firstNonEmpty(metadata.Title, entry.API)
			results = append(results, Result{
				ID:          "public-apis:" + entry.API + ":" + candidateURL,
				Source:      string(SourcePublicAPIs),
				Provider:    entry.API,
				Title:       title,
				Description: firstNonEmpty(metadata.Description, entry.Description),
				Categories:  nonEmptyStrings(entry.Category),
				SpecURL:     candidateURL,
				LandingURL:  entry.Link,
				Score:       entry.score,
				Validated:   true,
				Provenance:  "public-apis catalog plus well-known OpenAPI path probe",
			})
			if len(results) >= limit {
				sortResults(results)
				return results, attempts, nil
			}
		}
	}
	sortResults(results)
	return results, attempts, nil
}

func (c *Client) wellKnownURLs(rawBase string) []string {
	c = c.effective()
	parsed, err := url.Parse(strings.TrimSpace(rawBase))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil
	}
	paths := append([]string(nil), c.WellKnownPaths...)
	if len(paths) == 0 {
		paths = WellKnownPaths()
	}
	out := make([]string, 0, len(paths))
	for _, suffix := range paths {
		next := *parsed
		next.RawQuery = ""
		next.Fragment = ""
		next.Path = "/" + strings.TrimLeft(suffix, "/")
		out = append(out, next.String())
	}
	return out
}

type apisGuruAPI struct {
	Versions  map[string]apisGuruVersion `json:"versions"`
	Preferred string                     `json:"preferred"`
}

type apisGuruVersion struct {
	SwaggerURL     string `json:"swaggerUrl"`
	SwaggerYAMLURL string `json:"swaggerYamlUrl"`
	Info           struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Categories  []string `json:"x-apisguru-categories"`
	} `json:"info"`
}

func (a apisGuruAPI) preferred() (string, apisGuruVersion) {
	if a.Preferred != "" {
		if version, ok := a.Versions[a.Preferred]; ok {
			return a.Preferred, version
		}
	}
	keys := make([]string, 0, len(a.Versions))
	for key := range a.Versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "", apisGuruVersion{}
	}
	key := keys[len(keys)-1]
	return key, a.Versions[key]
}

type publicAPIsResponse struct {
	Entries []publicAPIEntry `json:"entries"`
}

type publicAPIEntry struct {
	API         string `json:"API"`
	Description string `json:"Description"`
	Link        string `json:"Link"`
	Category    string `json:"Category"`
	score       int
}
