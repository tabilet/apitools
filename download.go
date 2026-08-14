package apitools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPStatusError reports a non-2xx response from a downloaded OpenAPI URL.
type HTTPStatusError struct {
	Code   int
	Status string
}

func (err HTTPStatusError) Error() string {
	return fmt.Sprintf("download URL: %s", err.Status)
}

func (c *Client) downloadSpec(ctx context.Context, rawURL string) ([]byte, *url.URL, SpecMetadata, error) {
	c = c.effective()
	content, finalURL, err := c.downloadBounded(ctx, rawURL)
	if err != nil {
		return nil, nil, SpecMetadata{}, err
	}
	metadata, ok := downloadedSpecMetadata(ctx, content, finalURL.String())
	if !ok {
		return nil, nil, SpecMetadata{}, fmt.Errorf("downloaded document does not look like OpenAPI or Swagger")
	}
	return content, finalURL, metadata, nil
}

func (c *Client) downloadSpecWithCache(ctx context.Context, rawURL string, mode CacheMode, maxAge time.Duration) ([]byte, *url.URL, SpecMetadata, error) {
	c = c.effective()
	mode, err := normalizeCacheMode(mode)
	if err != nil {
		return nil, nil, SpecMetadata{}, err
	}
	if _, err := c.validateCacheURL(ctx, rawURL); err != nil {
		return nil, nil, SpecMetadata{}, err
	}
	if c.Cache != nil && mode != CacheModeRefresh && mode != CacheModeBypass {
		spec, ok, err := c.Cache.LoadSpec(ctx, rawURL, maxAge)
		if err != nil {
			if mode == CacheModeOffline || !errors.Is(err, ErrCachedSpecIntegrity) {
				return nil, nil, SpecMetadata{}, err
			}
			ok = false
		}
		if ok {
			content, finalURL, metadata, err := cachedSpecContent(ctx, rawURL, spec)
			if err == nil {
				return content, finalURL, metadata, nil
			}
			if mode == CacheModeOffline {
				return nil, nil, SpecMetadata{}, err
			}
		}
	}
	if mode == CacheModeOffline {
		if c.Cache == nil {
			return nil, nil, SpecMetadata{}, fmt.Errorf("cache is required for offline import")
		}
		return nil, nil, SpecMetadata{}, fmt.Errorf("OpenAPI document %q is not cached", rawURL)
	}
	content, finalURL, metadata, err := c.downloadSpec(ctx, rawURL)
	if err != nil {
		return nil, nil, SpecMetadata{}, err
	}
	if c.Cache != nil && mode != CacheModeBypass {
		digest := sha256.Sum256(content)
		err := c.Cache.StoreSpec(ctx, CachedSpec{
			OriginalURL: strings.TrimSpace(rawURL),
			FinalURL:    finalURL.String(),
			Content:     append([]byte(nil), content...),
			SHA256:      hex.EncodeToString(digest[:]),
			Bytes:       int64(len(content)),
			Metadata:    metadata,
		})
		if err != nil {
			return nil, nil, SpecMetadata{}, err
		}
	}
	return content, finalURL, metadata, nil
}

func cachedSpecContent(ctx context.Context, rawURL string, spec CachedSpec) ([]byte, *url.URL, SpecMetadata, error) {
	finalURL, err := url.Parse(strings.TrimSpace(spec.FinalURL))
	if err != nil {
		return nil, nil, SpecMetadata{}, fmt.Errorf("%w: cached OpenAPI document %q has invalid final URL: %v", ErrCachedSpecIntegrity, rawURL, err)
	}
	content := append([]byte(nil), spec.Content...)
	if spec.SHA256 != "" {
		digest := sha256.Sum256(content)
		if got := hex.EncodeToString(digest[:]); got != spec.SHA256 {
			return nil, nil, SpecMetadata{}, fmt.Errorf("%w: cached OpenAPI document %q has SHA256 %s, want %s", ErrCachedSpecIntegrity, rawURL, got, spec.SHA256)
		}
	}
	metadata, ok := downloadedSpecMetadata(ctx, content, finalURL.String())
	if !ok {
		return nil, nil, SpecMetadata{}, fmt.Errorf("%w: cached OpenAPI document %q is invalid", ErrCachedSpecIntegrity, rawURL)
	}
	return content, finalURL, metadata, nil
}

func (c *Client) downloadBounded(ctx context.Context, rawURL string) ([]byte, *url.URL, error) {
	content, finalURL, _, err := c.downloadBoundedWithAccept(ctx, rawURL, "")
	return content, finalURL, err
}

func (c *Client) downloadBoundedWithAccept(ctx context.Context, rawURL, accept string) ([]byte, *url.URL, string, error) {
	c = c.effective()
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := c.validateHTTPURL(ctx, rawURL)
	if err != nil {
		return nil, nil, "", err
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxBytes := c.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, nil, "", err
	}
	if strings.TrimSpace(accept) != "" {
		req.Header.Set("Accept", accept)
	}
	client, err := c.redirectSafeClient()
	if err != nil {
		return nil, nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, "", err
	}
	defer resp.Body.Close()
	finalURL := parsed
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL
		if err := c.rejectHost(ctx, finalURL.Hostname()); err != nil {
			return nil, nil, "", err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, "", HTTPStatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, nil, "", err
	}
	if int64(len(content)) > maxBytes {
		return nil, nil, "", fmt.Errorf("downloaded document is larger than %d bytes", maxBytes)
	}
	return content, finalURL, resp.Header.Get("Content-Type"), nil
}

func (c *Client) client() *http.Client {
	c = c.effective()
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) redirectSafeClient() (*http.Client, error) {
	c = c.effective()
	base := c.client()
	clone := *base
	transport, err := c.safeTransport(base.Transport)
	if err != nil {
		return nil, err
	}
	clone.Transport = transport
	baseCheck := base.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if req == nil || req.URL == nil {
			return fmt.Errorf("redirect target is missing")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("redirect URL scheme must be http or https")
		}
		if err := c.rejectHost(req.Context(), req.URL.Hostname()); err != nil {
			return err
		}
		if baseCheck != nil {
			return baseCheck(req, via)
		}
		return nil
	}
	return &clone, nil
}

func (c *Client) safeTransport(roundTripper http.RoundTripper) (http.RoundTripper, error) {
	c = c.effective()
	if c != nil && c.AllowUnsafeHosts {
		return roundTripper, nil
	}
	var transport *http.Transport
	if roundTripper == nil {
		if base, ok := http.DefaultTransport.(*http.Transport); ok {
			transport = base.Clone()
		}
	} else if base, ok := roundTripper.(*http.Transport); ok {
		transport = base.Clone()
	} else {
		return nil, fmt.Errorf("custom HTTP transport requires AllowUnsafeHosts")
	}
	if transport == nil {
		return nil, fmt.Errorf("default HTTP transport is not cloneable")
	}
	transport.Proxy = nil
	transport.DialContext = c.safeDialContext
	transport.DialTLS = nil
	transport.DialTLSContext = nil
	return transport, nil
}

func (c *Client) safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// Resolve through the process resolver, then filter every returned IP before
	// dialing. This keeps the guard in apitools even when a container or host DNS
	// resolver applies its own policy.
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var firstErr error
	dialer := &net.Dialer{}
	for _, addr := range addrs {
		if isUnsafeIP(addr.IP) {
			if firstErr == nil {
				firstErr = fmt.Errorf("refusing private URL host %q", host)
			}
			continue
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, fmt.Errorf("no public IP addresses found for %q", host)
}

func (c *Client) validateHTTPURL(ctx context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("valid URL is required")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be http or https")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.rejectHost(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *Client) validateCacheURL(ctx context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("valid URL is required")
	}
	if c != nil && c.AllowUnsafeHosts {
		return parsed, nil
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("valid URL is required")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be http or https")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.rejectHost(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *Client) rejectHost(ctx context.Context, host string) error {
	if c != nil && c.AllowUnsafeHosts {
		return nil
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("URL host is required")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("refusing localhost URL")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if isUnsafeIP(ip) {
			return fmt.Errorf("refusing private URL host %q", host)
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if isUnsafeIP(addr.IP) {
			return fmt.Errorf("refusing private URL host %q", host)
		}
	}
	return nil
}

func isUnsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
