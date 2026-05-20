package googlediscovery

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	apitools "github.com/OpenUdon/apitools"
)

func TestConvertDriveDiscovery(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "drive.discovery.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	out, err := Convert(data)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	doc := decodeConvertedDocument(t, out)
	info := doc["info"].(map[string]any)
	if got, want := info["title"], "Google Drive API"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	if got, want := serverURL(doc), "https://www.googleapis.com"; got != want {
		t.Fatalf("server url = %q, want %q", got, want)
	}

	for _, path := range []string{"/drive/v3/files", "/drive/v3/files/{fileId}", "/upload/drive/v3/files"} {
		if _, ok := doc["paths"].(map[string]any)[path]; !ok {
			t.Fatalf("missing path %q", path)
		}
	}

	inventory := mustInventory(t, out)
	if op := mustOperation(t, inventory, "drive_files_list"); op.Method != "GET" || op.Path != "/drive/v3/files" {
		t.Fatalf("drive_files_list = %#v", op)
	}
	create := mustOperation(t, inventory, "drive_files_create")
	if create.RequestBody == nil || !containsString(create.RequestBody.ContentTypes, "multipart/related") {
		t.Fatalf("drive_files_create request body = %#v", create.RequestBody)
	}

	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	for _, want := range []string{"File", "FileList", "User"} {
		if _, ok := schemas[want]; !ok {
			t.Fatalf("missing schema %q", want)
		}
	}
}

func TestConvertPreservesDiscoveryOAuthScopes(t *testing.T) {
	raw := map[string]any{
		"auth": map[string]any{
			"oauth2": map[string]any{
				"scopes": map[string]any{
					"https://www.googleapis.com/auth/example.read": map[string]any{
						"description": "Read example resources",
					},
				},
			},
		},
		"methods": map[string]any{
			"list": map[string]any{
				"id":         "examples.list",
				"httpMethod": "GET",
				"path":       "examples",
				"scopes": []any{
					"https://www.googleapis.com/auth/example.read",
				},
			},
		},
	}

	doc, err := ConvertMap(raw)
	if err != nil {
		t.Fatalf("ConvertMap failed: %v", err)
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal converted document: %v", err)
	}
	inventory := mustInventory(t, out)
	op := mustOperation(t, inventory, "examples_list")
	if len(op.Security) != 1 {
		t.Fatalf("security requirement count = %d, want 1", len(op.Security))
	}
	security := op.Security[0]
	if got, want := security.Name, "google_oauth2"; got != want {
		t.Fatalf("security scheme = %q, want %q", got, want)
	}
	if got, want := security.Flows, []string{"authorizationCode"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("oauth flows = %#v, want %#v", got, want)
	}
	if got, want := security.AuthorizationURL, "https://accounts.google.com/o/oauth2/v2/auth"; got != want {
		t.Fatalf("authorization URL = %q, want %q", got, want)
	}
	if got, want := security.TokenURL, "https://oauth2.googleapis.com/token"; got != want {
		t.Fatalf("token URL = %q, want %q", got, want)
	}
	if got, want := security.Scopes, []string{"https://www.googleapis.com/auth/example.read"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("operation scopes = %#v, want %#v", got, want)
	}
}

func TestConvertOmitsMethodSecurityWhenOAuthSchemeUnavailable(t *testing.T) {
	doc, err := ConvertMap(map[string]any{
		"methods": map[string]any{
			"list": map[string]any{
				"id":         "examples.list",
				"httpMethod": "GET",
				"path":       "examples",
				"scopes": []any{
					"https://www.googleapis.com/auth/example.read",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ConvertMap failed: %v", err)
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal converted document: %v", err)
	}
	op := mustOperation(t, mustInventory(t, out), "examples_list")
	if got := op.Security; got != nil {
		t.Fatalf("operation security = %#v, want nil", got)
	}
}

func TestConvertRejectsMalformedOptionalMaps(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "schemas",
			raw:  map[string]any{"schemas": []any{}},
			want: `field "schemas" must be an object`,
		},
		{
			name: "schema properties",
			raw:  map[string]any{"schemas": map[string]any{"Thing": map[string]any{"properties": "bad"}}},
			want: "schemas.Thing.properties must be an object",
		},
		{
			name: "resources",
			raw:  map[string]any{"resources": []any{}},
			want: `field "resources" must be an object`,
		},
		{
			name: "methods",
			raw:  map[string]any{"methods": []any{}},
			want: `field "methods" must be an object`,
		},
		{
			name: "parameters",
			raw: map[string]any{"methods": map[string]any{
				"list": map[string]any{
					"id":         "items.list",
					"httpMethod": "GET",
					"path":       "/items",
					"parameters": "bad",
				},
			}},
			want: `field "parameters" must be an object`,
		},
		{
			name: "request",
			raw: map[string]any{"methods": map[string]any{
				"create": map[string]any{
					"id":         "items.create",
					"httpMethod": "POST",
					"path":       "/items",
					"request":    "Thing",
				},
			}},
			want: `field "request" must be an object`,
		},
		{
			name: "response",
			raw: map[string]any{"methods": map[string]any{
				"get": map[string]any{
					"id":         "items.get",
					"httpMethod": "GET",
					"path":       "/items/{id}",
					"response":   "Thing",
				},
			}},
			want: `field "response" must be an object`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ConvertMap(tc.raw)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want containing %q", err, tc.want)
			}
		})
	}
}

func TestConvertUsesDiscoveryResponseRefObject(t *testing.T) {
	out, err := ConvertMap(map[string]any{
		"schemas": map[string]any{
			"Thing": map[string]any{"type": "object"},
		},
		"methods": map[string]any{
			"get": map[string]any{
				"id":         "things.get",
				"httpMethod": "GET",
				"path":       "/things/{id}",
				"response":   map[string]any{"$ref": "Thing"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	paths := out["paths"].(map[string]any)
	op := paths["/things/{id}"].(map[string]any)["get"].(map[string]any)
	responses := op["responses"].(map[string]any)
	ok := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if got, want := ok["$ref"], "#/components/schemas/Thing"; got != want {
		t.Fatalf("response ref = %v, want %v", got, want)
	}
}

func TestConvertGmailDiscovery(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "gmail.discovery.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	out, err := Convert(data)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	doc := decodeConvertedDocument(t, out)
	info := doc["info"].(map[string]any)
	if got, want := info["title"], "Gmail API"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	if got, want := serverURL(doc), "https://gmail.googleapis.com"; got != want {
		t.Fatalf("server url = %q, want %q", got, want)
	}
	for _, path := range []string{
		"/gmail/v1/users/{userId}/labels",
		"/gmail/v1/users/{userId}/messages",
		"/gmail/v1/users/{userId}/threads",
	} {
		if _, ok := doc["paths"].(map[string]any)[path]; !ok {
			t.Fatalf("missing path %q", path)
		}
	}

	inventory := mustInventory(t, out)
	for _, tc := range []struct {
		method string
		path   string
		id     string
	}{
		{method: "GET", path: "/gmail/v1/users/{userId}/labels", id: "gmail_users_labels_list"},
		{method: "POST", path: "/gmail/v1/users/{userId}/labels", id: "gmail_users_labels_create"},
		{method: "GET", path: "/gmail/v1/users/{userId}/messages", id: "gmail_users_messages_list"},
		{method: "GET", path: "/gmail/v1/users/{userId}/threads", id: "gmail_users_threads_list"},
	} {
		op := mustOperation(t, inventory, tc.id)
		if op.Method != tc.method || op.Path != tc.path {
			t.Fatalf("%s = %s %s, want %s %s", tc.id, op.Method, op.Path, tc.method, tc.path)
		}
	}
}

func TestConvertDriveDiscoveryIsValidJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "drive.discovery.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	out, err := Convert(data)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	var v any
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&v); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestConvertDriveDiscoveryFullFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "drive.full.discovery.json"))
	if err != nil {
		t.Fatalf("read full fixture: %v", err)
	}

	out, err := Convert(data)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	inventory := mustInventory(t, out)
	for _, tc := range []struct {
		path   string
		method string
		id     string
	}{
		{path: "/drive/v3/drives", method: "GET", id: "drive_drives_list"},
		{path: "/drive/v3/drives", method: "POST", id: "drive_drives_create"},
		{path: "/drive/v3/files/{fileId}/copy", method: "POST", id: "drive_files_copy"},
		{path: "/drive/v3/files/{fileId}/download", method: "POST", id: "drive_files_download"},
	} {
		op := mustOperation(t, inventory, tc.id)
		if op.Method != tc.method || op.Path != tc.path {
			t.Fatalf("%s = %s %s, want %s %s", tc.id, op.Method, op.Path, tc.method, tc.path)
		}
	}
}

func TestSanitizeParameterName(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "", want: ""},
		{name: "fileId", want: "fileId"},
		{name: "$.xgafv", want: "x_.xgafv"},
		{name: "@foo", want: "x_foo"},
	} {
		if got := sanitizeParameterName(tc.name); got != tc.want {
			t.Fatalf("sanitizeParameterName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func mustInventory(t *testing.T, content []byte) apitools.OperationInventory {
	t.Helper()
	inventory, err := apitools.BuildOperationInventory(context.Background(), apitools.InventoryOptions{
		Documents: []apitools.InventoryDocument{{
			Name:    "converted discovery",
			Content: content,
		}},
	})
	if err != nil {
		t.Fatalf("BuildOperationInventory failed: %v", err)
	}
	for _, diagnostic := range inventory.Diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("inventory has error diagnostics: %#v", inventory.Diagnostics)
		}
	}
	return inventory
}

func mustOperation(t *testing.T, inventory apitools.OperationInventory, id string) apitools.OperationSummary {
	t.Helper()
	for _, op := range inventory.Operations {
		if op.OperationID == id {
			return op
		}
	}
	t.Fatalf("missing operationId %q in %#v", id, inventory.Operations)
	return apitools.OperationSummary{}
}

func decodeConvertedDocument(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(content, &out); err != nil {
		t.Fatalf("unmarshal converted document: %v", err)
	}
	return out
}

func serverURL(doc map[string]any) string {
	servers, _ := doc["servers"].([]any)
	if len(servers) == 0 {
		return ""
	}
	first, _ := servers[0].(map[string]any)
	url, _ := first["url"].(string)
	return url
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
