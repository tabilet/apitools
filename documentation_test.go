package apitools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeDocumentationContextCapsAndKeepsProvenance(t *testing.T) {
	context, diagnostics := SanitizeDocumentationContext(DocumentationContext{
		Snippets: []DocumentationSnippet{
			{Title: " First\nSnippet ", Content: strings.Repeat("a", 20), SourceURL: "https://example.com/docs", Source: "fixture", Kind: "guide"},
			{Title: "Second", Content: "body"},
		},
	}, DocumentationContextOptions{MaxSnippets: 1, MaxContent: 10})
	if len(context.Snippets) != 1 {
		t.Fatalf("snippets = %#v", context.Snippets)
	}
	snippet := context.Snippets[0]
	if snippet.Title != "First Snippet" || snippet.SourceURL != "https://example.com/docs" || snippet.Content != "aaaaaaaaaa..." {
		t.Fatalf("snippet = %#v", snippet)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "documentation.snippet_limit" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestSanitizeDocumentationContextDropsCredentialLikeSnippet(t *testing.T) {
	context, diagnostics := SanitizeDocumentationContext(DocumentationContext{
		Snippets: []DocumentationSnippet{{
			Title:   "bad",
			Content: `api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"`,
		}},
	}, DocumentationContextOptions{})
	if len(context.Snippets) != 0 {
		t.Fatalf("snippets = %#v", context.Snippets)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "documentation.snippet_credential" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestArtifactSetMarshalJSONIncludesDocumentationContext(t *testing.T) {
	data, err := json.Marshal(ArtifactSet{
		DocumentationContext: &DocumentationContext{
			Snippets: []DocumentationSnippet{{Title: "Example", SourceURL: "https://example.com"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "documentation_context") || !strings.Contains(string(data), "Example") {
		t.Fatalf("json = %s", data)
	}
}
