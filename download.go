package apitools

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	// Set this ourselves so net/http does not transparently decompress the
	// response before the wire-byte budget can be enforced.
	req.Header.Set("Accept-Encoding", "gzip")
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
		if _, err := c.validateHTTPURL(ctx, finalURL.String()); err != nil {
			return nil, nil, "", err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, "", HTTPStatusError{Code: resp.StatusCode, Status: resp.Status}
	}
	content, err := readBoundedResponseBody(resp, maxBytes)
	if err != nil {
		return nil, nil, "", err
	}
	return content, finalURL, resp.Header.Get("Content-Type"), nil
}

func readBoundedResponseBody(resp *http.Response, maxBytes int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("download response body is missing")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("download byte limit must be positive")
	}
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("downloaded wire body is larger than %d bytes", maxBytes)
	}

	wire := &io.LimitedReader{R: resp.Body, N: maxBytes + 1}
	var decoded io.Reader = wire
	var closeDecoded func() error
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "", "identity":
	case "gzip":
		reader, err := gzip.NewReader(wire)
		if err != nil {
			return nil, fmt.Errorf("decode gzip response: %w", err)
		}
		decoded = reader
		closeDecoded = reader.Close
	default:
		return nil, fmt.Errorf("unsupported Content-Encoding %q", resp.Header.Get("Content-Encoding"))
	}

	content, err := io.ReadAll(io.LimitReader(decoded, maxBytes+1))
	if closeDecoded != nil {
		closeErr := closeDecoded()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("downloaded decoded body is larger than %d bytes", maxBytes)
	}
	// A decoder can finish before consuming trailing bytes. Drain only through
	// the bounded wire reader so appended data cannot evade the wire budget.
	if _, err := io.Copy(io.Discard, wire); err != nil {
		return nil, err
	}
	if wire.N == 0 {
		return nil, fmt.Errorf("downloaded wire body is larger than %d bytes", maxBytes)
	}
	return content, nil
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
		if _, err := c.validateHTTPURL(req.Context(), req.URL.String()); err != nil {
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
	if roundTripper != nil {
		return nil, fmt.Errorf("custom HTTP transport requires AllowUnsafeHosts")
	}
	// Construct the transport rather than cloning the mutable process default.
	// This prevents custom proxy and TLS dial hooks from bypassing the guarded
	// dial path while retaining the standard library's normal timeout profile.
	transport := &http.Transport{
		DialContext:           c.safeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
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
	if err := c.validateDialPort(port); err != nil {
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
	if parsed.User != nil {
		return nil, fmt.Errorf("URL userinfo is not allowed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.rejectHost(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	if err := c.validateURLPort(parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *Client) validateCacheURL(ctx context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("valid URL is required")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("valid URL is required")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme must be http or https")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("URL userinfo is not allowed")
	}
	if c != nil && c.AllowUnsafeHosts {
		if err := c.validateURLPort(parsed); err != nil {
			return nil, err
		}
		return parsed, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.rejectHost(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	if err := c.validateURLPort(parsed); err != nil {
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
	if strings.Contains(host, "%") {
		return fmt.Errorf("refusing scoped URL host %q", host)
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
	if ip == nil || !ip.IsGlobalUnicast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return ipInAnyCIDR(v4, unsafeIPv4Networks)
	}
	return ipInAnyCIDR(ip, unsafeIPv6Networks)
}

var unsafeIPv4Networks = mustIPNetworks(
	"0.0.0.0/8",
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
)

var unsafeIPv6Networks = mustIPNetworks(
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/32",   // Teredo transition addresses.
	"2001:2::/48", // Benchmarking.
	"2001:10::/28",
	"2001:20::/28",
	"2001:db8::/32",
	"2002::/16", // 6to4 transition addresses.
	"fc00::/7",
)

func mustIPNetworks(values ...string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			panic(err)
		}
		networks = append(networks, network)
	}
	return networks
}

func ipInAnyCIDR(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (c *Client) validateURLPort(parsed *url.URL) error {
	if parsed == nil {
		return fmt.Errorf("valid URL is required")
	}
	port := parsed.Port()
	if port == "" {
		return nil
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("URL port must be between 1 and 65535")
	}
	if c != nil && c.AllowUnsafeHosts {
		return nil
	}
	defaultPort := (parsed.Scheme == "http" && value == 80) || (parsed.Scheme == "https" && value == 443)
	if defaultPort {
		return nil
	}
	allowed, err := c.additionalPortAllowed(value)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("refusing URL port %d for %s", value, parsed.Scheme)
	}
	return nil
}

func (c *Client) validateDialPort(port string) error {
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("destination port must be between 1 and 65535")
	}
	if value == 80 || value == 443 {
		return nil
	}
	allowed, err := c.additionalPortAllowed(value)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("refusing destination port %d", value)
	}
	return nil
}

func (c *Client) additionalPortAllowed(port int) (bool, error) {
	if c == nil {
		return false, nil
	}
	allowed := false
	for _, candidate := range c.AllowedPorts {
		if candidate < 1 || candidate > 65535 {
			return false, fmt.Errorf("allowed port must be between 1 and 65535: %d", candidate)
		}
		if candidate == port {
			allowed = true
		}
	}
	return allowed, nil
}
