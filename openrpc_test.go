package apitools

import (
	"testing"

	"github.com/OpenUdon/apitools/openrpc"
)

func TestParseOpenRPCOperationSummaries(t *testing.T) {
	ops, err := ParseOpenRPCOperationSummaries([]byte(`{
  "openrpc": "1.3.2",
  "info": {"title": "Pet RPC", "version": "1.0.0"},
  "methods": [{
    "name": "pet.get",
    "summary": "Get pet",
    "description": "Returns a pet.",
    "params": [{"name": "petId", "schema": {"type": "string"}}],
    "result": {"name": "pet"}
  }]
}`), "openrpc/pets.json")
	if err != nil {
		t.Fatalf("ParseOpenRPCOperationSummaries() error = %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("operation count = %d, want 1", len(ops))
	}
	op := ops[0]
	if op.ID != "pet.get" || op.OperationID != "pet.get" {
		t.Fatalf("operation ids = %q %q, want pet.get", op.ID, op.OperationID)
	}
	if op.DocumentName != "Pet RPC" || op.DocumentPath != "openrpc/pets.json" || op.DocumentRelativePath != "openrpc/pets.json" {
		t.Fatalf("document metadata = %#v", op)
	}
	if op.Method != "json-rpc" || op.Path != "#/methods/pet.get" {
		t.Fatalf("operation method/path = %q %q, want json-rpc #/methods/pet.get", op.Method, op.Path)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "petId" || op.Parameters[0].In != "json-rpc" {
		t.Fatalf("parameters = %#v, want petId json-rpc", op.Parameters)
	}
	if op.Extensions["openrpc_selector"] != "#/methods/pet.get" || op.Extensions["openrpc_result"] != "pet" {
		t.Fatalf("extensions = %#v", op.Extensions)
	}
	if op.Summary != "Get pet" || op.Description != "Returns a pet." {
		t.Fatalf("operation text = %q %q", op.Summary, op.Description)
	}
	if op.Provenance != "openrpc" {
		t.Fatalf("provenance = %q, want openrpc", op.Provenance)
	}
}

func TestOpenRPCOperationSummariesSkipsNilAndBlankMethods(t *testing.T) {
	got := OpenRPCOperationSummaries("openrpc/pets.json", &openrpc.Model{
		Info: openrpc.Info{Title: "Pet RPC"},
		Methods: []*openrpc.Method{
			nil,
			{Name: "   "},
			{Name: "pet.get", Summary: "Get pet"},
		},
	})
	if len(got) != 1 || got[0].OperationID != "pet.get" {
		t.Fatalf("summaries = %#v, want pet.get only", got)
	}
}
