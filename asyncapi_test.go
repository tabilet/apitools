package apitools

import (
	"testing"

	"github.com/OpenUdon/asyncapi"
)

func TestParseAsyncAPIOperationSummariesFromYAML(t *testing.T) {
	ops, err := ParseAsyncAPIOperationSummaries([]byte(`asyncapi: 3.0.0
info:
  title: Billing Events
  description: Billing event stream.
operations:
  publishInvoice:
    action: send
    summary: Publish invoice
    description: Sends an invoice event.
    channel:
      $ref: '#/channels/invoices'
    messages:
      - $ref: '#/channels/invoices/messages/invoiceCreated'
channels:
  invoices:
    address: billing.invoices
    messages:
      invoiceCreated:
        payload:
          type: object
`), "asyncapi/billing.yaml")
	if err != nil {
		t.Fatalf("ParseAsyncAPIOperationSummaries() error = %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("operation count = %d, want 1", len(ops))
	}
	op := ops[0]
	if op.ID != "publishInvoice" || op.OperationID != "publishInvoice" {
		t.Fatalf("operation ids = %q %q, want publishInvoice", op.ID, op.OperationID)
	}
	if op.DocumentName != "Billing Events" || op.DocumentPath != "asyncapi/billing.yaml" || op.DocumentRelativePath != "asyncapi/billing.yaml" {
		t.Fatalf("document metadata = %#v", op)
	}
	if op.Method != "send" || op.Path != "#/channels/invoices" {
		t.Fatalf("operation method/path = %q %q, want send #/channels/invoices", op.Method, op.Path)
	}
	if op.Summary != "Publish invoice" || op.Description != "Sends an invoice event." {
		t.Fatalf("operation text = %q %q", op.Summary, op.Description)
	}
	if op.Provenance != "asyncapi" {
		t.Fatalf("provenance = %q, want asyncapi", op.Provenance)
	}
}

func TestParseAsyncAPIOperationSummariesFromJSON(t *testing.T) {
	ops, err := ParseAsyncAPIOperationSummaries([]byte(`{
  "asyncapi": "3.0.0",
  "info": {"title": "Webhook Events"},
  "operations": {
    "receiveCustomer": {
      "action": "receive",
      "summary": "Receive customer",
      "messages": [{"$ref": "#/channels/customers/messages/customerUpdated"}]
    }
  },
  "channels": {
    "customers": {
      "messages": {
        "customerUpdated": {"payload": {"type": "object"}}
      }
    }
  }
}`), "asyncapi/webhooks.json")
	if err != nil {
		t.Fatalf("ParseAsyncAPIOperationSummaries() error = %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("operation count = %d, want 1", len(ops))
	}
	op := ops[0]
	if op.OperationID != "receiveCustomer" || op.Method != "receive" {
		t.Fatalf("operation = %#v", op)
	}
	if op.Path != "#/channels/customers/messages/customerUpdated" {
		t.Fatalf("path = %q, want message ref", op.Path)
	}
}

func TestAsyncAPIOperationSummariesSkipsNilAndBlankOperations(t *testing.T) {
	got := AsyncAPIOperationSummaries("asyncapi/events.yaml", &asyncapi.Model{
		Title: "Events",
		Operations: []*asyncapi.Operation{
			nil,
			{ID: "   ", Action: "send"},
			{ID: "publishOrder", Action: "send"},
		},
	})
	if len(got) != 1 || got[0].OperationID != "publishOrder" {
		t.Fatalf("summaries = %#v, want publishOrder only", got)
	}
}
