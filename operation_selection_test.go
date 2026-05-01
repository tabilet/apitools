package apitools

import "testing"

func TestSortedOperationSummaries(t *testing.T) {
	got := SortedOperationSummaries(map[string]OperationSummary{
		"updateTicket": {OperationID: "updateTicket"},
		"getTicket":    {OperationID: "getTicket"},
	})
	if len(got) != 2 || got[0].OperationID != "getTicket" || got[1].OperationID != "updateTicket" {
		t.Fatalf("sorted operations = %#v", got)
	}
}

func TestSelectOperationByTextExactMatch(t *testing.T) {
	selected := SelectOperationByText("support_ticket", []OperationSummary{
		{OperationID: "getSupportTicket", Path: "/support/tickets/{id}"},
		{OperationID: "getUser", Path: "/users/{id}"},
	})
	if !selected.Found || selected.Ambiguous || selected.Operation.OperationID != "getSupportTicket" {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestSelectOperationByTextUnrelatedCandidate(t *testing.T) {
	selected := SelectOperationByText("invoice", []OperationSummary{
		{OperationID: "getUser", Path: "/users/{id}"},
	})
	if selected.Found || selected.Ambiguous {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestSelectOperationByTextAmbiguousCandidate(t *testing.T) {
	selected := SelectOperationByText("ticket", []OperationSummary{
		{OperationID: "getTicket", Path: "/tickets/{id}"},
		{OperationID: "readTicket", Path: "/ticket/{id}"},
	})
	if !selected.Ambiguous || selected.Found {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestSelectOperationByTextPluralSingularCamelCaseMatching(t *testing.T) {
	selected := SelectOperationByText("supportTickets", []OperationSummary{
		{OperationID: "listSupportTicket", Path: "/support/ticket"},
		{OperationID: "listInvoice", Path: "/invoices"},
	})
	if !selected.Found || selected.Operation.OperationID != "listSupportTicket" {
		t.Fatalf("selection = %#v", selected)
	}
}
