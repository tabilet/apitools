package apitools

import (
	"context"
	"fmt"
	"strings"
)

const (
	defaultDocumentationMaxSnippets = 5
	defaultDocumentationMaxContent  = 600
)

// DocumentationContextProvider fetches advisory documentation context for an
// authoring request. Returned snippets are not authoritative; OpenAPI remains
// the contract for operations, schemas, and security declarations.
type DocumentationContextProvider interface {
	FetchDocumentationContext(ctx context.Context, query DocumentationContextQuery) (DocumentationContext, error)
}

// DocumentationContextQuery describes the prompt-safe context a provider may
// use to retrieve advisory documentation snippets.
type DocumentationContextQuery struct {
	Brief       string             `json:"brief,omitempty"`
	ProjectName string             `json:"project_name,omitempty"`
	LibraryIDs  []string           `json:"library_ids,omitempty"`
	Documents   []OpenAPIDoc       `json:"documents,omitempty"`
	Operations  []OperationSummary `json:"operations,omitempty"`
	MaxSnippets int                `json:"max_snippets,omitempty"`
}

// DocumentationContext contains advisory documentation snippets and provider
// diagnostics. It must not be used as a substitute for OpenAPI validation.
type DocumentationContext struct {
	Snippets    []DocumentationSnippet `json:"snippets,omitempty"`
	Diagnostics []Diagnostic           `json:"diagnostics,omitempty"`
}

// DocumentationSnippet is a compact, provenance-preserving documentation or
// code excerpt that may help reviewers understand a draft.
type DocumentationSnippet struct {
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	Source    string `json:"source,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Language  string `json:"language,omitempty"`
	LibraryID string `json:"library_id,omitempty"`
}

// DocumentationContextOptions controls snippet sanitization.
type DocumentationContextOptions struct {
	MaxSnippets int `json:"max_snippets,omitempty"`
	MaxContent  int `json:"max_content,omitempty"`
}

// SanitizeDocumentationContext caps advisory snippets, removes likely
// credential-bearing content, trims fields, and returns non-blocking
// diagnostics for dropped snippets.
func SanitizeDocumentationContext(ctx DocumentationContext, opts DocumentationContextOptions) (DocumentationContext, []Diagnostic) {
	maxSnippets := opts.MaxSnippets
	if maxSnippets <= 0 {
		maxSnippets = defaultDocumentationMaxSnippets
	}
	maxContent := opts.MaxContent
	if maxContent <= 0 {
		maxContent = defaultDocumentationMaxContent
	}
	out := DocumentationContext{
		Diagnostics: append([]Diagnostic(nil), ctx.Diagnostics...),
	}
	var diagnostics []Diagnostic
	for i, snippet := range ctx.Snippets {
		if len(out.Snippets) >= maxSnippets {
			diagnostics = append(diagnostics, Diagnostic{
				Severity:    "warning",
				Code:        "documentation.snippet_limit",
				Message:     fmt.Sprintf("advisory documentation snippet limit of %d was reached", maxSnippets),
				Remediation: "Increase the advisory snippet limit only if reviewers need more context.",
			})
			break
		}
		clean := sanitizeDocumentationSnippet(snippet, maxContent)
		if clean.Content == "" && clean.Title == "" {
			continue
		}
		if ContainsLikelyCredentialValue([]byte(clean.Content)) || ContainsLikelyCredentialValue([]byte(clean.Title)) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity:    "warning",
				Code:        "documentation.snippet_credential",
				Message:     "advisory documentation snippet was dropped because it appears to contain a literal credential value",
				Path:        fmt.Sprintf("documentation.snippets[%d]", i),
				Remediation: "Fetch or provide documentation snippets without credential-like literals.",
			})
			continue
		}
		out.Snippets = append(out.Snippets, clean)
	}
	out.Diagnostics = append(out.Diagnostics, diagnostics...)
	return out, diagnostics
}

func sanitizeDocumentationSnippet(snippet DocumentationSnippet, maxContent int) DocumentationSnippet {
	clean := DocumentationSnippet{
		Title:     oneLine(snippet.Title),
		Content:   collapseWhitespace(snippet.Content),
		SourceURL: strings.TrimSpace(snippet.SourceURL),
		Source:    oneLine(snippet.Source),
		Kind:      oneLine(snippet.Kind),
		Language:  oneLine(snippet.Language),
		LibraryID: oneLine(snippet.LibraryID),
	}
	if maxContent > 0 && len(clean.Content) > maxContent {
		clean.Content = strings.TrimSpace(clean.Content[:maxContent]) + "..."
	}
	return clean
}

func oneLine(value string) string {
	return collapseWhitespace(value)
}

func collapseWhitespace(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	return strings.Join(fields, " ")
}
