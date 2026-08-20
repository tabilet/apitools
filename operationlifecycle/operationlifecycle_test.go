package operationlifecycle

import (
	"slices"
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestExpandCollectionItemLifecycle(t *testing.T) {
	operations := []apitools.OperationSummary{
		op("k8s", "createCoreV1NamespacedConfigMap", "POST", "/api/v1/namespaces/{namespace}/configmaps"),
		op("k8s", "readCoreV1NamespacedConfigMap", "GET", "/api/v1/namespaces/{namespace}/configmaps/{name}"),
		op("k8s", "replaceCoreV1NamespacedConfigMap", "PUT", "/api/v1/namespaces/{namespace}/configmaps/{name}"),
		op("k8s", "deleteCoreV1NamespacedConfigMap", "DELETE", "/api/v1/namespaces/{namespace}/configmaps/{name}"),
	}
	expanded := Expand(operations, operations[0], Options{Goal: "create, read, update, and delete configmaps", DesiredState: true})
	if got := roleIDs(expanded); !slices.Equal(got, []string{"create:createCoreV1NamespacedConfigMap", "read:readCoreV1NamespacedConfigMap", "update:replaceCoreV1NamespacedConfigMap", "delete:deleteCoreV1NamespacedConfigMap"}) {
		t.Fatalf("roles = %#v", got)
	}
}

func TestDiscoveryUploadNormalizationRequiresExplicitProvenance(t *testing.T) {
	seed := op("storage", "storage.objects.insert", "POST", "/upload/storage/v1/b/{bucket}/o")
	read := op("storage", "storage.objects.get", "GET", "/storage/v1/b/{bucket}/o/{object}")
	if got := roleIDs(Expand([]apitools.OperationSummary{seed, read}, seed, Options{DesiredState: true})); !slices.Equal(got, []string{"post:storage.objects.insert"}) {
		t.Fatalf("unprovenanced roles = %#v", got)
	}
	seed.Extensions = map[string]string{"x-uws-source-kind": apitools.APISourceKindGoogleDiscovery}
	read.Extensions = map[string]string{"x-uws-source-kind": apitools.APISourceKindGoogleDiscovery}
	if got := roleIDs(Expand([]apitools.OperationSummary{seed, read}, seed, Options{DesiredState: true})); !slices.Equal(got, []string{"create:storage.objects.insert", "read:storage.objects.get"}) {
		t.Fatalf("Discovery roles = %#v", got)
	}
}

func TestExpandGoogleDotOperationIDs(t *testing.T) {
	operations := []apitools.OperationSummary{
		op("storage", "storage.buckets.insert", "POST", "/b"),
		op("storage", "storage.buckets.get", "GET", "/b/{bucket}"),
		op("storage", "storage.buckets.patch", "PATCH", "/b/{bucket}"),
		op("storage", "storage.buckets.update", "PUT", "/b/{bucket}"),
		op("storage", "storage.buckets.delete", "DELETE", "/b/{bucket}"),
	}
	expanded := Expand(operations, operations[0], Options{Goal: "create, read, update, and delete buckets", DesiredState: true})
	if got := roleIDs(expanded); !slices.Equal(got, []string{"create:storage.buckets.insert", "read:storage.buckets.get", "update:storage.buckets.patch", "delete:storage.buckets.delete"}) {
		t.Fatalf("roles = %#v", got)
	}
}

func TestNormalizePathScopesUploadHandling(t *testing.T) {
	if got := normalizePath("/upload/storage/v1/b/{bucket}/o", false); got != "/upload/storage/v1/b/{bucket}/o" {
		t.Fatalf("unprovenanced upload path = %q", got)
	}
	if got := normalizePath("/upload/storage/v1/b/{bucket}/o", true); got != "/storage/v1/b/{bucket}/o" {
		t.Fatalf("Discovery upload path = %q", got)
	}
	if got := normalizePath("/uploading/storage/v1/b/{bucket}/o", true); got != "/uploading/storage/v1/b/{bucket}/o" {
		t.Fatalf("uploading path = %q", got)
	}
}

func TestExpandHyphenAndCreateOrUpdateFamilies(t *testing.T) {
	cloudflare := []apitools.OperationSummary{
		op("cloudflare", "r2-create-bucket", "POST", "/accounts/{account_id}/r2/buckets"),
		op("cloudflare", "r2-get-bucket", "GET", "/accounts/{account_id}/r2/buckets/{bucket_name}"),
		op("cloudflare", "r2-patch-bucket", "PATCH", "/accounts/{account_id}/r2/buckets/{bucket_name}"),
		op("cloudflare", "r2-delete-bucket", "DELETE", "/accounts/{account_id}/r2/buckets/{bucket_name}"),
	}
	if got := roleIDs(Expand(cloudflare, cloudflare[0], Options{Goal: "create, read, update, and delete buckets", DesiredState: true})); !slices.Equal(got, []string{"create:r2-create-bucket", "read:r2-get-bucket", "update:r2-patch-bucket", "delete:r2-delete-bucket"}) {
		t.Fatalf("Cloudflare roles = %#v", got)
	}
	azure := []apitools.OperationSummary{
		op("azure", "Databases_CreateOrUpdate", "PUT", "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}"),
		op("azure", "Databases_Get", "GET", "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}"),
		op("azure", "Databases_Delete", "DELETE", "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases/{databaseName}"),
	}
	if got := roleIDs(Expand(azure, azure[0], Options{DesiredState: true})); !slices.Equal(got, []string{"create:Databases_CreateOrUpdate", "read:Databases_Get", "delete:Databases_Delete"}) {
		t.Fatalf("Azure roles = %#v", got)
	}
}

func TestExpandRejectsAmbiguousSibling(t *testing.T) {
	operations := []apitools.OperationSummary{op("widgets", "createWidget", "POST", "/widgets"), op("widgets", "getWidget", "GET", "/widgets/{id}"), op("widgets", "readWidget", "GET", "/widgets/{id}")}
	expanded := Expand(operations, operations[0], Options{DesiredState: true})
	if len(expanded.Diagnostics) != 1 || expanded.Diagnostics[0].Code != "operation_lifecycle.ambiguous_read" {
		t.Fatalf("diagnostics = %#v", expanded.Diagnostics)
	}
}

func TestExpandPreservesAPIFirstSingleOperationRoles(t *testing.T) {
	for _, test := range []struct{ id, method, role string }{
		{id: "createWidget", method: "POST", role: "post"},
		{id: "putWidget", method: "PUT", role: "put"},
		{id: "deleteWidget", method: "DELETE", role: "delete"},
	} {
		operation := op("api", test.id, test.method, "/widgets/{id}")
		if got := roleIDs(Expand([]apitools.OperationSummary{operation}, operation, Options{})); !slices.Equal(got, []string{test.role + ":" + test.id}) {
			t.Fatalf("%s roles = %#v", test.id, got)
		}
	}
}

func op(source, id, method, path string) apitools.OperationSummary {
	return apitools.OperationSummary{ID: id, OperationID: id, DocumentName: source, Method: method, Path: path}
}

func roleIDs(expanded Expansion) []string {
	out := make([]string, 0, len(expanded.Roles))
	for _, role := range expanded.Roles {
		out = append(out, role.Role+":"+role.Operation.OperationID)
	}
	return out
}
