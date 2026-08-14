package apitools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDiscoverLocalSourcesFindsEverySupportedFamily(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"openapi/pets.yaml": `openapi: 3.0.3
info:
  title: Pets
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: ok
`,
		"google-discovery/drive.json": `{
  "kind":"discovery#restDescription", "discoveryVersion":"v1",
  "name":"drive", "version":"v3", "title":"Drive", "rootUrl":"https://example.invalid/", "servicePath":"drive/v3/",
  "resources":{"files":{"methods":{"get":{"id":"drive.files.get","path":"files/{fileId}","httpMethod":"GET"}}}}
}`,
		"aws-smithy/iam.json": `{
  "smithy":"2.0",
  "shapes":{
		"com.amazonaws.iam#IAM":{"type":"service","version":"2010-05-08","operations":[{"target":"com.amazonaws.iam#CreateRole"}],"traits":{"aws.api#service":{"sdkId":"IAM","endpointPrefix":"iam"},"aws.auth#sigv4":{"name":"iam"},"aws.protocols#awsQuery":{}}},
		"com.amazonaws.iam#CreateRole":{"type":"operation"}
  }
}`,
		"asyncapi/events.yaml": `asyncapi: 3.0.0
info:
  title: Events
  version: 1.0.0
operations:
  publishEvent:
    action: send
    messages:
      - $ref: '#/channels/events/messages/event'
channels:
  events:
    messages:
      event:
        payload:
          type: object
`,
		"graphql/service.graphql": `type Query { pet(id: ID!): Pet }
type Pet { id: ID! }
`,
		"openrpc/service.json": `{
  "openrpc":"1.3.2", "info":{"title":"RPC","version":"1.0.0"},
  "methods":[{"name":"pet.get","params":[],"result":{"name":"pet","schema":{"type":"object"}}}]
}`,
		"grpc-protobuf/service.proto": `syntax = "proto3";
package pets.v1;
message GetPetRequest { string id = 1; }
message Pet { string id = 1; }
service Pets { rpc GetPet(GetPetRequest) returns (Pet); }
`,
		"odata/metadata.xml": `<Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm">
  <EntityType Name="Pet"><Property Name="ID" Type="Edm.String" Nullable="false"/></EntityType>
  <EntityContainer Name="Container"><EntitySet Name="Pets" EntityType="Demo.Pet"/></EntityContainer>
</Schema>`,
	}
	for name, content := range files {
		writeLocalDiscoveryFile(t, dir, name, content)
	}

	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}, Query: "pet"})
	if err != nil {
		t.Fatalf("DiscoverLocalSources() error = %v", err)
	}
	if report.Version != LocalSourceDiscoveryVersion || report.Truncated {
		t.Fatalf("report metadata = %#v", report)
	}
	wantKinds := map[string]bool{
		APISourceKindOpenAPI: true, APISourceKindGoogleDiscovery: true,
		APISourceKindAWSSmithy: true, APISourceKindAsyncAPI: true,
		APISourceKindGraphQL: true, APISourceKindOpenRPC: true,
		APISourceKindGRPCProtobuf: true, APISourceKindOData: true,
	}
	if len(report.Candidates) != len(wantKinds) {
		t.Fatalf("candidates = %#v; rejected=%#v ambiguous=%#v", report.Candidates, report.Rejected, report.Ambiguous)
	}
	for _, candidate := range report.Candidates {
		delete(wantKinds, candidate.Kind)
		if candidate.SourceFamily != candidate.Kind || len(candidate.SHA256) != 64 || candidate.Bytes == 0 || !strings.HasPrefix(candidate.Provenance, "local:") {
			t.Fatalf("candidate evidence = %#v", candidate)
		}
		if candidate.OperationCount == 0 {
			t.Fatalf("candidate operation count = 0: %#v", candidate)
		}
	}
	if len(wantKinds) != 0 {
		t.Fatalf("missing kinds = %#v", wantKinds)
	}
}

func TestDiscoverLocalSourcesDeduplicatesContent(t *testing.T) {
	dir := t.TempDir()
	content := `{"openrpc":"1.3.2","info":{"title":"RPC","version":"1"},"methods":[{"name":"ping","params":[],"result":{"name":"pong","schema":{"type":"string"}}}]}`
	writeLocalDiscoveryFile(t, dir, "a.json", content)
	writeLocalDiscoveryFile(t, dir, "nested/b.json", content)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 || len(report.Candidates[0].DuplicatePaths) != 1 {
		t.Fatalf("deduplicated candidates = %#v", report.Candidates)
	}
}

func TestDiscoverLocalSourcesRequiresExplicitKindForAmbiguousJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeLocalDiscoveryFile(t, dir, "metadata.json", `{"name":"custom","methods":[]}`)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Ambiguous) != 1 || len(report.Candidates) != 0 {
		t.Fatalf("report = %#v", report)
	}

	report, err = DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{
		Sources: []LocalSource{{Kind: APISourceKindOpenRPC, ID: "rpc", Path: path}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rejected) != 1 || report.Rejected[0].Kind != APISourceKindOpenRPC {
		t.Fatalf("explicit invalid report = %#v", report)
	}
}

func TestDiscoverLocalSourcesExplicitKindAndID(t *testing.T) {
	dir := t.TempDir()
	path := writeLocalDiscoveryFile(t, dir, "service.data", `{"openrpc":"1.3.2","info":{"title":"RPC","version":"1"},"methods":[{"name":"ping","params":[],"result":{"name":"pong","schema":{"type":"string"}}}]}`)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{
		Sources: []LocalSource{{Kind: "openrpc", ID: "billing", Path: path}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].ID != "billing" || report.Candidates[0].Kind != APISourceKindOpenRPC {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
}

func TestDiscoverLocalSourcesReportsBoundsAndOversizedFiles(t *testing.T) {
	dir := t.TempDir()
	writeLocalDiscoveryFile(t, dir, "openrpc/too-big.json", strings.Repeat("x", 64))
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}, MaxBytes: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rejected) != 1 || report.Rejected[0].Code != "file.too_large" {
		t.Fatalf("oversized report = %#v", report)
	}

	dir = t.TempDir()
	for _, name := range []string{"a.json", "b.json"} {
		writeLocalDiscoveryFile(t, dir, name, `{"openrpc":"1.3.2","info":{"title":"`+name+`","version":"1"},"methods":[{"name":"ping","params":[],"result":{"name":"pong","schema":{"type":"string"}}}]}`)
	}
	report, err = DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}, MaxCandidates: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "local.limit.candidates" {
		t.Fatalf("candidate limit report = %#v", report)
	}

	report, err = DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}, MaxVisitedEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "local.limit.entries" {
		t.Fatalf("entry limit report = %#v", report)
	}
}

func TestDiscoverLocalSourcesRejectsSymlinkRootsAndHonorsCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges vary on Windows")
	}
	dir := t.TempDir()
	target := writeLocalDiscoveryFile(t, dir, "openrpc.json", `{"openrpc":"1.3.2","info":{"title":"RPC","version":"1"},"methods":[]}`)
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{link}}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink root error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := DiscoverLocalSources(ctx, LocalSourceDiscoveryOptions{Roots: []string{dir}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestDiscoverLocalSourcesAcceptsStructurallyValidAuthoringOpenAPI(t *testing.T) {
	dir := t.TempDir()
	writeLocalDiscoveryFile(t, dir, "minimal.json", `{"openapi":"3.0.3","info":{"title":"Minimal","version":"1"},"paths":{"/items":{"get":{"operationId":"listItems"}}}}`)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].OperationCount != 1 {
		t.Fatalf("candidates = %#v; rejected = %#v", report.Candidates, report.Rejected)
	}
}

func TestDiscoverLocalSourcesRejectsAuthoringOpenAPIWithoutOperationID(t *testing.T) {
	dir := t.TempDir()
	writeLocalDiscoveryFile(t, dir, "invalid.json", `{"openapi":"3.0.3","info":{"title":"Invalid","version":"1"},"paths":{"/items":{"get":{}}}}`)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 0 || len(report.Rejected) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestDiscoverLocalSourcesIgnoresSecurityOverlaySidecars(t *testing.T) {
	dir := t.TempDir()
	writeLocalDiscoveryFile(t, dir, "service.json", `{"openapi":"3.0.3","info":{"title":"Service","version":"1"},"paths":{"/items":{"get":{"operationId":"listItems"}}}}`)
	writeLocalDiscoveryFile(t, dir, "service.security-overlay.json", `{"operations":{"listItems":{"side_effect":"read"}}}`)
	report, err := DiscoverLocalSources(context.Background(), LocalSourceDiscoveryOptions{Roots: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 || len(report.Ambiguous) != 0 || len(report.Rejected) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func writeLocalDiscoveryFile(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
