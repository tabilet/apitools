package apitools

import (
	"context"
	"strings"
	"testing"
)

func TestBuildOperationInventoryOpenAPI3(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Query: "create ticket",
		Documents: []InventoryDocument{{
			Name:    "support",
			Path:    "openapi/support.yaml",
			Content: []byte(openAPI3InventoryFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Documents) != 1 || inventory.Documents[0].OperationCount != 2 {
		t.Fatalf("documents = %#v", inventory.Documents)
	}
	if len(inventory.Operations) != 2 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	got := inventory.Operations[0]
	if got.OperationID != "createTicket" || got.Method != "POST" || got.Path != "/tickets" {
		t.Fatalf("first operation = %#v", got)
	}
	if got.Score == 0 {
		t.Fatalf("expected positive score for matched operation")
	}
	if len(got.Parameters) != 1 || got.Parameters[0].Name != "tenant_id" || !got.Parameters[0].Required {
		t.Fatalf("parameters = %#v", got.Parameters)
	}
	if got.RequestBody == nil || got.RequestBody.Schema == nil {
		t.Fatalf("missing request body: %#v", got.RequestBody)
	}
	if len(got.RequestBody.Schema.Properties) != 2 || got.RequestBody.Schema.Properties[0].Name != "priority" {
		t.Fatalf("schema properties = %#v", got.RequestBody.Schema.Properties)
	}
	if len(got.Security) != 1 || got.Security[0].Name != "apiKeyAuth" || got.Security[0].In != "header" {
		t.Fatalf("security = %#v", got.Security)
	}
}

func TestBuildOperationInventorySwagger2(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name:    "legacy",
			Path:    "openapi/legacy.yaml",
			Content: []byte(swagger2InventoryFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	got := inventory.Operations[0]
	if got.OperationID != "getWidget" || got.Method != "GET" {
		t.Fatalf("operation = %#v", got)
	}
	if len(got.Parameters) != 1 || got.Parameters[0].Name != "id" || got.Parameters[0].Type != "string" {
		t.Fatalf("parameters = %#v", got.Parameters)
	}
	if len(got.Security) != 1 || got.Security[0].Name != "api_key" || got.Security[0].Type != "apiKey" {
		t.Fatalf("security = %#v", got.Security)
	}
}

func TestBuildOperationInventoryLimitAndPromptSafety(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Query: "mail",
		Limit: 1,
		Documents: []InventoryDocument{{
			Name:    "mail",
			Content: []byte(openAPI3SafetyFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	got := inventory.Operations[0]
	if got.OperationID != "sendMail" {
		t.Fatalf("operation = %#v", got)
	}
	text := inventoryText(inventory)
	for _, forbidden := range []string{"sk_live_secret", "Bearer token-value", "example@example.com"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("inventory included prompt-unsafe value %q in:\n%s", forbidden, text)
		}
	}
}

func TestBuildOperationInventoryReportsMissingOperationIDAndRefs(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name:    "refs",
			Content: []byte(openAPI3RefFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	got := inventory.Operations[0]
	if got.OperationID != "" || got.ID == "" {
		t.Fatalf("operation id handling = %#v", got)
	}
	if len(got.ReadinessIssues) < 2 {
		t.Fatalf("readiness issues = %#v", got.ReadinessIssues)
	}
	if len(inventory.ReadinessIssues) == 0 {
		t.Fatalf("inventory readiness issues missing")
	}
	if !hasReadinessIssue(got.ReadinessIssues, "schema.ref_unresolved", "#/components/parameters/PathTenant") {
		t.Fatalf("operation did not report path-level parameter ref: %#v", got.ReadinessIssues)
	}
	if !hasReadinessIssue(inventory.ReadinessIssues, "schema.ref_unresolved", "#/components/parameters/PathTenant") {
		t.Fatalf("inventory did not report path-level parameter ref: %#v", inventory.ReadinessIssues)
	}
}

func TestBuildOperationInventoryRequestBodyFieldsAreRecursiveAndPromptSafe(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name:    "nested",
			Content: []byte(openAPI3NestedRequestFixture()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	body := inventory.Operations[0].RequestBody
	if body == nil {
		t.Fatalf("missing request body")
	}
	var paths []string
	for _, field := range body.Fields {
		paths = append(paths, field.Path)
	}
	joined := strings.Join(paths, ",")
	for _, expected := range []string{"user", "user.email", "user.profile", "user.profile.display_name", "groups[]", "groups[].name"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in fields %#v", expected, body.Fields)
		}
	}
	for _, forbidden := range []string{"password", "api_key", "token"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("secret-like field %q leaked in %#v", forbidden, body.Fields)
		}
	}
	if got := strings.Join(body.RequiredFieldPaths, ","); !strings.Contains(got, "user.email") || strings.Contains(got, "user.password") {
		t.Fatalf("required paths = %#v", body.RequiredFieldPaths)
	}
}

func inventoryText(inventory OperationInventory) string {
	var b strings.Builder
	for _, op := range inventory.Operations {
		b.WriteString(op.OperationID)
		b.WriteString(op.Summary)
		b.WriteString(op.Description)
		for _, parameter := range op.Parameters {
			b.WriteString(parameter.Name)
			b.WriteString(parameter.Description)
		}
		if op.RequestBody != nil && op.RequestBody.Schema != nil {
			b.WriteString(op.RequestBody.Schema.Description)
			for _, property := range op.RequestBody.Schema.Properties {
				b.WriteString(property.Name)
				b.WriteString(property.Description)
			}
		}
	}
	return b.String()
}

func hasReadinessIssue(issues []ReadinessIssue, code, path string) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.Path == path {
			return true
		}
	}
	return false
}

func openAPI3InventoryFixture() string {
	return `openapi: 3.0.0
info:
  title: Support API
  version: 1.0.0
  description: Manage support tickets.
components:
  securitySchemes:
    apiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - apiKeyAuth: []
paths:
  /tickets:
    parameters:
      - name: tenant_id
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: listTickets
      summary: List tickets
      responses:
        "200":
          description: ok
    post:
      operationId: createTicket
      summary: Create a support ticket
      description: Create ticket records for support triage.
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [subject, priority]
              properties:
                subject:
                  type: string
                  description: Ticket subject.
                priority:
                  type: string
                  enum: [low, high]
      responses:
        "201":
          description: created
`
}

func swagger2InventoryFixture() string {
	return `swagger: "2.0"
info:
  title: Legacy API
  version: 1.0.0
securityDefinitions:
  api_key:
    type: apiKey
    in: header
    name: X-API-Key
security:
  - api_key: []
paths:
  /widgets/{id}:
    get:
      operationId: getWidget
      parameters:
        - name: id
          in: path
          required: true
          type: string
      responses:
        "200":
          description: ok
`
}

func openAPI3SafetyFixture() string {
	return `openapi: 3.0.0
info:
  title: Mail API
  version: 1.0.0
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
paths:
  /mail:
    post:
      operationId: sendMail
      summary: Send mail
      parameters:
        - name: authorization
          in: header
          required: true
          schema:
            type: string
            default: Bearer token-value
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                to:
                  type: string
                  example: example@example.com
                token:
                  type: string
                  default: sk_live_secret
      responses:
        "200":
          description: ok
  /status:
    get:
      operationId: getStatus
      summary: Fetch status
      responses:
        "200":
          description: ok
`
}

func openAPI3RefFixture() string {
	return `openapi: 3.0.0
info:
  title: Ref API
  version: 1.0.0
paths:
  /items:
    parameters:
      - $ref: "#/components/parameters/PathTenant"
    post:
      parameters:
        - $ref: "#/components/parameters/Tenant"
      requestBody:
        $ref: "#/components/requestBodies/Item"
      responses:
        "200":
          description: ok
`
}

func openAPI3NestedRequestFixture() string {
	return `openapi: 3.0.0
info:
  title: Nested API
  version: 1.0.0
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [user, groups]
              properties:
                user:
                  type: object
                  required: [email, password]
                  properties:
                    email:
                      type: string
                    password:
                      type: string
                    profile:
                      type: object
                      properties:
                        display_name:
                          type: string
                        api_key:
                          type: string
                groups:
                  type: array
                  items:
                    type: object
                    required: [name]
                    properties:
                      name:
                        type: string
                      token:
                        type: string
      responses:
        "200":
          description: ok
`
}
