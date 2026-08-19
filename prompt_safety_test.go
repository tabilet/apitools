package apitools

import (
	"errors"
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
	if !errors.As(err, &diagnosticErr) || report.SelectedOperations != 2 || len(report.Documents) != 0 || report.Bytes <= oneOperationBytes+8 {
		t.Fatalf("selected report = %#v error=%T %v", report, err, err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "prompt.selected_budget" {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
}
