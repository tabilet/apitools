package awssmithy_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apitools "github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
)

func TestConvertRestJSONOperation(t *testing.T) {
	doc := mustConvertMap(t, restJSONFixture())

	if got, want := nestedString(doc, "info", "title"), "AWS Lambda"; got != want {
		t.Fatalf("title = %q, want %q", got, want)
	}
	if got, want := doc["x-smithy-protocol"], "restJson1"; got != want {
		t.Fatalf("protocol = %v, want %v", got, want)
	}
	if got, want := doc["x-aws-signing-name"], "lambda"; got != want {
		t.Fatalf("signing name = %v, want %v", got, want)
	}

	op := operation(t, doc, "/2015-03-31/functions/{FunctionName}", "get")
	if got, want := op["operationId"], "GetFunction"; got != want {
		t.Fatalf("operationId = %v, want %v", got, want)
	}
	if got, want := op["x-smithy-id"], "com.amazonaws.lambda#GetFunction"; got != want {
		t.Fatalf("x-smithy-id = %v, want %v", got, want)
	}
	params := op["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("parameters = %#v, want 2 entries", params)
	}
	assertParameter(t, params, "path", "FunctionName", true)
	assertParameter(t, params, "query", "Qualifier", false)

	responses := op["responses"].(map[string]any)
	schema := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if got, want := schema["$ref"], "#/components/schemas/GetFunctionResponse"; got != want {
		t.Fatalf("response schema = %v, want %v", got, want)
	}

	out, err := awssmithy.Convert(mustJSON(t, restJSONFixture()))
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	inventory, err := apitools.BuildOperationInventory(context.Background(), apitools.InventoryOptions{
		Documents: []apitools.InventoryDocument{{Name: "lambda", Content: out}},
	})
	if err != nil {
		t.Fatalf("BuildOperationInventory failed: %v", err)
	}
	if len(inventory.Operations) != 1 || inventory.Operations[0].OperationID != "GetFunction" {
		t.Fatalf("inventory operations = %#v", inventory.Operations)
	}
}

func TestConvertRestXMLOperation(t *testing.T) {
	doc := mustConvertMap(t, restXMLFixture())
	op := operation(t, doc, "/{Bucket}/{Key}", "put")
	if got, want := op["operationId"], "PutObject"; got != want {
		t.Fatalf("operationId = %v, want %v", got, want)
	}
	if got := stringSlice(op["x-smithy-greedy-labels"]); len(got) != 1 || got[0] != "Key" {
		t.Fatalf("greedy labels = %#v, want [Key]", got)
	}
	params := op["parameters"].([]any)
	assertParameter(t, params, "path", "Bucket", true)
	assertParameter(t, params, "path", "Key", true)
	assertParameter(t, params, "header", "Content-Type", false)

	body := op["requestBody"].(map[string]any)
	content := body["content"].(map[string]any)
	if _, ok := content["application/octet-stream"]; !ok {
		t.Fatalf("request content = %#v, want application/octet-stream", content)
	}
}

func TestConvertAWSQueryOperation(t *testing.T) {
	doc := mustConvertMap(t, awsQueryFixture())
	op := operation(t, doc, "/?Action=Publish", "post")
	if got, want := op["x-aws-query-action"], "Publish"; got != want {
		t.Fatalf("query action = %v, want %v", got, want)
	}
	body := op["requestBody"].(map[string]any)
	if got, want := body["x-aws-query-version"], "2010-03-31"; got != want {
		t.Fatalf("query version = %v, want %v", got, want)
	}
	schema := body["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	if got, want := props["Action"].(map[string]any)["default"], "Publish"; got != want {
		t.Fatalf("Action default = %v, want %v", got, want)
	}
	if got, want := props["Version"].(map[string]any)["default"], "2010-03-31"; got != want {
		t.Fatalf("Version default = %v, want %v", got, want)
	}
	required := stringSlice(schema["required"])
	for _, want := range []string{"Action", "Version", "Message", "TopicArn"} {
		if !contains(required, want) {
			t.Fatalf("required = %#v, missing %q", required, want)
		}
	}
}

func TestConvertPreservesCollidingSmithyOperations(t *testing.T) {
	doc := mustConvertMap(t, map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.s3#AmazonS3": serviceShape("S3", "restXml", []any{
				ref("com.amazonaws.s3#GetBucketAcl"),
				ref("com.amazonaws.s3#GetBucketTagging"),
			}),
			"com.amazonaws.s3#GetBucketAcl": map[string]any{"type": "operation", "traits": map[string]any{
				"smithy.api#http": map[string]any{"method": "GET", "uri": "/{Bucket}?acl", "code": float64(200)},
			}},
			"com.amazonaws.s3#GetBucketTagging": map[string]any{"type": "operation", "traits": map[string]any{
				"smithy.api#http": map[string]any{"method": "GET", "uri": "/{Bucket}?tagging", "code": float64(200)},
			}},
		},
	})
	if got := operation(t, doc, "/{Bucket}?acl", "get")["operationId"]; got != "GetBucketAcl" {
		t.Fatalf("acl operationId = %v, want GetBucketAcl", got)
	}
	if got := operation(t, doc, "/{Bucket}?tagging", "get")["operationId"]; got != "GetBucketTagging" {
		t.Fatalf("tagging operationId = %v, want GetBucketTagging", got)
	}

	queryDoc := mustConvertMap(t, map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.sns#SNS": serviceShape("SNS", "awsQuery", []any{
				ref("com.amazonaws.sns#Publish"),
				ref("com.amazonaws.sns#Subscribe"),
			}),
			"com.amazonaws.sns#Publish":   map[string]any{"type": "operation"},
			"com.amazonaws.sns#Subscribe": map[string]any{"type": "operation"},
		},
	})
	if got := operation(t, queryDoc, "/?Action=Publish", "post")["operationId"]; got != "Publish" {
		t.Fatalf("publish operationId = %v, want Publish", got)
	}
	if got := operation(t, queryDoc, "/?Action=Subscribe", "post")["operationId"]; got != "Subscribe" {
		t.Fatalf("subscribe operationId = %v, want Subscribe", got)
	}
}

func TestConvertModelsHTTPPrefixHeaders(t *testing.T) {
	doc := mustConvertMap(t, restXMLFixture())
	op := operation(t, doc, "/{Bucket}/{Key}", "put")
	params := op["parameters"].([]any)
	var found bool
	for _, param := range params {
		m := param.(map[string]any)
		if m["in"] == "header" && m["name"] == "x-amz-meta-" {
			found = true
			if got, want := m["x-smithy-http-prefix-headers"], "x-amz-meta-"; got != want {
				t.Fatalf("prefix extension = %v, want %v", got, want)
			}
		}
	}
	if !found {
		t.Fatalf("missing prefix header parameter in %#v", params)
	}
}

func TestConvertModelsResponseHTTPBindings(t *testing.T) {
	doc := mustConvertMap(t, map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.lambda#Lambda": serviceShape("Lambda", "restJson1", []any{ref("com.amazonaws.lambda#Invoke")}),
			"com.amazonaws.lambda#Invoke": map[string]any{
				"type":   "operation",
				"output": ref("com.amazonaws.lambda#InvocationResponse"),
				"traits": map[string]any{
					"smithy.api#http": map[string]any{"method": "POST", "uri": "/2015-03-31/functions/{FunctionName}/invocations", "code": float64(200)},
				},
			},
			"com.amazonaws.lambda#InvocationResponse": map[string]any{"type": "structure", "members": map[string]any{
				"StatusCode":      member("smithy.api#Integer", map[string]any{"smithy.api#httpResponseCode": map[string]any{}}),
				"FunctionError":   member("smithy.api#String", map[string]any{"smithy.api#httpHeader": "X-Amz-Function-Error"}),
				"ExecutedVersion": member("smithy.api#String", map[string]any{"smithy.api#httpHeader": "X-Amz-Executed-Version"}),
				"Payload":         member("com.amazonaws.lambda#Blob", map[string]any{"smithy.api#httpPayload": map[string]any{}}),
			}},
			"com.amazonaws.lambda#Blob": map[string]any{"type": "blob"},
		},
	})
	op := operation(t, doc, "/2015-03-31/functions/{FunctionName}/invocations", "post")
	response := op["responses"].(map[string]any)["200"].(map[string]any)
	if got, want := response["x-smithy-http-response-code-member"], "StatusCode"; got != want {
		t.Fatalf("response code member = %v, want %v", got, want)
	}
	headers := response["headers"].(map[string]any)
	if _, ok := headers["X-Amz-Function-Error"]; !ok {
		t.Fatalf("headers = %#v, missing function error", headers)
	}
	content := response["content"].(map[string]any)
	if _, ok := content["application/octet-stream"]; !ok {
		t.Fatalf("content = %#v, want octet-stream payload", content)
	}
}

func TestConvertRejectsMalformedSmithy(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  map[string]any
		want string
	}{
		{name: "unsupported version", raw: map[string]any{"smithy": "1.0", "shapes": map[string]any{}}, want: "unsupported smithy version"},
		{name: "missing shapes", raw: map[string]any{"smithy": "2.0"}, want: "missing shapes"},
		{name: "no service", raw: map[string]any{"smithy": "2.0", "shapes": map[string]any{"example#String": map[string]any{"type": "string"}}}, want: "exactly one service"},
		{name: "multiple services", raw: map[string]any{"smithy": "2.0", "shapes": map[string]any{
			"example#A": serviceShape("A", "restJson1", nil),
			"example#B": serviceShape("B", "restJson1", nil),
		}}, want: "multiple service"},
		{name: "unsupported protocol", raw: map[string]any{"smithy": "2.0", "shapes": map[string]any{
			"example#Svc": map[string]any{"type": "service", "version": "1", "traits": map[string]any{}},
		}}, want: "unsupported or missing"},
		{name: "invalid ref", raw: map[string]any{"smithy": "2.0", "shapes": map[string]any{
			"example#Svc": serviceShape("Svc", "restJson1", []any{ref("example#Broken")}),
			"example#Broken": map[string]any{"type": "operation", "input": ref("example#Missing"), "traits": map[string]any{
				"smithy.api#http": map[string]any{"method": "POST", "uri": "/broken", "code": float64(200)},
			}},
		}}, want: "missing input shape"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := awssmithy.ConvertMap(tc.raw)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want containing %q", err, tc.want)
			}
		})
	}
}

func TestConvertRecursiveSchemaUsesRefs(t *testing.T) {
	doc := mustConvertMap(t, map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"example#Svc": serviceShape("Svc", "restJson1", []any{ref("example#GetNode")}),
			"example#GetNode": map[string]any{"type": "operation", "output": ref("example#Node"), "traits": map[string]any{
				"smithy.api#http": map[string]any{"method": "GET", "uri": "/node", "code": float64(200)},
			}},
			"example#Node": map[string]any{"type": "structure", "members": map[string]any{
				"Child": ref("example#Node"),
			}},
		},
	})
	node := doc["components"].(map[string]any)["schemas"].(map[string]any)["Node"].(map[string]any)
	child := node["properties"].(map[string]any)["Child"].(map[string]any)
	if got, want := child["$ref"], "#/components/schemas/Node"; got != want {
		t.Fatalf("recursive ref = %v, want %v", got, want)
	}
}

func TestCatalogAWSArtifactsRemainSmithyReviewArtifacts(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "catalog-openapi-cache", "openapi", "aws-lambda-smithy-model.json"),
		filepath.Join("..", "catalog-openapi-cache", "openapi", "aws-s3-smithy-model.json"),
		filepath.Join("..", "catalog-openapi-cache", "openapi", "aws-sns-smithy-model.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				t.Skipf("catalog review artifact %s is not present", path)
			}
			t.Fatalf("read %s: %v", path, err)
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if raw["smithy"] != "2.0" {
			t.Fatalf("%s smithy = %v, want 2.0", path, raw["smithy"])
		}
		if raw["openapi"] != nil || raw["swagger"] != nil {
			t.Fatalf("%s is unexpectedly OpenAPI-shaped", path)
		}
		if _, err := awssmithy.Convert(data); err != nil {
			t.Fatalf("convert %s: %v", path, err)
		}
		assertConvertedOperationCount(t, path, data, raw)
	}
}

func restJSONFixture() map[string]any {
	return map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.lambda#Lambda": serviceShape("Lambda", "restJson1", []any{ref("com.amazonaws.lambda#GetFunction")}),
			"com.amazonaws.lambda#GetFunction": map[string]any{
				"type":   "operation",
				"input":  ref("com.amazonaws.lambda#GetFunctionRequest"),
				"output": ref("com.amazonaws.lambda#GetFunctionResponse"),
				"traits": map[string]any{
					"smithy.api#documentation": "Returns information about the function.",
					"smithy.api#http":          map[string]any{"method": "GET", "uri": "/2015-03-31/functions/{FunctionName}", "code": float64(200)},
				},
			},
			"com.amazonaws.lambda#GetFunctionRequest": map[string]any{"type": "structure", "members": map[string]any{
				"FunctionName": member("com.amazonaws.lambda#FunctionName", map[string]any{"smithy.api#httpLabel": map[string]any{}, "smithy.api#required": map[string]any{}}),
				"Qualifier":    member("com.amazonaws.lambda#Qualifier", map[string]any{"smithy.api#httpQuery": "Qualifier"}),
			}},
			"com.amazonaws.lambda#GetFunctionResponse": map[string]any{"type": "structure", "members": map[string]any{
				"Configuration": ref("com.amazonaws.lambda#FunctionConfiguration"),
			}},
			"com.amazonaws.lambda#FunctionConfiguration": map[string]any{"type": "structure", "members": map[string]any{
				"FunctionArn": ref("com.amazonaws.lambda#String"),
			}},
			"com.amazonaws.lambda#FunctionName": map[string]any{"type": "string"},
			"com.amazonaws.lambda#Qualifier":    map[string]any{"type": "string"},
			"com.amazonaws.lambda#String":       map[string]any{"type": "string"},
		},
	}
}

func restXMLFixture() map[string]any {
	return map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.s3#AmazonS3": serviceShape("S3", "restXml", []any{ref("com.amazonaws.s3#PutObject")}),
			"com.amazonaws.s3#PutObject": map[string]any{
				"type":  "operation",
				"input": ref("com.amazonaws.s3#PutObjectRequest"),
				"traits": map[string]any{
					"smithy.api#http": map[string]any{"method": "PUT", "uri": "/{Bucket}/{Key+}", "code": float64(200)},
				},
			},
			"com.amazonaws.s3#PutObjectRequest": map[string]any{"type": "structure", "members": map[string]any{
				"Bucket":      member("com.amazonaws.s3#BucketName", map[string]any{"smithy.api#httpLabel": map[string]any{}, "smithy.api#required": map[string]any{}}),
				"Key":         member("com.amazonaws.s3#ObjectKey", map[string]any{"smithy.api#httpLabel": map[string]any{}, "smithy.api#required": map[string]any{}}),
				"Body":        member("smithy.api#Blob", map[string]any{"smithy.api#httpPayload": map[string]any{}}),
				"ContentType": member("smithy.api#String", map[string]any{"smithy.api#httpHeader": "Content-Type"}),
				"Metadata":    member("com.amazonaws.s3#Metadata", map[string]any{"smithy.api#httpPrefixHeaders": "x-amz-meta-"}),
			}},
			"com.amazonaws.s3#BucketName": map[string]any{"type": "string"},
			"com.amazonaws.s3#Metadata":   map[string]any{"type": "map", "key": ref("smithy.api#String"), "value": ref("smithy.api#String")},
			"com.amazonaws.s3#ObjectKey":  map[string]any{"type": "string"},
		},
	}
}

func awsQueryFixture() map[string]any {
	return map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.sns#SNS": serviceShape("SNS", "awsQuery", []any{ref("com.amazonaws.sns#Publish")}),
			"com.amazonaws.sns#Publish": map[string]any{
				"type":  "operation",
				"input": ref("com.amazonaws.sns#PublishInput"),
			},
			"com.amazonaws.sns#PublishInput": map[string]any{"type": "structure", "members": map[string]any{
				"TopicArn": member("smithy.api#String", map[string]any{"smithy.api#required": map[string]any{}}),
				"Message":  member("smithy.api#String", map[string]any{"smithy.api#required": map[string]any{}}),
			}},
		},
	}
}

func serviceShape(name, protocol string, operations []any) map[string]any {
	traits := map[string]any{
		"aws.api#service": map[string]any{
			"sdkId":          name,
			"endpointPrefix": strings.ToLower(name),
		},
		"aws.auth#sigv4":            map[string]any{"name": strings.ToLower(name)},
		"smithy.api#title":          "AWS " + name,
		"aws.protocols#" + protocol: map[string]any{},
	}
	return map[string]any{
		"type":       "service",
		"version":    "2010-03-31",
		"operations": operations,
		"traits":     traits,
	}
}

func ref(target string) map[string]any {
	return map[string]any{"target": target}
}

func member(target string, traits map[string]any) map[string]any {
	return map[string]any{"target": target, "traits": traits}
}

func mustConvertMap(t *testing.T, raw map[string]any) map[string]any {
	t.Helper()
	doc, err := awssmithy.ConvertMap(raw)
	if err != nil {
		t.Fatalf("ConvertMap failed: %v", err)
	}
	return doc
}

func mustJSON(t *testing.T, raw map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

func operation(t *testing.T, doc map[string]any, path, method string) map[string]any {
	t.Helper()
	paths := doc["paths"].(map[string]any)
	item, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("missing path %q in %#v", path, paths)
	}
	op, ok := item[method].(map[string]any)
	if !ok {
		t.Fatalf("missing operation %s %s in %#v", method, path, item)
	}
	return op
}

func assertConvertedOperationCount(t *testing.T, path string, data []byte, raw map[string]any) {
	t.Helper()
	source := 0
	for _, shape := range raw["shapes"].(map[string]any) {
		if stringValue(mapValue(shape)["type"]) == "operation" {
			source++
		}
	}
	out, err := awssmithy.Convert(data)
	if err != nil {
		t.Fatalf("convert %s: %v", path, err)
	}
	var converted map[string]any
	if err := json.Unmarshal(out, &converted); err != nil {
		t.Fatalf("parse converted %s: %v", path, err)
	}
	actual := convertedOperationCount(converted)
	if actual != source {
		t.Fatalf("%s converted operations = %d, want %d", path, actual, source)
	}
}

func convertedOperationCount(doc map[string]any) int {
	count := 0
	for _, itemValue := range doc["paths"].(map[string]any) {
		item := itemValue.(map[string]any)
		for method := range item {
			switch method {
			case "get", "put", "post", "delete", "options", "head", "patch", "trace":
				count++
			}
		}
	}
	return count
}

func assertParameter(t *testing.T, params []any, in, name string, required bool) {
	t.Helper()
	for _, param := range params {
		m := param.(map[string]any)
		if m["in"] == in && m["name"] == name {
			if m["required"] != required {
				t.Fatalf("%s %s required = %v, want %v", in, name, m["required"], required)
			}
			return
		}
	}
	t.Fatalf("missing %s parameter %q in %#v", in, name, params)
}

func mapValue(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func nestedString(root map[string]any, keys ...string) string {
	current := root
	for i, key := range keys {
		if i == len(keys)-1 {
			value, _ := current[key].(string)
			return value
		}
		current, _ = current[key].(map[string]any)
	}
	return ""
}

func stringSlice(v any) []string {
	if values, ok := v.([]string); ok {
		return append([]string(nil), values...)
	}
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
