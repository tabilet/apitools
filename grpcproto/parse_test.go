package grpcproto

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/internal/sourceguard"
)

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

func TestParseBinaryDescriptorSetFallsBackAfterProtoTextHeuristic(t *testing.T) {
	data := message(
		fieldBytes(1, message(
			fieldString(1, "issues.proto"),
			fieldString(2, "issues.v1"),
			fieldBytes(4, message(fieldString(1, "Request"))),
			fieldBytes(6, message(
				fieldString(1, "IssueService"),
				fieldBytes(2, message(
					fieldString(1, "GetIssue"),
					fieldString(2, ".issues.v1.Request"),
					fieldString(3, ".issues.v1.Response"),
				)),
			)),
			fieldBytes(9, message(fieldBytes(1, message(fieldString(3, "\nservice documented in source info"))))),
			fieldString(12, "proto3"),
		)),
	)
	if !looksLikeProtoText(data) {
		t.Fatal("fixture did not exercise the proto-text heuristic")
	}
	model, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindDescriptorSet {
		t.Fatalf("source kind = %q, want descriptor set", model.SourceKind)
	}
}

func TestParseBinaryDescriptorSetRejectsDeepNestedMessages(t *testing.T) {
	data := message(
		fieldBytes(1, message(
			fieldString(1, "deep.proto"),
			fieldString(2, "deep.v1"),
			fieldBytes(4, deepNestedDescriptorMessage(maxDescriptorProtoDepth+2)),
			fieldString(12, "proto3"),
		)),
	)
	_, err := parseBinaryDescriptorSet(data)
	if err == nil {
		t.Fatal("Parse() error = nil, want descriptor nesting depth error")
	}
	if !strings.Contains(err.Error(), "maximum message depth") {
		t.Fatalf("Parse() error = %v, want maximum message depth", err)
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

func TestParseRejectsStrayClosingBrace(t *testing.T) {
	_, err := Parse([]byte(`syntax }`))
	if err == nil || !strings.Contains(err.Error(), "closing brace") {
		t.Fatalf("Parse() error = %v, want closing brace error", err)
	}
}

func TestParseRejectsResourceLimitViolations(t *testing.T) {
	t.Run("document bytes", func(t *testing.T) {
		_, err := Parse(bytes.Repeat([]byte{'x'}, sourceguard.MaxDocumentBytes+1))
		if err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("Parse() error = %v, want maximum size", err)
		}
	})
	t.Run("text nesting", func(t *testing.T) {
		input := `message Root { ` + strings.Repeat(`message Nested { `, sourceguard.MaxNestingDepth+1) + strings.Repeat(`}`, sourceguard.MaxNestingDepth+2)
		_, err := Parse([]byte(input))
		if err == nil || !strings.Contains(err.Error(), "nesting") {
			t.Fatalf("Parse() error = %v, want nesting limit", err)
		}
	})
	t.Run("descriptor JSON nesting", func(t *testing.T) {
		input := `{"file":[{"messageType":[` + strings.Repeat(`{"name":"N","nestedType":[`, sourceguard.MaxNestingDepth) + `{}` + strings.Repeat(`]}`, sourceguard.MaxNestingDepth) + `]}]}`
		input = strings.ReplaceAll(input, `\"`, `"`)
		_, err := Parse([]byte(input))
		if err == nil || !strings.Contains(err.Error(), "JSON nesting") {
			t.Fatalf("Parse() error = %v, want JSON nesting limit", err)
		}
	})
	t.Run("binary work", func(t *testing.T) {
		input := bytes.Repeat([]byte{0x10, 0x00}, sourceguard.MaxWorkItems+1)
		_, err := parseBinaryDescriptorSet(input)
		if err == nil || !strings.Contains(err.Error(), "parser work") {
			t.Fatalf("parseBinaryDescriptorSet() error = %v, want work limit", err)
		}
	})
}

func FuzzParse(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`syntax = "proto3"; message M { string id = 1; }`),
		[]byte(`{"file":[{"name":"seed.proto","messageType":[{"name":"M"}]}]}`),
		{0x0a, 0x00},
		[]byte(`syntax }`),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
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

func deepNestedDescriptorMessage(depth int) []byte {
	parts := [][]byte{fieldString(1, fmt.Sprintf("M%d", depth))}
	if depth > 0 {
		parts = append(parts, fieldBytes(3, deepNestedDescriptorMessage(depth-1)))
	}
	return message(parts...)
}

func encodeVarint(value uint64) []byte {
	var out []byte
	for value >= 0x80 {
		out = append(out, byte(value)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}
