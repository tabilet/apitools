package apitools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildAuthoringAPIDocumentsGroupsPromptSafeOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openapi", "support.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`openapi: 3.0.0
info:
  title: Support API
  description: Ticket operations
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      summary: Get ticket
      security:
        - ApiKeyAuth: []
      parameters:
        - name: ticketId
          in: path
          required: true
          schema: {type: string}
  /tickets:
    post:
      operationId: createTicket
      summary: Create ticket
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [customer]
              properties:
                customer:
                  type: object
                  required: [email]
                  properties:
                    email:
                      type: string
                      example: ada@example.com
`), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := BuildAuthoringAPIDocuments(context.Background(), AuthoringAPIDocumentOptions{
		Documents: []InventoryDocument{{Name: "support", Path: path}},
		BaseDir:   dir,
		Query:     "create ticket",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].ID != "support" || docs[0].RelativePath != "openapi/support.yaml" || len(docs[0].Operations) != 2 {
		t.Fatalf("docs = %#v", docs)
	}
	var create OperationSummary
	for _, op := range docs[0].Operations {
		if op.OperationID == "createTicket" {
			create = op
		}
	}
	if got := strings.Join(RequiredOperationFields(create), ","); got != "customer.email" {
		t.Fatalf("required fields = %q", got)
	}
	bodyText := inventoryJSON(t, create.RequestBody)
	if strings.Contains(bodyText, "ada@example.com") {
		t.Fatalf("request body leaked example: %s", bodyText)
	}
	var get OperationSummary
	for _, op := range docs[0].Operations {
		if op.OperationID == "getTicket" {
			get = op
		}
	}
	if !reflect.DeepEqual(OperationCredentialFields(get), []string{"api_key_auth"}) {
		t.Fatalf("credential fields = %#v", OperationCredentialFields(get))
	}
}

func TestRankAuthoringAPIDocumentsCapsAndPreservesSelected(t *testing.T) {
	docs := []AuthoringAPIDocument{{
		RelativePath: "openapi/support.yaml",
		Operations: []OperationSummary{
			{OperationID: "getTicket", Method: "GET", Path: "/tickets/{id}"},
			{OperationID: "searchTickets", Method: "GET", Path: "/tickets/search", Summary: "Search tickets"},
			{OperationID: "closeTicket", Method: "POST", Path: "/tickets/{id}/close"},
		},
	}}
	ranked := RankAuthoringAPIDocuments(docs, AuthoringOperationRankingOptions{
		Query:              "search support tickets",
		Limit:              1,
		SelectedOperations: []AuthoringOperationRef{{DocumentPath: "openapi/support.yaml", OperationID: "closeTicket"}},
	})
	if got := []string{ranked[0].Operations[0].OperationID, ranked[0].Operations[1].OperationID}; !reflect.DeepEqual(got, []string{"closeTicket", "searchTickets"}) {
		t.Fatalf("ranked operations = %#v", got)
	}
}

func inventoryJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
