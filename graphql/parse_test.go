package graphql

import (
	"bytes"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/internal/sourceguard"
)

func TestParseIntrospectionPreservesSchemaRootOperations(t *testing.T) {
	model, err := Parse([]byte(`{
  "data": {
    "__schema": {
      "queryType": {"name": "Query"},
      "mutationType": {"name": "Mutation"},
      "types": [
        {
          "kind": "OBJECT",
          "name": "Query",
          "fields": [
            {
              "name": "node",
              "description": "Fetch a node.",
              "args": [{"name": "id", "type": {"kind": "NON_NULL", "ofType": {"kind": "SCALAR", "name": "ID"}}}],
              "type": {"kind": "OBJECT", "name": "Node"}
            }
          ]
        },
        {
          "kind": "OBJECT",
          "name": "Mutation",
          "fields": [
            {"name": "createIssue", "type": {"kind": "OBJECT", "name": "Issue"}}
          ]
        }
      ]
    }
  }
}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindIntrospectionJSON {
		t.Fatalf("source kind = %q, want introspection", model.SourceKind)
	}
	summaries := model.OperationSummaries()
	if len(summaries) != 2 {
		t.Fatalf("operation summaries = %d, want 2: %#v", len(summaries), summaries)
	}
	if summaries[0].ID != "mutation.createIssue" || summaries[1].ID != "query.node" {
		t.Fatalf("operation ids = %#v, want deterministic mutation/query ids", summaries)
	}
	query, ok := model.OperationByID("query.node")
	if !ok {
		t.Fatalf("missing query.node operation")
	}
	if query.Selector != "#/schema/Query/fields/node" || query.FieldType.Display != "Node" {
		t.Fatalf("query metadata = %#v", query)
	}
	target, err := model.ResolveSelector("query.node")
	if err != nil {
		t.Fatalf("ResolveSelector() error = %v", err)
	}
	if target.Kind != SelectorKindOperation || target.Summary.SourceOperationRef != "#/schema/Query/fields/node" {
		t.Fatalf("selector target = %#v", target)
	}
}

func TestParseSDLAndOperationDocument(t *testing.T) {
	model, err := Parse([]byte(`
schema {
  query: RootQuery
  mutation: RootMutation
}

type RootQuery {
  """Find one issue."""
  issue(id: ID!): Issue
}

type RootMutation {
  createIssue(input: IssueInput!): Issue
}

input IssueInput {
  title: String!
}

query GetIssue($id: ID!) {
  issue(id: $id) { id title }
}
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindSDL {
		t.Fatalf("source kind = %q, want SDL", model.SourceKind)
	}
	for _, id := range []string{"mutation.createIssue", "query.GetIssue", "query.issue"} {
		if _, ok := model.OperationByID(id); !ok {
			t.Fatalf("missing operation %q in %#v", id, model.OperationSummaries())
		}
	}
	op, _ := model.OperationByID("query.GetIssue")
	if len(op.Variables) != 1 || op.Variables[0].Name != "id" || op.Variables[0].Type.Display != "ID!" {
		t.Fatalf("operation variables = %#v", op.Variables)
	}
	if len(op.SelectionNames) != 1 || op.SelectionNames[0] != "issue" {
		t.Fatalf("selection names = %#v", op.SelectionNames)
	}
}

func TestDetectLocalArtifact(t *testing.T) {
	detection, ok := DetectLocalArtifact([]byte(`query FindViewer { viewer { login } }`), "ops.graphql")
	if !ok {
		t.Fatalf("DetectLocalArtifact() did not detect operation document")
	}
	if detection.Kind != ArtifactKindOperationDocument || detection.Format != "graphql" {
		t.Fatalf("detection = %#v", detection)
	}
	if _, ok := DetectLocalArtifact([]byte(`{"title":"not graphql"}`), "spec.json"); ok {
		t.Fatalf("non-GraphQL JSON detected")
	}
}

func TestParseRejectsMalformedGraphQL(t *testing.T) {
	if _, err := Parse([]byte(`not graphql`)); err == nil {
		t.Fatalf("Parse() error = nil, want malformed GraphQL error")
	}
	if _, err := Parse([]byte(`query`)); err == nil {
		t.Fatalf("Parse() error = nil, want incomplete operation error")
	}
}

func TestParseRejectsResourceLimitViolations(t *testing.T) {
	t.Run("document bytes", func(t *testing.T) {
		_, err := Parse(bytes.Repeat([]byte{'x'}, sourceguard.MaxDocumentBytes+1))
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("Parse() error = %v, want maximum size", err)
		}
	})
	t.Run("text type nesting", func(t *testing.T) {
		input := `query Q($v: ` + strings.Repeat(`[`, sourceguard.MaxNestingDepth+1) + `String` + strings.Repeat(`]`, sourceguard.MaxNestingDepth+1) + `) { viewer }`
		_, err := Parse([]byte(input))
		if err == nil || !strings.Contains(err.Error(), "nesting") {
			t.Fatalf("Parse() error = %v, want nesting limit", err)
		}
	})
	t.Run("introspection nesting", func(t *testing.T) {
		ref := `{"kind":"SCALAR","name":"String"}`
		for range sourceguard.MaxNestingDepth {
			ref = `{"kind":"LIST","ofType":` + ref + `}`
		}
		input := `{"data":{"__schema":{"types":[{"kind":"OBJECT","name":"Query","fields":[{"name":"x","type":` + ref + `}]}]}}}`
		input = strings.ReplaceAll(input, `\"`, `"`)
		_, err := Parse([]byte(input))
		if err == nil || !strings.Contains(err.Error(), "JSON nesting") {
			t.Fatalf("Parse() error = %v, want JSON nesting limit", err)
		}
	})
}

func FuzzParse(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`type Query { viewer: User }`),
		[]byte(`query Viewer { viewer { id } }`),
		[]byte(`{"data":{"__schema":{"types":[]}}}`),
		[]byte(strings.Repeat(`[`, sourceguard.MaxNestingDepth+1)),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
