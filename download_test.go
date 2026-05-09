package apitools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDownloadBoundedRejectsNonHTTPScheme(t *testing.T) {
	c := &Client{}
	for _, raw := range []string{"ftp://example.com/spec", "gopher://example.com/0/", "ws://example.com/"} {
		_, _, err := c.downloadBounded(context.Background(), raw)
		if err == nil || !strings.Contains(err.Error(), "scheme") {
			t.Fatalf("scheme %q: expected scheme error, got %v", raw, err)
		}
	}
}

func TestDownloadBoundedRejectsURLWithoutHost(t *testing.T) {
	c := &Client{}
	for _, raw := range []string{"file:///etc/passwd", "javascript:void(0)", "https:///"} {
		_, _, err := c.downloadBounded(context.Background(), raw)
		if err == nil {
			t.Fatalf("input %q: expected error, got nil", raw)
		}
	}
}

func TestDownloadBoundedRejectsLocalhost(t *testing.T) {
	c := &Client{}
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]", "10.0.0.1", "192.168.1.1", "169.254.0.1", "224.0.0.1", "0.0.0.0"} {
		_, _, err := c.downloadBounded(context.Background(), "http://"+host+"/openapi.yaml")
		if err == nil {
			t.Fatalf("host %q: expected rejection, got nil error", host)
		}
		if !strings.Contains(err.Error(), "private") && !strings.Contains(err.Error(), "localhost") {
			t.Fatalf("host %q: expected private/localhost error, got %v", host, err)
		}
	}
}

func TestDownloadBoundedRejectsBlankURL(t *testing.T) {
	c := &Client{}
	for _, raw := range []string{"", "   ", "not a url", "://missing-scheme"} {
		_, _, err := c.downloadBounded(context.Background(), raw)
		if err == nil {
			t.Fatalf("input %q: expected error, got nil", raw)
		}
	}
}

func TestDownloadBoundedAllowUnsafeHostsReachesLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	body, finalURL, err := c.downloadBounded(context.Background(), server.URL+"/spec")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" || finalURL == nil {
		t.Fatalf("body=%q url=%v", body, finalURL)
	}
}

func TestDownloadBoundedRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, err := c.downloadBounded(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
	var statusErr HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected HTTPStatusError, got %T", err)
	}
	if statusErr.Code != http.StatusInternalServerError || statusErr.Status != "500 Internal Server Error" {
		t.Fatalf("status error = %#v", statusErr)
	}
}

func TestDownloadBoundedEnforcesSizeCap(t *testing.T) {
	body := strings.Repeat("a", 2048)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true, MaxBytes: 1024}
	_, _, err := c.downloadBounded(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestDownloadBoundedEnforcesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true, Timeout: 50 * time.Millisecond}
	_, _, err := c.downloadBounded(context.Background(), server.URL)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestDownloadBoundedRejectsTooManyRedirects(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, server.URL+"/next", http.StatusFound)
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, err := c.downloadBounded(context.Background(), server.URL+"/start")
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect error, got %v", err)
	}
}

func TestDownloadBoundedRejectsRedirectToNonHTTPScheme(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "ftp://example.com/data", http.StatusFound)
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, err := c.downloadBounded(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected scheme error on redirect, got %v", err)
	}
}

func TestSafeTransportRejectsCustomTransportWithoutOptIn(t *testing.T) {
	c := &Client{HTTPClient: &http.Client{Transport: stubTransport{}}}
	_, err := c.safeTransport(stubTransport{})
	if err == nil || !strings.Contains(err.Error(), "AllowUnsafeHosts") {
		t.Fatalf("expected unsafe-transport error, got %v", err)
	}
}

func TestSafeTransportPassesThroughWhenAllowed(t *testing.T) {
	c := &Client{AllowUnsafeHosts: true}
	rt, err := c.safeTransport(stubTransport{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rt.(stubTransport); !ok {
		t.Fatalf("expected pass-through transport, got %T", rt)
	}
}

func TestRejectHostBypassedWhenAllowUnsafeHosts(t *testing.T) {
	c := &Client{AllowUnsafeHosts: true}
	if err := c.rejectHost(context.Background(), "localhost"); err != nil {
		t.Fatalf("AllowUnsafeHosts should bypass: %v", err)
	}
}

func TestValidateHTTPURLRejectsEmptyHost(t *testing.T) {
	c := &Client{}
	_, err := c.validateHTTPURL(context.Background(), "https:///path")
	if err == nil {
		t.Fatalf("expected error for empty host")
	}
}

func TestDownloadSpecReportsInvalidContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not openapi at all"))
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, _, err := c.downloadSpec(context.Background(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "OpenAPI") {
		t.Fatalf("expected OpenAPI shape error, got %v", err)
	}
}

func TestDownloadSpecAcceptsValidOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, validOpenAPISpec)
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, metadata, err := c.downloadSpec(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Example" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestDownloadedSpecMetadataAcceptsAWSOpenAPIFromAPIsGuruFallbackSource(t *testing.T) {
	content := []byte(`openapi: 3.0.0
info:
  title: AWS Lambda
  version: "2015-03-31"
paths:
  /2021-10-31/functions/{FunctionName}/url:
    post:
      operationId: CreateFunctionUrlConfig
`)

	metadata, ok := downloadedSpecMetadata(context.Background(), content, "https://raw.githubusercontent.com/APIs-guru/openapi-directory/main/APIs/amazonaws.com/lambda/2015-03-31/openapi.yaml")
	if !ok {
		t.Fatal("expected AWS APIs.guru fallback document to be accepted")
	}
	if metadata.Title != "AWS Lambda" || metadata.OpenAPI != "3.0.0" || metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestDownloadSpecRejectsLooseAWSOpenAPIFromUntrustedSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: AWS Lambda
  version: "2015-03-31"
paths:
  /2021-10-31/functions/{FunctionName}/url:
    post:
      operationId: CreateFunctionUrlConfig
`))
	}))
	defer server.Close()

	c := &Client{AllowUnsafeHosts: true}
	_, _, _, err := c.downloadSpec(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected loose AWS-looking document from untrusted source to be rejected")
	}
}

type stubTransport struct{}

func (stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("stub")
}

const validOpenAPISpec = `{
  "openapi": "3.0.0",
  "info": {"title": "Example", "version": "1.0.0"},
  "paths": {"/ping": {"get": {"responses": {"200": {"description": "ok"}}}}}
}`
