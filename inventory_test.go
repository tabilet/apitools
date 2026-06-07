package apitools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
	if got.ResponseBody == nil || got.ResponseBody.StatusCode != "201" || got.ResponseBody.Schema == nil {
		t.Fatalf("missing response body: %#v", got.ResponseBody)
	}
	if names := responseFieldNames(got.ResponseBody.Fields); strings.Join(names, ",") != "id,name,status" {
		t.Fatalf("response fields = %#v", got.ResponseBody.Fields)
	}
	if len(got.Security) != 1 || got.Security[0].Name != "apiKeyAuth" || got.Security[0].In != "header" || got.Security[0].ParameterName != "X-API-Key" {
		t.Fatalf("security = %#v", got.Security)
	}
}

func TestBuildOperationInventoryOAuthFlowURLs(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name: "gmail",
			Content: []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "Gmail", "version": "v1"},
  "paths": {
    "/gmail/v1/users/{userId}/profile": {
      "get": {
        "operationId": "gmail.users.getProfile",
        "security": [
          {"Oauth2c": ["https://www.googleapis.com/auth/gmail.readonly"]}
        ],
        "responses": {"200": {"description": "ok"}}
      }
    }
  },
  "components": {
    "securitySchemes": {
      "Oauth2c": {
        "type": "oauth2",
        "flows": {
          "authorizationCode": {
            "authorizationUrl": "https://accounts.google.com/o/oauth2/auth",
            "tokenUrl": "https://oauth2.googleapis.com/token",
            "refreshUrl": "https://oauth2.googleapis.com/token",
            "scopes": {
              "https://www.googleapis.com/auth/gmail.readonly": "Read Gmail"
            }
          }
        }
      }
    }
  }
}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 || len(inventory.Operations[0].Security) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	security := inventory.Operations[0].Security[0]
	if security.AuthorizationURL != "https://accounts.google.com/o/oauth2/auth" || security.TokenURL != "https://oauth2.googleapis.com/token" || security.RefreshURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("security = %#v", security)
	}
	if len(security.Flows) != 1 || security.Flows[0] != "authorizationCode" {
		t.Fatalf("flows = %#v", security.Flows)
	}
	if len(security.OAuthFlows) != 1 || security.OAuthFlows[0].Name != "authorizationCode" || security.OAuthFlows[0].TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("oauth flows = %#v", security.OAuthFlows)
	}
}

func TestBuildOperationInventoryKeepsOAuthFlowURLsSeparate(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name: "oauth",
			Content: []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "OAuth", "version": "v1"},
  "paths": {
    "/me": {
      "get": {
        "operationId": "getMe",
        "security": [{"Oauth2": ["profile.read"]}],
        "responses": {"200": {"description": "ok"}}
      }
    }
  },
  "components": {
    "securitySchemes": {
      "Oauth2": {
        "type": "oauth2",
        "flows": {
          "authorizationCode": {
            "authorizationUrl": "https://auth.example.com/authorize",
            "tokenUrl": "https://auth.example.com/auth-code-token",
            "refreshUrl": "https://auth.example.com/auth-code-refresh",
            "scopes": {"profile.read": "Read profile"}
          },
          "clientCredentials": {
            "tokenUrl": "https://auth.example.com/client-token",
            "scopes": {"profile.admin": "Admin profile"}
          }
        }
      }
    }
  }
}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Operations) != 1 || len(inventory.Operations[0].Security) != 1 {
		t.Fatalf("operations = %#v", inventory.Operations)
	}
	security := inventory.Operations[0].Security[0]
	if len(security.OAuthFlows) != 2 {
		t.Fatalf("oauth flows = %#v", security.OAuthFlows)
	}
	byName := map[string]OAuthFlowSummary{}
	for _, flow := range security.OAuthFlows {
		byName[flow.Name] = flow
	}
	if byName["authorizationCode"].TokenURL != "https://auth.example.com/auth-code-token" || byName["authorizationCode"].RefreshURL != "https://auth.example.com/auth-code-refresh" {
		t.Fatalf("authorizationCode flow = %#v", byName["authorizationCode"])
	}
	if byName["clientCredentials"].TokenURL != "https://auth.example.com/client-token" {
		t.Fatalf("clientCredentials flow = %#v", byName["clientCredentials"])
	}
	if !strings.Contains(strings.Join(byName["authorizationCode"].Scopes, ","), "profile.read") {
		t.Fatalf("authorizationCode scopes = %#v", byName["authorizationCode"].Scopes)
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
	if len(got.Security) != 1 || got.Security[0].Name != "api_key" || got.Security[0].Type != "apiKey" || got.Security[0].ParameterName != "X-API-Key" {
		t.Fatalf("security = %#v", got.Security)
	}
	if got.ResponseBody == nil || got.ResponseBody.StatusCode != "200" || got.ResponseBody.Schema == nil {
		t.Fatalf("response body = %#v", got.ResponseBody)
	}
	if names := responseFieldNames(got.ResponseBody.Fields); strings.Join(names, ",") != "id,name" {
		t.Fatalf("response fields = %#v", got.ResponseBody.Fields)
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

func TestBuildOperationInventoryResolvesSwaggerBodyRefFields(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{
			Name: "kubernetes",
			Content: []byte(`swagger: "2.0"
info:
  title: Kubernetes
  version: v1
paths:
  /api/v1/namespaces:
    post:
      operationId: createCoreV1Namespace
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: "#/definitions/io.k8s.api.core.v1.Namespace"
      responses:
        "201":
          description: created
definitions:
  io.k8s.api.core.v1.Namespace:
    type: object
    required: [metadata]
    properties:
      metadata:
        type: object
        required: [name]
        properties:
          name:
            type: string
          labels:
            type: object
            additionalProperties:
              type: string
          owner:
            $ref: "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.Owner"
          previousOwner:
            $ref: "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.Owner"
  io.k8s.apimachinery.pkg.apis.meta.v1.Owner:
    type: object
    properties:
      name:
        type: string
`),
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
	for _, expected := range []string{"metadata", "metadata.name", "metadata.labels", "metadata.owner.name", "metadata.previousOwner.name"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in fields %#v", expected, body.Fields)
		}
	}
	if strings.Contains(joined, "body") {
		t.Fatalf("body ref stayed opaque: %#v", body.Fields)
	}
	if got := strings.Join(body.RequiredFieldPaths, ","); !strings.Contains(got, "metadata.name") {
		t.Fatalf("required paths = %#v", body.RequiredFieldPaths)
	}
}

func TestBuildOperationInventoryAllowsRequestBodyWithoutSchema(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Query: "create",
		Documents: []InventoryDocument{{
			Name: "empty-body",
			Content: []byte(`openapi: 3.0.0
info:
  title: Empty Body API
  version: 1.0.0
paths:
  /tickets:
    post:
      operationId: createTicket
      requestBody:
        description: Body is intentionally unspecified.
        content: {}
      responses:
        "200":
          description: ok
`),
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
	if body.Schema != nil {
		t.Fatalf("schema = %#v, want nil", body.Schema)
	}
}

func TestBuildOperationInventoryReportsMalformedYAML(t *testing.T) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{Name: "bad", Content: []byte("openapi: [")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Diagnostics) != 1 || inventory.Diagnostics[0].Code != "document.parse" {
		t.Fatalf("diagnostics = %#v", inventory.Diagnostics)
	}
}

func TestBuildOperationInventoryReportsReadFailuresForUnsafePaths(t *testing.T) {
	base := t.TempDir()
	symlinkTarget := filepath.Join(base, "target.yaml")
	writeLocalFile(t, symlinkTarget, openAPI3InventoryFixture())
	symlinkPath := filepath.Join(base, "symlink.yaml")
	symlinkOrSkip(t, symlinkTarget, symlinkPath)

	dirPath := filepath.Join(base, "directory.yaml")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	oversizedPath := filepath.Join(base, "oversized.yaml")
	writeLocalFile(t, oversizedPath, openAPI3InventoryFixture())

	docs := []InventoryDocument{
		{Name: "symlink", Path: symlinkPath},
		{Name: "directory", Path: dirPath},
		{Name: "oversized", Path: oversizedPath},
	}
	expectedMessages := []string{"symlink", "directory", "larger than 8 bytes"}
	if runtime.GOOS != "windows" {
		if info, err := os.Lstat(os.DevNull); err == nil && !info.Mode().IsRegular() {
			docs = append(docs, InventoryDocument{Name: "special", Path: os.DevNull})
			expectedMessages = append(expectedMessages, "not a regular file")
		}
	}

	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: docs,
		MaxBytes:  8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Diagnostics) != len(docs) {
		t.Fatalf("diagnostics = %#v, want %d", inventory.Diagnostics, len(docs))
	}
	for _, diagnostic := range inventory.Diagnostics {
		if diagnostic.Code != "document.read" || diagnostic.Severity != "error" {
			t.Fatalf("diagnostic = %#v", diagnostic)
		}
	}
	joined := diagnosticsText(inventory.Diagnostics)
	for _, expected := range expectedMessages {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in diagnostics %#v", expected, inventory.Diagnostics)
		}
	}
}

func TestBuildOperationInventoryRejectsSymlinkedParentComponents(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLocalFile(t, filepath.Join(realDir, "openapi.yaml"), openAPI3InventoryFixture())
	linkDir := filepath.Join(base, "linked")
	symlinkOrSkip(t, realDir, linkDir)

	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{Path: filepath.Join(linkDir, "openapi.yaml")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Diagnostics) != 1 || inventory.Diagnostics[0].Code != "document.read" || !strings.Contains(inventory.Diagnostics[0].Message, "symlink") {
		t.Fatalf("diagnostics = %#v", inventory.Diagnostics)
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

func diagnosticsText(diagnostics []Diagnostic) string {
	var b strings.Builder
	for _, diagnostic := range diagnostics {
		b.WriteString(diagnostic.Message)
		b.WriteByte('\n')
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
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
                  status:
                    type: string
`
}

func responseFieldNames(fields []RequestFieldSummary) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		out = append(out, field.Path)
	}
	return out
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
          schema:
            type: object
            properties:
              id:
                type: string
              name:
                type: string
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
