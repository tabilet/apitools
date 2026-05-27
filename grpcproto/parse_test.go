package grpcproto

import "testing"

func TestParseProtoPreservesServicesMessagesAndSelectors(t *testing.T) {
	model, err := Parse([]byte(`
syntax = "proto3";
package issues.v1;

message GetIssueRequest {
  string id = 1;
  repeated string labels = 2;
}

message Issue {
  string id = 1;
}

service IssueService {
  rpc GetIssue(GetIssueRequest) returns (Issue);
  rpc WatchIssues(stream GetIssueRequest) returns (stream Issue);
}
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindProtoFile {
		t.Fatalf("source kind = %q, want proto file", model.SourceKind)
	}
	summaries := model.MethodSummaries()
	if len(summaries) != 2 {
		t.Fatalf("method summaries = %d, want 2: %#v", len(summaries), summaries)
	}
	get := summaries[0]
	if get.ID != "issues.v1.IssueService/GetIssue" || get.FullMethod != "/issues.v1.IssueService/GetIssue" {
		t.Fatalf("get ids = %#v", get)
	}
	if get.Selector != "#/services/issues.v1.IssueService/methods/GetIssue" || get.SourceOperationRef != get.Selector {
		t.Fatalf("selector = %q ref = %q", get.Selector, get.SourceOperationRef)
	}
	if len(get.RequestFields) != 2 || get.RequestFields[0].Name != "id" || get.RequestFields[1].Name != "labels" || !get.RequestFields[1].Repeated {
		t.Fatalf("request fields = %#v", get.RequestFields)
	}
	target, err := model.ResolveSelector("/issues.v1.IssueService/GetIssue")
	if err != nil {
		t.Fatalf("ResolveSelector() error = %v", err)
	}
	if target.Kind != SelectorKindMethod || target.Summary.ID != get.ID {
		t.Fatalf("target = %#v", target)
	}
	watch := summaries[1]
	if !watch.ClientStreaming || !watch.ServerStreaming {
		t.Fatalf("streaming flags = %#v", watch)
	}
}

func TestParseDescriptorJSON(t *testing.T) {
	model, err := Parse([]byte(`{
  "file": [{
    "name": "issues.proto",
    "package": "issues.v1",
    "syntax": "proto3",
    "messageType": [{
      "name": "GetIssueRequest",
      "field": [{"name": "id", "number": 1, "label": "LABEL_OPTIONAL", "type": "TYPE_STRING", "jsonName": "id"}]
    }],
    "service": [{
      "name": "IssueService",
      "method": [{"name": "GetIssue", "inputType": ".issues.v1.GetIssueRequest", "outputType": ".issues.v1.Issue"}]
    }]
  }]
}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindDescriptorJSON {
		t.Fatalf("source kind = %q, want descriptor JSON", model.SourceKind)
	}
	summaries := model.MethodSummaries()
	if len(summaries) != 1 || summaries[0].ID != "issues.v1.IssueService/GetIssue" {
		t.Fatalf("summaries = %#v", summaries)
	}
}

func TestParseBinaryDescriptorSet(t *testing.T) {
	data := message(
		fieldBytes(1, message(
			fieldString(1, "issues.proto"),
			fieldString(2, "issues.v1"),
			fieldBytes(4, message(
				fieldString(1, "GetIssueRequest"),
				fieldBytes(2, message(
					fieldString(1, "id"),
					fieldVarint(3, 1),
					fieldVarint(4, 1),
					fieldVarint(5, 9),
					fieldString(10, "id"),
				)),
			)),
			fieldBytes(6, message(
				fieldString(1, "IssueService"),
				fieldBytes(2, message(
					fieldString(1, "GetIssue"),
					fieldString(2, ".issues.v1.GetIssueRequest"),
					fieldString(3, ".issues.v1.Issue"),
				)),
			)),
			fieldString(12, "proto3"),
		)),
	)
	model, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindDescriptorSet {
		t.Fatalf("source kind = %q, want descriptor set", model.SourceKind)
	}
	summaries := model.MethodSummaries()
	if len(summaries) != 1 || summaries[0].RequestFields[0].Type != "string" {
		t.Fatalf("summaries = %#v", summaries)
	}
}

func TestDetectLocalArtifact(t *testing.T) {
	detection, ok := DetectLocalArtifact([]byte(`syntax = "proto3"; service S { rpc M (Req) returns (Resp); } message Req { string id = 1; }`), "api.proto")
	if !ok {
		t.Fatalf("DetectLocalArtifact() did not detect proto")
	}
	if detection.Kind != ArtifactKindProtoFile || detection.Format != "proto" {
		t.Fatalf("detection = %#v", detection)
	}
	if _, ok := DetectLocalArtifact([]byte(`{"title":"not protobuf"}`), "spec.json"); ok {
		t.Fatalf("non-protobuf JSON detected")
	}
}

func TestParseRejectsMalformedSource(t *testing.T) {
	if _, err := Parse([]byte(`not protobuf`)); err == nil {
		t.Fatalf("Parse() error = nil, want malformed source error")
	}
}

func message(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func fieldString(number int, value string) []byte {
	return fieldBytes(number, []byte(value))
}

func fieldBytes(number int, value []byte) []byte {
	out := encodeVarint(uint64(number<<3 | 2))
	out = append(out, encodeVarint(uint64(len(value)))...)
	out = append(out, value...)
	return out
}

func fieldVarint(number int, value uint64) []byte {
	out := encodeVarint(uint64(number << 3))
	out = append(out, encodeVarint(value)...)
	return out
}

func encodeVarint(value uint64) []byte {
	var out []byte
	for value >= 0x80 {
		out = append(out, byte(value)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}
