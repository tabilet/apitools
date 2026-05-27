package apitools

import "testing"

func TestParseGRPCProtobufOperationSummaries(t *testing.T) {
	ops, err := ParseGRPCProtobufOperationSummaries([]byte(`
syntax = "proto3";
package issues.v1;

message GetIssueRequest {
  string id = 1;
}

message Issue {
  string id = 1;
}

service IssueService {
  rpc GetIssue(GetIssueRequest) returns (Issue);
}
`), "grpc-protobuf/issues.proto")
	if err != nil {
		t.Fatalf("ParseGRPCProtobufOperationSummaries() error = %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("operation count = %d, want 1", len(ops))
	}
	op := ops[0]
	if op.ID != "issues.v1.IssueService/GetIssue" || op.Method != "grpc" {
		t.Fatalf("operation metadata = %#v", op)
	}
	if op.Path != "#/services/issues.v1.IssueService/methods/GetIssue" || op.Provenance != "grpc-protobuf" {
		t.Fatalf("path/provenance = %q/%q", op.Path, op.Provenance)
	}
	if op.Extensions["grpc_full_method"] != "/issues.v1.IssueService/GetIssue" || op.Extensions["source_operation_ref"] != op.Path {
		t.Fatalf("extensions = %#v", op.Extensions)
	}
	if op.RequestBody == nil || len(op.RequestBody.Fields) != 1 || op.RequestBody.Fields[0].Path != "id" {
		t.Fatalf("request body = %#v", op.RequestBody)
	}
}
