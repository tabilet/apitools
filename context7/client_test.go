package context7

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestFetchDocumentationContextMapsSnippets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/context" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("libraryId") != "/acme/api" || r.URL.Query().Get("type") != "json" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(contextResponse{
			InfoSnippets: []infoSnippet{{PageID: "https://example.com/docs", Breadcrumb: "API > Create", Content: "Create things."}},
			CodeSnippets: []codeSnippet{{
				CodeTitle:       "Create example",
				CodeDescription: "Example request.",
				CodeLanguage:    "json",
				CodeID:          "https://example.com/code",
				CodeList:        []codeExample{{Language: "json", Code: "{}"}},
			}},
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, APIKey: "test-key"}
	got, err := client.FetchDocumentationContext(context.Background(), apitools.DocumentationContextQuery{
		Brief:      "create a thing",
		LibraryIDs: []string{"/acme/api"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Snippets) != 2 {
		t.Fatalf("snippets = %#v", got.Snippets)
	}
	if got.Snippets[0].Kind != "documentation" || got.Snippets[1].Kind != "code" {
		t.Fatalf("snippets = %#v", got.Snippets)
	}
}

func TestUploadOpenAPIUsesMultipartFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/add/openapi-upload" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatal(err)
		}
		file, _, err := r.FormFile("openapiFile")
		if err != nil {
			t.Fatal(err)
		}
		_ = file.Close()
		_ = json.NewEncoder(w).Encode(addLibraryResponse{LibraryName: "/acme/uploaded", Message: "ok"})
	}))
	defer server.Close()

	path := t.TempDir() + "/openapi.yaml"
	if err := os.WriteFile(path, []byte("openapi: 3.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	client := &Client{BaseURL: server.URL, APIKey: "test-key"}
	libraryID, err := client.UploadOpenAPI(context.Background(), path, "Title", "Description")
	if err != nil {
		t.Fatal(err)
	}
	if libraryID != "/acme/uploaded" {
		t.Fatalf("libraryID = %q", libraryID)
	}
}

func TestAddOpenAPIURLAndNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/add/openapi":
			_ = json.NewEncoder(w).Encode(addLibraryResponse{LibraryName: "/acme/url", Message: "ok"})
		case "/v2/context":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, APIKey: "test-key"}
	libraryID, err := client.AddOpenAPIURL(context.Background(), "https://example.com/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if libraryID != "/acme/url" {
		t.Fatalf("libraryID = %q", libraryID)
	}
	got, err := client.FetchDocumentationContext(context.Background(), apitools.DocumentationContextQuery{
		Brief:      "x",
		LibraryIDs: []string{"/acme/url"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Diagnostics) != 1 || !strings.Contains(got.Diagnostics[0].Message, ErrLibraryNotReady.Error()) {
		t.Fatalf("context = %#v", got)
	}
}

func TestFetchDocumentationContextKeepsPartialResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("libraryId") {
		case "/acme/good":
			_ = json.NewEncoder(w).Encode(contextResponse{
				InfoSnippets: []infoSnippet{{PageID: "https://example.com/good", Content: "good docs"}},
			})
		case "/acme/not-ready":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("library = %s", r.URL.Query().Get("libraryId"))
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	got, err := client.FetchDocumentationContext(context.Background(), apitools.DocumentationContextQuery{
		Brief:      "x",
		LibraryIDs: []string{"/acme/good", "/acme/not-ready"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Snippets) != 1 || len(got.Diagnostics) != 1 {
		t.Fatalf("context = %#v", got)
	}
}

func TestUploadOpenAPIRejectsOversizedFile(t *testing.T) {
	path := t.TempDir() + "/large-openapi.yaml"
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{1}, maxOpenAPIUploadBytes); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	client := &Client{}
	_, err = client.UploadOpenAPI(context.Background(), path, "", "")
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("err = %v", err)
	}
}
