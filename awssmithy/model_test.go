package awssmithy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/awssmithy"
)

func TestParseSupportsAWSProtocolFamilies(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fixture    map[string]any
		protocol   string
		operation  string
		mediaType  string
		staticHead string
	}{
		{name: "restJson1", fixture: restJSONFixture(), protocol: "restJson1", operation: "GetFunction", mediaType: "application/json"},
		{name: "restXml", fixture: restXMLFixture(), protocol: "restXml", operation: "PutObject", mediaType: "application/octet-stream"},
		{name: "awsQuery", fixture: awsQueryFixture(), protocol: "awsQuery", operation: "Publish", mediaType: "application/x-www-form-urlencoded"},
		{name: "ec2Query", fixture: ec2QueryFixture(), protocol: "ec2Query", operation: "DescribeInstances", mediaType: "application/x-www-form-urlencoded"},
		{name: "awsJson1_0", fixture: awsJSONFixture("awsJson1_0"), protocol: "awsJson1_0", operation: "PutItem", mediaType: "application/x-amz-json-1.0", staticHead: "DynamoDB.PutItem"},
		{name: "awsJson1_1", fixture: awsJSONFixture("awsJson1_1"), protocol: "awsJson1_1", operation: "PutItem", mediaType: "application/x-amz-json-1.1", staticHead: "DynamoDB.PutItem"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model, err := awssmithy.Parse(mustJSON(t, tc.fixture))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if got := model.Protocol; got != tc.protocol {
				t.Fatalf("protocol = %q, want %q", got, tc.protocol)
			}
			op, ok := model.OperationByName(tc.operation)
			if !ok {
				t.Fatalf("missing operation %q", tc.operation)
			}
			if got := op.RequestMediaType; got != tc.mediaType {
				t.Fatalf("request media = %q, want %q", got, tc.mediaType)
			}
			if tc.staticHead != "" {
				if got := op.StaticHeaders["X-Amz-Target"]; got != tc.staticHead {
					t.Fatalf("X-Amz-Target = %q, want %q", got, tc.staticHead)
				}
			}
		})
	}
}

func TestParsePreservesRuntimeHTTPBindings(t *testing.T) {
	model, err := awssmithy.Parse(mustJSON(t, restXMLFixture()))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	op, ok := model.OperationByName("PutObject")
	if !ok {
		t.Fatal("missing PutObject")
	}
	if len(op.GreedyLabels) != 1 || op.GreedyLabels[0] != "Key" {
		t.Fatalf("greedy labels = %#v, want Key", op.GreedyLabels)
	}
	if op.Payload == nil || op.Payload.MemberName != "Body" {
		t.Fatalf("payload binding = %#v, want Body", op.Payload)
	}
	var metadata *awssmithy.MemberBinding
	for _, binding := range op.InputBindings {
		if binding != nil && binding.MemberName == "Metadata" {
			metadata = binding
			break
		}
	}
	if metadata == nil {
		t.Fatal("missing Metadata binding")
	}
	if got, want := metadata.Location, "prefixHeaders"; got != want {
		t.Fatalf("metadata location = %q, want %q", got, want)
	}
	if got, want := metadata.WireName, "x-amz-meta-"; got != want {
		t.Fatalf("metadata prefix = %q, want %q", got, want)
	}
}

func TestParsePreservesResponseHTTPBindings(t *testing.T) {
	model, err := awssmithy.Parse(mustJSON(t, lambdaInvokeFixture()))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	op, ok := model.OperationByName("Invoke")
	if !ok {
		t.Fatal("missing Invoke")
	}
	seen := map[string]string{}
	for _, binding := range op.OutputBindings {
		if binding != nil {
			seen[binding.MemberName] = binding.Location
		}
	}
	for member, location := range map[string]string{
		"StatusCode":    "responseCode",
		"FunctionError": "header",
		"Payload":       "payload",
	} {
		if got := seen[member]; got != location {
			t.Fatalf("%s location = %q, want %q; bindings %#v", member, got, location, seen)
		}
	}
}

func TestParsePreservesProtocolDistinctOperationPaths(t *testing.T) {
	model, err := awssmithy.Parse(mustJSON(t, map[string]any{
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
	}))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	acl, ok := model.OperationByName("GetBucketAcl")
	if !ok {
		t.Fatal("missing GetBucketAcl")
	}
	if got, want := acl.Path, "/{Bucket}?acl"; got != want {
		t.Fatalf("acl path = %q, want %q", got, want)
	}
	tagging, ok := model.OperationByName("GetBucketTagging")
	if !ok {
		t.Fatal("missing GetBucketTagging")
	}
	if got, want := tagging.Path, "/{Bucket}?tagging"; got != want {
		t.Fatalf("tagging path = %q, want %q", got, want)
	}

	queryModel, err := awssmithy.Parse(mustJSON(t, map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.sns#SNS": serviceShape("SNS", "awsQuery", []any{
				ref("com.amazonaws.sns#Publish"),
				ref("com.amazonaws.sns#Subscribe"),
			}),
			"com.amazonaws.sns#Publish":   map[string]any{"type": "operation"},
			"com.amazonaws.sns#Subscribe": map[string]any{"type": "operation"},
		},
	}))
	if err != nil {
		t.Fatalf("Parse query model failed: %v", err)
	}
	publish, ok := queryModel.OperationByName("Publish")
	if !ok {
		t.Fatal("missing Publish")
	}
	if got, want := publish.Path, "/?Action=Publish"; got != want {
		t.Fatalf("publish path = %q, want %q", got, want)
	}
	subscribe, ok := queryModel.OperationByName("Subscribe")
	if !ok {
		t.Fatal("missing Subscribe")
	}
	if got, want := subscribe.Path, "/?Action=Subscribe"; got != want {
		t.Fatalf("subscribe path = %q, want %q", got, want)
	}
}

func TestParseRejectsMalformedSmithy(t *testing.T) {
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
			_, err := awssmithy.Parse(mustJSON(t, tc.raw))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want containing %q", err, tc.want)
			}
		})
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
		if _, err := awssmithy.Parse(data); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
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

func ec2QueryFixture() map[string]any {
	return map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.ec2#AmazonEC2": serviceShape("EC2", "ec2Query", []any{ref("com.amazonaws.ec2#DescribeInstances")}),
			"com.amazonaws.ec2#DescribeInstances": map[string]any{
				"type":  "operation",
				"input": ref("com.amazonaws.ec2#DescribeInstancesRequest"),
			},
			"com.amazonaws.ec2#DescribeInstancesRequest": map[string]any{"type": "structure", "members": map[string]any{
				"InstanceIds": member("com.amazonaws.ec2#InstanceIdList", nil),
			}},
			"com.amazonaws.ec2#InstanceIdList": map[string]any{"type": "list", "member": ref("smithy.api#String")},
		},
	}
}

func awsJSONFixture(protocol string) map[string]any {
	return map[string]any{
		"smithy": "2.0",
		"shapes": map[string]any{
			"com.amazonaws.dynamodb#DynamoDB": serviceShape("DynamoDB", protocol, []any{ref("com.amazonaws.dynamodb#PutItem")}),
			"com.amazonaws.dynamodb#PutItem": map[string]any{
				"type":  "operation",
				"input": ref("com.amazonaws.dynamodb#PutItemInput"),
			},
			"com.amazonaws.dynamodb#PutItemInput": map[string]any{"type": "structure", "members": map[string]any{
				"TableName": member("smithy.api#String", map[string]any{"smithy.api#required": map[string]any{}}),
			}},
		},
	}
}

func lambdaInvokeFixture() map[string]any {
	raw := restJSONFixture()
	shapes := raw["shapes"].(map[string]any)
	service := shapes["com.amazonaws.lambda#Lambda"].(map[string]any)
	service["operations"] = []any{ref("com.amazonaws.lambda#Invoke")}
	shapes["com.amazonaws.lambda#Invoke"] = map[string]any{
		"type":   "operation",
		"input":  ref("com.amazonaws.lambda#InvokeRequest"),
		"output": ref("com.amazonaws.lambda#InvokeResponse"),
		"traits": map[string]any{
			"smithy.api#http": map[string]any{"method": "POST", "uri": "/2015-03-31/functions/{FunctionName}/invocations", "code": float64(200)},
		},
	}
	shapes["com.amazonaws.lambda#InvokeRequest"] = map[string]any{"type": "structure", "members": map[string]any{
		"FunctionName": member("com.amazonaws.lambda#FunctionName", map[string]any{"smithy.api#httpLabel": map[string]any{}, "smithy.api#required": map[string]any{}}),
		"Payload":      member("com.amazonaws.lambda#Blob", map[string]any{"smithy.api#httpPayload": map[string]any{}}),
	}}
	shapes["com.amazonaws.lambda#InvokeResponse"] = map[string]any{"type": "structure", "members": map[string]any{
		"StatusCode":    member("smithy.api#Integer", map[string]any{"smithy.api#httpResponseCode": map[string]any{}}),
		"FunctionError": member("smithy.api#String", map[string]any{"smithy.api#httpHeader": "X-Amz-Function-Error"}),
		"Payload":       member("com.amazonaws.lambda#Blob", map[string]any{"smithy.api#httpPayload": map[string]any{}}),
	}}
	shapes["com.amazonaws.lambda#Blob"] = map[string]any{"type": "blob"}
	return raw
}

func serviceShape(name, protocol string, operations []any) map[string]any {
	return map[string]any{
		"type":       "service",
		"version":    "2010-03-31",
		"operations": operations,
		"traits": map[string]any{
			"aws.api#service": map[string]any{
				"sdkId":          name,
				"endpointPrefix": strings.ToLower(name),
			},
			"aws.auth#sigv4":            map[string]any{"name": strings.ToLower(name)},
			"smithy.api#title":          "AWS " + name,
			"aws.protocols#" + protocol: map[string]any{},
		},
	}
}

func ref(target string) map[string]any {
	return map[string]any{"target": target}
}

func member(target string, traits map[string]any) map[string]any {
	return map[string]any{"target": target, "traits": traits}
}

func mustJSON(t *testing.T, raw map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}
