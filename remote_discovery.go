package apitools

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type lapSearchEntry struct {
	Name           string
	URL            string
	Description    string
	Version        string
	BaseURL        string
	SourceURL      string
	ProviderSlug   string
	ProviderName   string
	ProviderDomain string
}

func (c *Client) searchLAPRegistry(ctx context.Context, query string, limit int) ([]Result, []SearchAttempt, error) {
	c = c.effective()
	searchURL := strings.TrimSpace(c.LAPSearchURL)
	if searchURL == "" {
		searchURL = DefaultLAPSearchURL
	}
	parsed, err := url.Parse(searchURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, nil, fmt.Errorf("valid LAP Registry search URL is required")
	}
	values := parsed.Query()
	values.Set("q", query)
	values.Set("limit", strconv.Itoa(limit))
	parsed.RawQuery = values.Encode()
	requestURL := parsed.String()
	body, finalURL, contentType, err := c.downloadBoundedWithAccept(ctx, requestURL, "text/lap")
	attempt := SearchAttempt{Source: string(SourceLAPRegistry), URL: requestURL, Status: "pass"}
	if err != nil {
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	if finalURL != nil {
		attempt.URL = finalURL.String()
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "text/lap") {
		err := fmt.Errorf("LAP Registry search must return text/lap, got %q", contentType)
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	entries, err := parseLAPSearch(body)
	if err != nil {
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	results := make([]Result, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		specURL, ok := absoluteHTTPURL(entry.SourceURL)
		if strings.TrimSpace(entry.Name) == "" {
			err := fmt.Errorf("parse LAP Registry search: result name is required")
			attempt.Status = "fail"
			attempt.Detail = err.Error()
			return nil, []SearchAttempt{attempt}, err
		}
		if !ok {
			err := fmt.Errorf("parse LAP Registry search result %q: valid HTTP(S) source_url is required", entry.Name)
			attempt.Status = "fail"
			attempt.Detail = err.Error()
			return nil, []SearchAttempt{attempt}, err
		}
		if _, err := c.validateHTTPURL(ctx, specURL); err != nil {
			err := fmt.Errorf("LAP Registry source_url %q: %w", specURL, err)
			attempt.Status = "fail"
			attempt.Detail = err.Error()
			return nil, []SearchAttempt{attempt}, err
		}
		if _, ok := seen[specURL]; ok {
			continue
		}
		seen[specURL] = struct{}{}
		provider := firstNonEmpty(entry.ProviderName, entry.ProviderDomain, entry.ProviderSlug)
		title := firstNonEmpty(entry.Description, entry.Name)
		score := scoreText(query, strings.Join([]string{entry.Name, entry.Description, entry.ProviderName, entry.ProviderDomain, entry.BaseURL}, " "))
		if score == 0 {
			continue
		}
		results = append(results, Result{
			ID:           "lap-registry:" + entry.Name,
			Source:       string(SourceLAPRegistry),
			Provider:     provider,
			Title:        title,
			Description:  entry.Description,
			Version:      entry.Version,
			SpecURL:      specURL,
			LandingURL:   entry.URL,
			Score:        score,
			Validated:    false,
			Provenance:   "LAP Registry reported original source URL; document not yet validated by apitools",
			Experimental: true,
		})
	}
	sortResults(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, []SearchAttempt{attempt}, nil
}

func parseLAPSearch(body []byte) ([]lapSearchEntry, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var entries []lapSearchEntry
	var current *lapSearchEntry
	seenHeader := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "@") {
			continue
		}
		name, value, _ := strings.Cut(strings.TrimPrefix(line, "@"), " ")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		switch name {
		case "search_query":
			seenHeader = true
		case "result":
			if current != nil {
				return nil, fmt.Errorf("parse LAP Registry search: nested result block")
			}
			current = &lapSearchEntry{}
		case "endresult":
			if current == nil {
				return nil, fmt.Errorf("parse LAP Registry search: endresult without result")
			}
			entries = append(entries, *current)
			current = nil
		default:
			if current == nil {
				continue
			}
			switch name {
			case "name":
				current.Name = value
			case "url":
				current.URL = value
			case "description":
				current.Description = value
			case "version":
				current.Version = value
			case "base":
				current.BaseURL = value
			case "source_url":
				current.SourceURL = value
			case "provider":
				parts := strings.Split(value, "|")
				if len(parts) > 0 {
					current.ProviderSlug = strings.TrimSpace(parts[0])
				}
				if len(parts) > 1 {
					current.ProviderName = strings.TrimSpace(parts[1])
				}
				if len(parts) > 2 {
					current.ProviderDomain = strings.TrimSpace(parts[2])
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse LAP Registry search: %w", err)
	}
	if current != nil {
		return nil, fmt.Errorf("parse LAP Registry search: unterminated result block")
	}
	if !seenHeader {
		return nil, fmt.Errorf("parse LAP Registry search: missing search_query header")
	}
	return entries, nil
}

type rfc9727Document struct {
	Linkset []rfc9727Linkset `json:"linkset"`
}

type rfc9727Linkset struct {
	Anchor      string          `json:"anchor"`
	ServiceDesc json.RawMessage `json:"service-desc"`
}

type rfc9727Target struct {
	Href  string `json:"href"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

func (c *Client) searchRFC9727(ctx context.Context, query, providerURL string, limit int) ([]Result, []SearchAttempt, error) {
	c = c.effective()
	catalogURL, err := rfc9727CatalogURL(providerURL)
	if err != nil {
		return nil, nil, err
	}
	body, finalURL, contentType, err := c.downloadBoundedWithAccept(ctx, catalogURL, "application/linkset+json")
	attempt := SearchAttempt{Source: string(SourceRFC9727), URL: catalogURL, Status: "pass"}
	if err != nil {
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	if finalURL != nil {
		attempt.URL = finalURL.String()
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/linkset+json") {
		err := fmt.Errorf("RFC 9727 catalog must return application/linkset+json, got %q", contentType)
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	document, err := parseRFC9727Document(body)
	if err != nil {
		err := fmt.Errorf("parse RFC 9727 API catalog: %w", err)
		attempt.Status = "fail"
		attempt.Detail = err.Error()
		return nil, []SearchAttempt{attempt}, err
	}
	baseURL := finalURL
	if baseURL == nil {
		baseURL, _ = url.Parse(catalogURL)
	}
	provider := baseURL.Hostname()
	var results []Result
	seen := map[string]struct{}{}
	linkLimit := c.RFC9727LinkLimit
	if linkLimit <= 0 {
		linkLimit = DefaultRFC9727LinkLimit
	}
	visitedTargets := 0
	for _, linkset := range document.Linkset {
		targets, err := parseRFC9727Targets(linkset.ServiceDesc)
		if err != nil {
			err := fmt.Errorf("parse RFC 9727 service-desc: %w", err)
			attempt.Status = "fail"
			attempt.Detail = err.Error()
			return nil, []SearchAttempt{attempt}, err
		}
		if len(targets) == 0 {
			continue
		}
		anchorURL, err := c.resolveSafeRFC9727URL(ctx, baseURL, linkset.Anchor)
		if err != nil {
			err := fmt.Errorf("parse RFC 9727 anchor: %w", err)
			attempt.Status = "fail"
			attempt.Detail = err.Error()
			return nil, []SearchAttempt{attempt}, err
		}
		for _, target := range targets {
			visitedTargets++
			if visitedTargets > linkLimit {
				err := fmt.Errorf("RFC 9727 catalog exceeds the %d service-desc link limit", linkLimit)
				attempt.Status = "fail"
				attempt.Detail = err.Error()
				return nil, []SearchAttempt{attempt}, err
			}
			if !openAPIDescriptionTarget(target) {
				continue
			}
			specURL, err := c.resolveSafeRFC9727URL(ctx, baseURL, target.Href)
			if err != nil {
				candidateURL := target.Href
				if specURL != nil {
					candidateURL = specURL.String()
				}
				err := fmt.Errorf("RFC 9727 service-desc URL %q: %w", candidateURL, err)
				attempt.Status = "fail"
				attempt.Detail = err.Error()
				return nil, []SearchAttempt{attempt}, err
			}
			if _, ok := seen[specURL.String()]; ok {
				continue
			}
			seen[specURL.String()] = struct{}{}
			title := firstNonEmpty(strings.TrimSpace(target.Title), rfc9727AnchorTitle(anchorURL), provider)
			score := scoreText(query, strings.Join([]string{title, provider, anchorURL.String(), specURL.String()}, " "))
			if score == 0 {
				score = 1
			}
			results = append(results, Result{
				ID:           rfc9727ResultID(provider, specURL.String()),
				Source:       string(SourceRFC9727),
				Provider:     provider,
				Title:        title,
				SpecURL:      specURL.String(),
				LandingURL:   anchorURL.String(),
				Score:        score,
				Validated:    false,
				Provenance:   "RFC 9727 publisher API catalog " + baseURL.String() + "; service description not yet validated by apitools",
				MediaType:    normalizeMediaType(target.Type),
				Experimental: true,
			})
		}
	}
	sortResults(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, []SearchAttempt{attempt}, nil
}

func rfc9727CatalogURL(providerURL string) (string, error) {
	raw := strings.TrimSpace(providerURL)
	if raw == "" {
		return "", fmt.Errorf("provider URL or hostname is required for RFC 9727 discovery")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Hostname() == "" {
		return "", fmt.Errorf("valid provider URL or hostname is required for RFC 9727 discovery")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("provider URL scheme must be http or https")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("provider URL must not contain user information")
	}
	parsed.Path = DefaultAPICatalogPath
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func parseRFC9727Document(body []byte) (rfc9727Document, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return rfc9727Document{}, err
	}
	if len(envelope) != 1 {
		return rfc9727Document{}, fmt.Errorf("linkset must be the sole top-level member")
	}
	raw, ok := envelope["linkset"]
	if !ok {
		return rfc9727Document{}, fmt.Errorf("linkset array is required")
	}
	var document rfc9727Document
	if err := json.Unmarshal(raw, &document.Linkset); err != nil {
		return rfc9727Document{}, fmt.Errorf("linkset must be an array: %w", err)
	}
	if document.Linkset == nil {
		return rfc9727Document{}, fmt.Errorf("linkset array is required")
	}
	return document, nil
}

func parseRFC9727Targets(raw json.RawMessage) ([]rfc9727Target, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	if raw[0] != '[' {
		return nil, fmt.Errorf("service-desc must be an array")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	out := make([]rfc9727Target, 0, len(values))
	for _, value := range values {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(value, &fields); err != nil {
			return nil, err
		}
		href, ok := fields["href"]
		if !ok {
			return nil, fmt.Errorf("target href is required")
		}
		var target rfc9727Target
		if err := json.Unmarshal(href, &target.Href); err != nil {
			return nil, fmt.Errorf("target href must be a string: %w", err)
		}
		if value, ok := fields["type"]; ok {
			if err := json.Unmarshal(value, &target.Type); err != nil {
				return nil, fmt.Errorf("target type must be a string: %w", err)
			}
		}
		if value, ok := fields["title"]; ok {
			if err := json.Unmarshal(value, &target.Title); err != nil {
				return nil, fmt.Errorf("target title must be a string: %w", err)
			}
		}
		out = append(out, target)
	}
	return out, nil
}

func resolveRFC9727URL(base *url.URL, raw string) (*url.URL, error) {
	if base == nil {
		return nil, fmt.Errorf("base URL is required")
	}
	if strings.TrimSpace(raw) == "" {
		return base, nil
	}
	reference, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	resolved := base.ResolveReference(reference)
	if resolved.Scheme != "http" && resolved.Scheme != "https" || resolved.Hostname() == "" {
		return nil, fmt.Errorf("resolved URL must use HTTP(S) and include a host")
	}
	return resolved, nil
}

func (c *Client) resolveSafeRFC9727URL(ctx context.Context, base *url.URL, raw string) (*url.URL, error) {
	resolved, err := resolveRFC9727URL(base, raw)
	if err != nil {
		return nil, err
	}
	if _, err := c.validateHTTPURL(ctx, resolved.String()); err != nil {
		return resolved, err
	}
	return resolved, nil
}

func openAPIDescriptionTarget(target rfc9727Target) bool {
	mediaType := normalizeMediaType(target.Type)
	switch mediaType {
	case "application/json", "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml", "application/openapi+json", "application/openapi+yaml":
		return true
	}
	if strings.HasPrefix(mediaType, "application/vnd.oai.openapi") {
		return true
	}
	href := strings.ToLower(strings.TrimSpace(target.Href))
	if parsed, err := url.Parse(href); err == nil {
		href = parsed.Path
	}
	return mediaType == "" && (strings.HasSuffix(href, ".json") || strings.HasSuffix(href, ".yaml") || strings.HasSuffix(href, ".yml"))
}

func normalizeMediaType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	return strings.ToLower(mediaType)
}

func absoluteHTTPURL(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	return parsed.String(), true
}

func rfc9727AnchorTitle(anchor *url.URL) string {
	if anchor == nil {
		return ""
	}
	title := path.Base(strings.TrimSuffix(anchor.Path, "/"))
	if title == "." || title == "/" {
		return ""
	}
	decoded, err := url.PathUnescape(title)
	if err == nil {
		title = decoded
	}
	return title
}

func rfc9727ResultID(provider, specURL string) string {
	digest := sha256.Sum256([]byte(specURL))
	return "rfc9727:" + provider + ":" + hex.EncodeToString(digest[:8])
}
