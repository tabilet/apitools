package openrpc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/internal/sourceguard"
)

func TestParsePreservesOpenRPCMetadataAndSelectors(t *testing.T) {
	model, err := Parse([]byte(`{
  "openrpc": "1.3.2",
  "info": {"title": "Pet RPC", "version": "1.0.0", "description": "Pet methods"},
  "servers": [{"name": "prod", "url": "https://rpc.example.com", "summary": "Production"}],
  "methods": [{
    "name": "pet.get",
    "summary": "Get pet",
    "description": "Returns one pet.",
    "paramStructure": "by-name",
    "params": [{"$ref": "#/components/contentDescriptors/PetID"}],
    "result": {"$ref": "#/components/contentDescriptors/Pet"},
    "errors": [{"$ref": "#/components/errors/NotFound"}],
    "tags": [{"name": "pets"}]
  }],
  "components": {
    "schemas": {
      "Pet": {"type": "object", "properties": {"id": {"type": "string"}}}
    },
    "contentDescriptors": {
      "PetID": {"name": "petId", "required": true, "schema": {"type": "string"}},
      "Pet": {"name": "pet", "schema": {"$ref": "#/components/schemas/Pet"}}
    },
    "errors": {
      "NotFound": {"code": -32004, "message": "Pet not found"}
    }
  }
}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.OpenRPC != "1.3.2" || model.Info.Title != "Pet RPC" || len(model.Servers) != 1 {
		t.Fatalf("document metadata = %#v", model)
	}
	method, ok := model.MethodByName("pet.get")
	if !ok {
		t.Fatalf("missing pet.get method")
	}
	if method.ParamStructure != "by-name" || len(method.Params) != 1 || method.Params[0].Ref != "#/components/contentDescriptors/PetID" {
		t.Fatalf("method metadata = %#v", method)
	}

	summaries := model.MethodSummaries()
	if len(summaries) != 1 {
		t.Fatalf("summary count = %d, want 1", len(summaries))
	}
	summary := summaries[0]
	if summary.ID != "pet.get" || summary.Selector != "#/methods/pet.get" || summary.ResultName != "pet" {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.ParamNames) != 1 || summary.ParamNames[0] != "petId" {
		t.Fatalf("params = %#v, want petId", summary.ParamNames)
	}
	if len(summary.Params) != 1 || summary.Params[0].Name != "petId" || summary.Params[0].Ref != "#/components/contentDescriptors/PetID" || !summary.Params[0].Required {
		t.Fatalf("param summaries = %#v, want resolved petId ref", summary.Params)
	}
	if len(summary.ErrorCodes) != 1 || summary.ErrorCodes[0] != "-32004" {
		t.Fatalf("error codes = %#v, want -32004", summary.ErrorCodes)
	}

	for _, selector := range []string{"pet.get", "method:pet.get", "#/methods/pet.get"} {
		target, err := model.ResolveSelector(selector)
		if err != nil {
			t.Fatalf("ResolveSelector(%q) error = %v", selector, err)
		}
		if target.Kind != SelectorKindMethod || target.Method.Name != "pet.get" {
			t.Fatalf("selector target = %#v", target)
		}
	}
}

func TestParseRejectsMalformedOpenRPCDocuments(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "empty", content: `   `, wantErr: "empty"},
		{name: "invalid json", content: `{`, wantErr: "decode"},
		{name: "trailing", content: `{"openrpc":"1.3.2","info":{},"methods":[]} {}`, wantErr: "trailing"},
		{name: "missing openrpc", content: `{"info":{},"methods":[]}`, wantErr: "missing openrpc"},
		{name: "missing info", content: `{"openrpc":"1.3.2","methods":[]}`, wantErr: "missing info"},
		{name: "missing methods", content: `{"openrpc":"1.3.2","info":{}}`, wantErr: "missing methods"},
		{name: "method missing name", content: `{"openrpc":"1.3.2","info":{},"methods":[{}]}`, wantErr: "missing name"},
		{name: "duplicate method", content: `{"openrpc":"1.3.2","info":{},"methods":[{"name":"pet.get"},{"name":"pet.get"}]}`, wantErr: "duplicate"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Parse() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestMethodSelectorEscapesJSONPointerSegments(t *testing.T) {
	model, err := Parse([]byte(`{
  "openrpc": "1.3.2",
  "info": {"title": "Escaped"},
  "methods": [{"name": "system/foo~bar"}]
}`))
	if err != nil {
		t.Fatal(err)
	}
	summary := model.MethodSummaries()[0]
	if summary.Selector != "#/methods/system~1foo~0bar" {
		t.Fatalf("selector = %q", summary.Selector)
	}
	target, err := model.ResolveSelector("#/methods/system~1foo~0bar")
	if err != nil {
		t.Fatal(err)
	}
	if target.Method.Name != "system/foo~bar" {
		t.Fatalf("target = %#v", target)
	}
}

func TestParseRejectsResourceLimitViolations(t *testing.T) {
	_, err := Parse(bytes.Repeat([]byte{'x'}, sourceguard.MaxDocumentBytes+1))
	if err == nil || !strings.Contains(err.Error(), "maximum size") {
		t.Fatalf("Parse() error = %v, want maximum size", err)
	}
	nested := strings.Repeat(`{"x":`, sourceguard.MaxNestingDepth+1) + `0` + strings.Repeat(`}`, sourceguard.MaxNestingDepth+1)
	nested = strings.ReplaceAll(nested, `\"`, `"`)
	_, err = Parse([]byte(nested))
	if err == nil || !strings.Contains(err.Error(), "JSON nesting") {
		t.Fatalf("Parse() error = %v, want JSON nesting limit", err)
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"openrpc":"1.3.2","info":{"title":"seed"},"methods":[]}`),
		[]byte(`{}`),
		[]byte(`{"openrpc":`),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
