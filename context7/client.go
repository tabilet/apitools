// Package context7 provides an optional Context7 implementation of the generic
// apitools documentation context provider.
package context7

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
)

const DefaultBaseURL = "https://context7.com/api"

const maxOpenAPIUploadBytes = 10 << 20

var ErrLibraryNotReady = errors.New("context7 library is not finalized")

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

func (c *Client) FetchDocumentationContext(ctx context.Context, query apitools.DocumentationContextQuery) (apitools.DocumentationContext, error) {
	var out apitools.DocumentationContext
	for _, libraryID := range query.LibraryIDs {
		libraryID = strings.TrimSpace(libraryID)
		if libraryID == "" {
			continue
		}
		ctxResponse, err := c.getContext(ctx, libraryID, contextQueryText(query))
		if err != nil {
			out.Diagnostics = append(out.Diagnostics, apitools.Diagnostic{
				Severity:    "warning",
				Code:        "documentation.context_unavailable",
				Message:     fmt.Sprintf("context7 library %q unavailable: %v", libraryID, err),
				Remediation: "Retry advisory context later or continue from reviewed OpenAPI inputs.",
			})
			continue
		}
		out.Snippets = append(out.Snippets, mapContextResponse(libraryID, ctxResponse)...)
	}
	return out, nil
}

func (c *Client) AddOpenAPIURL(ctx context.Context, rawURL string) (string, error) {
	body, err := json.Marshal(map[string]string{"openApiUrl": strings.TrimSpace(rawURL)})
	if err != nil {
		return "", err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v2/add/openapi", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	var response addLibraryResponse
	if err := c.doJSON(req, &response); err != nil {
		return "", err
	}
	return response.LibraryName, nil
}

func (c *Client) UploadOpenAPI(ctx context.Context, path, title, description string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxOpenAPIUploadBytes {
		return "", fmt.Errorf("OpenAPI file is larger than Context7 upload limit of %d bytes", maxOpenAPIUploadBytes)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("openapiFile", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, io.LimitReader(file, maxOpenAPIUploadBytes+1)); err != nil {
		return "", err
	}
	if strings.TrimSpace(title) != "" {
		if err := writer.WriteField("libraryTitle", strings.TrimSpace(title)); err != nil {
			return "", err
		}
	}
	if strings.TrimSpace(description) != "" {
		if err := writer.WriteField("description", strings.TrimSpace(description)); err != nil {
			return "", err
		}
	}
	if err := writer.WriteField("skipBenchmark", "true"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/v2/add/openapi-upload", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	var response addLibraryResponse
	if err := c.doJSON(req, &response); err != nil {
		return "", err
	}
	return response.LibraryName, nil
}

func (c *Client) getContext(ctx context.Context, libraryID, query string) (contextResponse, error) {
	values := url.Values{}
	values.Set("libraryId", libraryID)
	values.Set("query", query)
	values.Set("type", "json")
	values.Set("fast", "true")
	req, err := c.newRequest(ctx, http.MethodGet, "/v2/context?"+values.Encode(), nil)
	if err != nil {
		return contextResponse{}, err
	}
	var response contextResponse
	if err := c.doJSON(req, &response); err != nil {
		return contextResponse{}, err
	}
	return response, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func (c *Client) doJSON(req *http.Request, out any) error {
	client := c.HTTPClient
	if client == nil {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted {
		return ErrLibraryNotReady
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("context7 %s %s: status %d", req.Method, req.URL.Path, resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func contextQueryText(query apitools.DocumentationContextQuery) string {
	text := strings.TrimSpace(query.Brief)
	if text == "" {
		text = strings.TrimSpace(query.ProjectName)
	}
	if text == "" {
		var labels []string
		for _, op := range query.Operations {
			labels = append(labels, strings.TrimSpace(firstNonEmpty(op.Summary, op.OperationID, op.ID, op.Method+" "+op.Path)))
		}
		text = strings.Join(labels, "; ")
	}
	if text == "" {
		text = "OpenAPI usage and examples"
	}
	return text
}

func mapContextResponse(libraryID string, response contextResponse) []apitools.DocumentationSnippet {
	var snippets []apitools.DocumentationSnippet
	for _, info := range response.InfoSnippets {
		snippets = append(snippets, apitools.DocumentationSnippet{
			Title:     info.Breadcrumb,
			Content:   info.Content,
			SourceURL: info.PageID,
			Source:    "context7",
			Kind:      "documentation",
			LibraryID: libraryID,
		})
	}
	for _, code := range response.CodeSnippets {
		content := code.CodeDescription
		if content == "" && len(code.CodeList) > 0 {
			content = code.CodeList[0].Code
		}
		snippets = append(snippets, apitools.DocumentationSnippet{
			Title:     firstNonEmpty(code.CodeTitle, code.PageTitle),
			Content:   content,
			SourceURL: code.CodeID,
			Source:    "context7",
			Kind:      "code",
			Language:  code.CodeLanguage,
			LibraryID: libraryID,
		})
	}
	return snippets
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type addLibraryResponse struct {
	LibraryName string `json:"libraryName"`
	Message     string `json:"message"`
}

type contextResponse struct {
	CodeSnippets []codeSnippet `json:"codeSnippets"`
	InfoSnippets []infoSnippet `json:"infoSnippets"`
}

type codeSnippet struct {
	CodeTitle       string        `json:"codeTitle"`
	CodeDescription string        `json:"codeDescription"`
	CodeLanguage    string        `json:"codeLanguage"`
	CodeID          string        `json:"codeId"`
	PageTitle       string        `json:"pageTitle"`
	CodeList        []codeExample `json:"codeList"`
}

type codeExample struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type infoSnippet struct {
	PageID     string `json:"pageId"`
	Breadcrumb string `json:"breadcrumb"`
	Content    string `json:"content"`
}
