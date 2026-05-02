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

func TestSelectOperationByHintsAWSQueryPrefersGETForReadAction(t *testing.T) {
	selected := SelectOperationByHints(OperationSelectionHints{
		Provider:   "aws",
		Purpose:    "read",
		Path:       "/",
		Parameters: map[string]string{"Action": "DescribeInstanceStatus"},
	}, []OperationSummary{
		{OperationID: "GET_DescribeInstanceStatus", Method: "GET", Path: "/#Action=DescribeInstanceStatus"},
		{OperationID: "POST_DescribeInstanceStatus", Method: "POST", Path: "/#Action=DescribeInstanceStatus"},
	})
	if !selected.Found || selected.Ambiguous || selected.Operation.OperationID != "GET_DescribeInstanceStatus" {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestSelectOperationByHintsAWSQueryHonorsExplicitMethod(t *testing.T) {
	selected := SelectOperationByHints(OperationSelectionHints{
		Provider:   "aws",
		Purpose:    "read",
		Method:     "POST",
		Path:       "/",
		Parameters: map[string]string{"Action": "DescribeInstanceStatus"},
	}, []OperationSummary{
		{OperationID: "GET_DescribeInstanceStatus", Method: "GET", Path: "/"},
		{OperationID: "POST_DescribeInstanceStatus", Method: "POST", Path: "/"},
	})
	if !selected.Found || selected.Operation.OperationID != "POST_DescribeInstanceStatus" {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestClassifyOperationPurposeAWSQueryDescribePostIsRead(t *testing.T) {
	got := ClassifyOperationPurpose(OperationSummary{
		OperationID: "POST_DescribeInstanceStatus",
		Method:      "POST",
		Path:        "/",
		Extensions:  map[string]string{"x-aws-operation-name": "DescribeInstanceStatus"},
	}, OperationSelectionHints{Provider: "aws"})
	if got != "read" {
		t.Fatalf("purpose = %q, want read", got)
	}
}

func TestSelectOperationByHintsAWSQueryUsesVendorOperationName(t *testing.T) {
	selected := SelectOperationByHints(OperationSelectionHints{
		Provider:   "aws",
		Purpose:    "read",
		Parameters: map[string]string{"Action": "DescribeInstanceStatus"},
	}, []OperationSummary{
		{OperationID: "ec2DescribeStatus", Method: "GET", Path: "/", Extensions: map[string]string{"x-aws-operation-name": "DescribeInstanceStatus"}},
	})
	if !selected.Found || selected.Operation.OperationID != "ec2DescribeStatus" {
		t.Fatalf("selection = %#v", selected)
	}
}
