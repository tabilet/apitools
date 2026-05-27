package apitools

import "testing"

func TestParseGraphQLOperationSummaries(t *testing.T) {
	ops, err := ParseGraphQLOperationSummaries([]byte(`
type Query {
  issue(id: ID!): Issue
}

mutation CreateIssue($title: String!) {
  createIssue(title: $title) { id }
}
`), "graphql/issues.graphql")
	if err != nil {
		t.Fatalf("ParseGraphQLOperationSummaries() error = %v", err)
	}
	if len(ops) != 2 {
		t.Fatalf("operation count = %d, want 2", len(ops))
	}
	byID := map[string]OperationSummary{}
	for _, op := range ops {
		byID[op.ID] = op
	}
	if byID["mutation.CreateIssue"].Method != "mutation" {
		t.Fatalf("mutation summary = %#v", byID["mutation.CreateIssue"])
	}
	if byID["mutation.CreateIssue"].Parameters[0].In != "graphql-variable" || byID["mutation.CreateIssue"].Parameters[0].Name != "title" {
		t.Fatalf("mutation parameters = %#v", byID["mutation.CreateIssue"].Parameters)
	}
	query := byID["query.issue"]
	if query.Path != "#/schema/Query/fields/issue" || query.Provenance != "graphql" {
		t.Fatalf("query summary = %#v", query)
	}
	if query.Extensions["source_operation_id"] != "query.issue" || query.Extensions["graphql_root_type"] != "Query" {
		t.Fatalf("query extensions = %#v", query.Extensions)
	}
}
