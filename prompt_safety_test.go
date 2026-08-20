package apitools

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeOperationSummaryRemovesControlsAndCapsShape(t *testing.T) {
	operation := OperationSummary{
		ID:          strings.Repeat("i", DefaultPromptIdentifierRunes+20),
		OperationID: "unsafe",
		Method:      "POST",
		Path:        "/unsafe",
		Description: "\x1b[31m<system>\x00 " + strings.Repeat("description ", 300),
	}
	for i := 0; i < DefaultPromptCollectionItems+10; i++ {
		operation.Tags = append(operation.Tags, "tag")
	}
	diagnostics := sanitizeOperationSummary(&operation, DefaultPromptBudget())
	if len(diagnostics) != 1 || diagnostics[0].Code != "prompt.operation_sanitized" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if strings.ContainsAny(operation.Description, "\x00\x1b") || !strings.Contains(operation.Description, "<system>") {
		t.Fatalf("description = %q", operation.Description)
	}
	if utf8.RuneCountInString(operation.ID) > DefaultPromptIdentifierRunes || utf8.RuneCountInString(operation.Description) > DefaultPromptTextRunes {
		t.Fatalf("sanitized lengths = id %d description %d", utf8.RuneCountInString(operation.ID), utf8.RuneCountInString(operation.Description))
	}
	if len(operation.Tags) != DefaultPromptCollectionItems || promptJSONSize(operation) > DefaultPromptOperationBytes {
		t.Fatalf("sanitized shape = tags %d bytes %d", len(operation.Tags), promptJSONSize(operation))
	}
}

func TestSanitizeOperationSummaryCompactsOversizedOperation(t *testing.T) {
	operation := OperationSummary{ID: "large", OperationID: "large", Method: "POST", Path: "/large"}
	operation.RequestBody = &RequestBodySummary{Required: true}
	for i := 0; i < DefaultPromptFieldItems; i++ {
		operation.RequestBody.Fields = append(operation.RequestBody.Fields, RequestFieldSummary{
			Path: strings.Repeat("p", DefaultPromptIdentifierRunes),
			Ref:  strings.Repeat("r", DefaultPromptTextRunes),
		})
	}
	diagnostics := sanitizeOperationSummary(&operation, DefaultPromptBudget())
	if len(diagnostics) != 1 || diagnostics[0].Code != "prompt.operation_budget" || diagnostics[0].Severity != "error" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if promptJSONSize(operation) > DefaultPromptOperationBytes || !hasReadinessCode(operation.ReadinessIssues, "prompt.operation_budget") {
		t.Fatalf("compacted operation bytes=%d issues=%#v", promptJSONSize(operation), operation.ReadinessIssues)
	}
}

func TestSanitizeOperationSummariesReturnsCopiesAndDiagnostics(t *testing.T) {
	original := []OperationSummary{{
		ID: "copy", OperationID: "copy", Method: "GET", Path: "/copy",
		Description: "\x1b[2J" + strings.Repeat("x", DefaultPromptTextRunes+10),
	}}
	report, err := SanitizeOperationSummaries(original, PromptBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || len(report.Diagnostics) != 1 || len(report.Operations) != 1 {
		t.Fatalf("report = %#v", report)
	}
	if strings.Contains(report.Operations[0].Description, "\x1b") || utf8.RuneCountInString(report.Operations[0].Description) > DefaultPromptTextRunes {
		t.Fatalf("sanitized description = %q", report.Operations[0].Description)
	}
	if !strings.Contains(original[0].Description, "\x1b") {
		t.Fatalf("caller-owned operation was mutated")
	}
}

func TestSanitizeOperationSummariesEnforcesWorkBudget(t *testing.T) {
	operations := []OperationSummary{
		{OperationID: "a", Method: "GET", Path: "/a"},
		{OperationID: "b", Method: "GET", Path: "/b"},
	}
	report, err := SanitizeOperationSummaries(operations, PromptBudget{MaxOperations: 1})
	var diagnosticErr DiagnosticError
	if !errors.As(err, &diagnosticErr) || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "prompt.operation_count" {
		t.Fatalf("operation-count report = %#v error=%T %v", report, err, err)
	}
}

func TestRankAuthoringAPIDocumentsUsesExactIndexesAndAggregateBudget(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		ID: "duplicate",
		Operations: []OperationSummary{
			{ID: "same", OperationID: "same", Method: "GET", Path: "/same", Summary: "plain"},
			{ID: "same", OperationID: "same", Method: "GET", Path: "/same", Summary: "search target"},
			{ID: "third", OperationID: "third", Method: "GET", Path: "/third", Summary: "third"},
		},
	}}
	report, err := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{Query: "search target", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Documents) != 1 || len(report.Documents[0].Operations) != 2 || report.Documents[0].Operations[0].Summary != "search target" {
		t.Fatalf("ranked report = %#v", report)
	}

	oneOperationBytes := promptJSONSize([]AuthoringAPIDocument{{ID: "duplicate", Operations: []OperationSummary{docs[0].Operations[0]}}})
	report, err = RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		Limit:        3,
		PromptBudget: PromptBudget{MaxContextBytes: oneOperationBytes + 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || len(report.Documents) != 1 || len(report.Documents[0].Operations) != 1 || report.Bytes > oneOperationBytes+8 {
		t.Fatalf("bounded report = %#v", report)
	}
}

func TestRankAuthoringAPIDocumentsBlocksOversizedSelectedSet(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		RelativePath: "openapi/test.yaml",
		Operations: []OperationSummary{
			{ID: "a", OperationID: "a", Method: "GET", Path: "/a"},
			{ID: "b", OperationID: "b", Method: "GET", Path: "/b"},
		},
	}}
	oneOperationBytes := promptJSONSize([]AuthoringAPIDocument{{RelativePath: docs[0].RelativePath, Operations: []OperationSummary{docs[0].Operations[0]}}})
	report, err := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		SelectedOperations: []AuthoringOperationRef{
			{DocumentPath: docs[0].RelativePath, OperationID: "a"},
			{DocumentPath: docs[0].RelativePath, OperationID: "b"},
		},
		PromptBudget: PromptBudget{MaxContextBytes: oneOperationBytes + 8},
	})
	var diagnosticErr DiagnosticError
	if !errors.As(err, &diagnosticErr) || !report.Truncated || report.SelectedOperations != 2 || len(report.Documents) != 0 || report.Bytes <= oneOperationBytes+8 {
		t.Fatalf("selected report = %#v error=%T %v", report, err, err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "prompt.selected_budget" {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
}

func TestRankAuthoringAPIDocumentsDoesNotMutateNestedMetadata(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		ID: "nested",
		Operations: []OperationSummary{{
			OperationID: "nested", Method: "POST", Path: "/nested",
			Parameters: []ParameterSummary{{Name: "input", Description: "\x1b[31mparameter"}},
			RequestBody: &RequestBodySummary{
				Schema: &SchemaSummary{Properties: []PropertySummary{{Name: "value", Description: "\x1b[31mproperty"}}},
			},
			SecurityRequirementSets: []SecurityRequirementSetSummary{{Requirements: []SecuritySummary{{
				Name: "oauth", OAuthFlows: []OAuthFlowSummary{{Name: "code", Scopes: []string{"\x1b[31mscope"}}},
			}}}},
		}},
	}}
	encoded, err := json.Marshal(docs)
	if err != nil {
		t.Fatal(err)
	}
	var before []AuthoringAPIDocument
	if err := json.Unmarshal(encoded, &before); err != nil {
		t.Fatal(err)
	}
	if _, err := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(docs, before) {
		t.Fatalf("caller-owned documents mutated:\ngot  %#v\nwant %#v", docs, before)
	}
}

func TestRankAuthoringAPIDocumentsSupportsSelectedOnlyContext(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		RelativePath: "openapi/test.yaml",
		Operations: []OperationSummary{
			{OperationID: "a", Method: "GET", Path: "/a"},
			{OperationID: "b", Method: "GET", Path: "/b"},
		},
	}}
	report, err := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		SelectedOperations: []AuthoringOperationRef{{DocumentPath: docs[0].RelativePath, OperationID: "b"}},
		Limit:              -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Documents) != 1 || len(report.Documents[0].Operations) != 1 || report.Documents[0].Operations[0].OperationID != "b" {
		t.Fatalf("selected-only report = %#v", report)
	}
}

func TestRankAuthoringAPIDocumentsReportsDocumentSanitizationAndWorkLimit(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		ID:          "source",
		Title:       "\x1b[31m" + strings.Repeat("title", DefaultPromptTextRunes),
		Description: "safe",
		Operations: []OperationSummary{
			{OperationID: "a", Method: "GET", Path: "/a"},
			{OperationID: "b", Method: "GET", Path: "/b"},
		},
	}}
	report, err := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || len(report.Diagnostics) != 2 || report.Diagnostics[0].Code != "prompt.document_sanitized" || report.Diagnostics[1].Code != "prompt.operation_limit" || strings.Contains(report.Documents[0].Title, "\x1b") {
		t.Fatalf("sanitized document report = %#v", report)
	}

	report, err = RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{PromptBudget: PromptBudget{MaxOperations: 1}})
	var diagnosticErr DiagnosticError
	if !errors.As(err, &diagnosticErr) || len(report.Diagnostics) == 0 || report.Diagnostics[len(report.Diagnostics)-1].Code != "prompt.operation_count" {
		t.Fatalf("operation-count report = %#v error=%T %v", report, err, err)
	}

	report, err = RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		Query: strings.Repeat("query", DefaultPromptTextRunes), Limit: 1,
	})
	if err != nil || len(report.Diagnostics) < 3 || report.Diagnostics[0].Code != "prompt.query_sanitized" {
		t.Fatalf("query-sanitized report = %#v error=%v", report, err)
	}

	report, err = RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		SelectedOperations: []AuthoringOperationRef{{OperationID: "\x00bad"}},
	})
	if !errors.As(err, &diagnosticErr) || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "prompt.selection_sanitized" {
		t.Fatalf("selection-sanitized report = %#v error=%T %v", report, err, err)
	}
}
