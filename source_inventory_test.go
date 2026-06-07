package apitools

import (
	"context"
	"testing"
)

func TestBuildAPISourceOperationInventoryNativeSources(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		content   string
		operation string
	}{
		{
			name:      "aws smithy",
			kind:      APISourceKindAWSSmithy,
			operation: "CreateRole",
			content: `{
  "smithy": "2.0",
  "shapes": {
    "com.amazonaws.iam#IAM": {
      "type": "service",
      "version": "2010-05-08",
      "operations": [{"target": "com.amazonaws.iam#CreateRole"}],
      "traits": {
        "aws.api#service": {"sdkId": "IAM", "endpointPrefix": "iam"},
        "aws.auth#sigv4": {"name": "iam"},
        "aws.protocols#awsQuery": {}
      }
    },
    "com.amazonaws.iam#CreateRole": {
      "type": "operation",
      "input": {"target": "com.amazonaws.iam#CreateRoleRequest"}
    },
    "com.amazonaws.iam#CreateRoleRequest": {
      "type": "structure",
      "members": {
        "RoleName": {
          "target": "com.amazonaws.iam#roleNameType",
          "traits": {"smithy.api#required": {}}
        }
      },
      "traits": {"smithy.api#input": {}}
    },
    "com.amazonaws.iam#roleNameType": {"type": "string"}
  }
}`,
		},
		{
			name:      "google discovery",
			kind:      APISourceKindGoogleDiscovery,
			operation: "storage.buckets.insert",
			content: `{
  "discoveryVersion": "v1",
  "name": "storage",
  "version": "v1",
  "rootUrl": "https://storage.googleapis.com/",
  "servicePath": "storage/v1/",
  "schemas": {
    "Bucket": {
      "id": "Bucket",
      "type": "object",
      "properties": {"name": {"type": "string"}}
    }
  },
  "resources": {
    "buckets": {
      "methods": {
        "insert": {
          "id": "storage.buckets.insert",
          "path": "b",
          "httpMethod": "POST",
          "parameters": {
            "project": {"type": "string", "required": true, "location": "query"}
          },
          "request": {"$ref": "Bucket"},
          "response": {"$ref": "Bucket"},
          "scopes": ["https://www.googleapis.com/auth/devstorage.full_control"]
        }
      }
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventory, err := BuildAPISourceOperationInventory(context.Background(), APISourceInventoryOptions{
				Documents: []APISourceDocument{{Kind: tt.kind, Name: "source", Content: []byte(tt.content)}},
			})
			if err != nil {
				t.Fatalf("BuildAPISourceOperationInventory() error = %v", err)
			}
			index, err := NewOperationIndex(inventory)
			if err != nil {
				t.Fatalf("NewOperationIndex() error = %v diagnostics=%#v", err, inventory.Diagnostics)
			}
			if _, ok := index.LookupOperationID(tt.operation); !ok {
				t.Fatalf("operation %q missing from %#v", tt.operation, index.OperationIDs)
			}
			if tt.kind == APISourceKindGoogleDiscovery {
				op, ok := index.LookupOperationID(tt.operation)
				if !ok || op.ResponseBody == nil || op.ResponseBody.Ref != "Bucket" {
					t.Fatalf("google response body = %#v", op.ResponseBody)
				}
				if names := responseFieldNames(op.ResponseBody.Fields); len(names) != 1 || names[0] != "name" {
					t.Fatalf("google response fields = %#v", op.ResponseBody.Fields)
				}
			}
		})
	}
}
