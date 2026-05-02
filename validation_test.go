package apitools

import (
	"context"
	"strings"
	"testing"
)

func TestSpecMetadataAcceptsOpenAPI30(t *testing.T) {
	metadata, ok := specMetadata(context.Background(), []byte(`openapi: 3.0.0
info:
  title: Example v3
  description: A v3 spec.
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        "200":
          description: ok
`))
	if !ok {
		t.Fatal("expected valid v3.0 spec")
	}
	if metadata.OpenAPI != "3.0.0" || metadata.Title != "Example v3" || metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestSpecMetadataAcceptsOpenAPI31(t *testing.T) {
	metadata, ok := specMetadata(context.Background(), []byte(`openapi: 3.1.0
info:
  title: Example v3.1
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
    post:
      responses:
        "201":
          description: ok
`))
	if !ok {
		t.Fatal("expected valid v3.1 spec")
	}
	if metadata.OpenAPI != "3.1.0" || metadata.OperationCount != 2 {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestSpecMetadataAcceptsSwagger2(t *testing.T) {
	metadata, ok := specMetadata(context.Background(), []byte(`swagger: "2.0"
info:
  title: Legacy
  version: 1.0.0
paths:
  /thing:
    get:
      responses:
        "200":
          description: ok
`))
	if !ok {
		t.Fatal("expected valid v2.0 spec")
	}
	if metadata.Swagger != "2.0" || metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestSpecMetadataAcceptsJSONInput(t *testing.T) {
	metadata, ok := specMetadata(context.Background(), []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "JSON Example", "version": "1.0.0"},
  "paths": {"/x": {"get": {"responses": {"200": {"description": "ok"}}}}}
}`))
	if !ok || metadata.Title != "JSON Example" {
		t.Fatalf("metadata = %#v ok = %v", metadata, ok)
	}
}

func TestSpecMetadataRejectsExternalRef(t *testing.T) {
	for _, body := range []string{
		`{"openapi":"3.0.0","info":{"title":"E","version":"1"},"paths":{"/x":{"$ref":"https://evil.example/path.yaml"}}}`,
		`{"openapi":"3.0.0","info":{"title":"E","version":"1"},"paths":{"/x":{"$ref":"./local.yaml"}}}`,
	} {
		if _, ok := specMetadata(context.Background(), []byte(body)); ok {
			t.Fatalf("expected rejection for body %s", body)
		}
	}
}

func TestSpecMetadataAllowsInternalRef(t *testing.T) {
	body := `{
  "openapi": "3.0.0",
  "info": {"title": "Refs", "version": "1.0.0"},
  "paths": {"/x": {"get": {"responses": {"200": {"$ref": "#/components/responses/OK"}}}}},
  "components": {"responses": {"OK": {"description": "ok"}}}
}`
	if _, ok := specMetadata(context.Background(), []byte(body)); !ok {
		t.Fatalf("internal $ref should be allowed")
	}
}

func TestSpecMetadataRejectsUnknownVersion(t *testing.T) {
	body := `{"openapi":"4.0.0","info":{"title":"x","version":"1"},"paths":{}}`
	if _, ok := specMetadata(context.Background(), []byte(body)); ok {
		t.Fatal("expected rejection for unknown openapi version")
	}
}

func TestSpecMetadataRejectsEmptyAndUnshaped(t *testing.T) {
	for _, raw := range []string{"", "   ", "not-yaml-or-json: [", `{"hello":"world"}`} {
		if _, ok := specMetadata(context.Background(), []byte(raw)); ok {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

func TestSpecMetadataHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := specMetadata(ctx, []byte(validOpenAPISpec)); ok {
		t.Fatal("cancelled context should short-circuit")
	}
}

func TestSpecMetadataRejectsDeeplyNestedDocument(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"openapi":"3.0.0","info":{"title":"D","version":"1"},"paths":{},"deep":`)
	closer := ""
	for i := 0; i <= maxRefScanDepth+5; i++ {
		b.WriteString(`{"x":`)
		closer = "}" + closer
	}
	b.WriteString(`null`)
	b.WriteString(closer)
	b.WriteString(`}`)
	if _, ok := specMetadata(context.Background(), []byte(b.String())); ok {
		t.Fatal("documents past maxRefScanDepth should be rejected")
	}
}

func TestContainsExternalRefRecognizesShallow(t *testing.T) {
	value := map[string]any{
		"a": map[string]any{"$ref": "https://example.com/x.yaml"},
	}
	if !containsExternalRef(value, 0) {
		t.Fatal("expected external $ref to be flagged")
	}
}

func TestContainsExternalRefIgnoresFragmentRef(t *testing.T) {
	value := map[string]any{
		"a": map[string]any{"$ref": "#/components/schemas/A"},
	}
	if containsExternalRef(value, 0) {
		t.Fatal("fragment $ref should not be flagged")
	}
}
