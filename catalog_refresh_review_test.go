package apitools

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/apitools/catalog"
)

func TestCatalogSpecRefreshReviewReportsMissingRegistration(t *testing.T) {
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)}, CatalogSpecRefreshReviewOptions{
		CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshMissingRegistration {
		t.Fatalf("status = %q, want %q", result.Status, CatalogRefreshMissingRegistration)
	}
	if result.Protocol != catalog.SpecProtocolOpenAPI {
		t.Fatalf("protocol = %q, want openapi", result.Protocol)
	}
	if result.Exists {
		t.Fatalf("exists = true, want false")
	}
	if len(result.ManualFollowUps) == 0 {
		t.Fatalf("missing manual follow-ups")
	}
}

func TestCatalogSpecRefreshReviewReportsMissingRegisteredFile(t *testing.T) {
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshMissingFile {
		t.Fatalf("status = %q, want %q", result.Status, CatalogRefreshMissingFile)
	}
	if result.SavedPath == "" {
		t.Fatalf("saved path is empty")
	}
}

func TestCatalogSpecRefreshReviewValidOpenAPI(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshReviewAvailable {
		t.Fatalf("file status = %q, want %q", result.Status, CatalogRefreshReviewAvailable)
	}
	if result.RawValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshValidOpenAPI)
	}
	if result.Protocol != catalog.SpecProtocolOpenAPI || result.ProtocolVersion != "3.0.0" {
		t.Fatalf("protocol = %q %q, want openapi 3.0.0", result.Protocol, result.ProtocolVersion)
	}
	if result.Bytes == 0 || result.SHA256 == "" || !result.Exists {
		t.Fatalf("missing file evidence: %#v", result)
	}
}

func TestCatalogSpecRefreshReviewSeparatesRawAndCorrectedValidation(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("highlevel", catalog.SpecKindOpenAPI)
	ref.ProviderID = "highlevel"
	ref.SpecRefID = "highlevel-contacts-openapi"
	ref.RegisteredArtifactPath = "openapi/highlevel-contacts.json"
	content := []byte(`{
		"openapi":"3.0.0",
		"info":{"title":"Contacts API","version":"1.0"},
		"paths":{"/contacts":{"get":{"responses":{"200":{"description":""},"400":{"description":"Bad Request","content":{"application/json":{"schema":{"$ref":"../common/common-schemas.json#/components/schemas/BadRequestDTO"}}}}}}}},
		"components":{"schemas":{"NestedArray":{"type":"array","items":{"type":"array"}}}}
	}`)
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, string(content))

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshReviewAvailable {
		t.Fatalf("file status = %q, want %q", result.Status, CatalogRefreshReviewAvailable)
	}
	if result.RawValidationStatus == CatalogRefreshValidOpenAPI || result.RawValidationError == "" {
		t.Fatalf("raw validation was mislabeled as corrected: %#v", result)
	}
	if result.CorrectedValidationStatus != CatalogRefreshValidOpenAPI || result.CorrectedValidationError != "" {
		t.Fatalf("corrected validation = %q error %q", result.CorrectedValidationStatus, result.CorrectedValidationError)
	}
	if result.CorrectedMetadata == nil || result.CorrectedMetadata.Title != "Contacts API" || len(result.CorrectionNotes) == 0 {
		t.Fatalf("corrected evidence = %#v", result)
	}
	digest := sha256.Sum256(content)
	if result.SHA256 != hex.EncodeToString(digest[:]) || result.Bytes != int64(len(content)) {
		t.Fatalf("raw provenance = sha256 %q bytes %d", result.SHA256, result.Bytes)
	}
}

func TestCatalogSpecRefreshReviewReportsParseableInvalidOpenAPI(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RawValidationStatus != CatalogRefreshParseableOpenAPIInvalid {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshParseableOpenAPIInvalid)
	}
	if result.RawValidationError == "" {
		t.Fatalf("validation error is empty")
	}
	if result.RawMetadata.Title != "Demo" || result.RawMetadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.RawMetadata)
	}
	if !hasRefreshReviewFollowUp(result, "strict validation errors") {
		t.Fatalf("missing strict validation follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewValidStructuredArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind catalog.SpecKind
		body string
	}{
		{name: "google", kind: catalog.SpecKindGoogleDiscovery, body: `{"title":"Google","version":"v1","resources":{}}`},
		{name: "index", kind: catalog.SpecKindOpenAPIIndex, body: `{"title":"Index","apis":{"demo":{"openapi":"demo.yaml"}}}`},
		{name: "smithy", kind: catalog.SpecKindSmithyJSON, body: `{"smithy":"2.0","shapes":{}}`},
		{name: "asyncapi", kind: catalog.SpecKindAsyncAPI, body: `{"asyncapi":"3.0.0","info":{"title":"Events","version":"1.0.0"},"operations":{}}`},
		{name: "openrpc", kind: catalog.SpecKindOpenRPC, body: `{"openrpc":"1.3.2","info":{"title":"Pet RPC","version":"1.0.0"},"methods":[{"name":"pet.get"}]}`},
		{name: "graphql", kind: catalog.SpecKindGraphQL, body: `type Query { issue(id: ID!): Issue }`},
		{name: "grpc-protobuf", kind: catalog.SpecKindGRPCProtobuf, body: `syntax = "proto3"; package issues.v1; service IssueService { rpc GetIssue(GetIssueRequest) returns (Issue); } message GetIssueRequest { string id = 1; }`},
		{name: "odata", kind: catalog.SpecKindOData, body: `<Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm"><EntityContainer Name="Container"><EntitySet Name="Products" EntityType="Demo.Product"/></EntityContainer></Schema>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ref := refreshReviewTestRef(tc.name, tc.kind)
			ref.RegisteredArtifactPath = "openapi/" + tc.name + ".json"
			switch tc.kind {
			case catalog.SpecKindGoogleDiscovery:
				ref.RegisteredArtifactPath = "google-discovery/" + tc.name + ".json"
			case catalog.SpecKindSmithyJSON:
				ref.RegisteredArtifactPath = "aws-smithy/" + tc.name + ".json"
			case catalog.SpecKindAsyncAPI:
				ref.RegisteredArtifactPath = "asyncapi/" + tc.name + ".json"
			case catalog.SpecKindOpenRPC:
				ref.RegisteredArtifactPath = "openrpc/" + tc.name + ".json"
			case catalog.SpecKindGraphQL:
				ref.RegisteredArtifactPath = "graphql/" + tc.name + ".graphql"
			case catalog.SpecKindGRPCProtobuf:
				ref.RegisteredArtifactPath = "grpc-protobuf/" + tc.name + ".proto"
			case catalog.SpecKindOData:
				ref.RegisteredArtifactPath = "odata/" + tc.name + ".xml"
			}
			writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, tc.body)

			report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
			if err != nil {
				t.Fatal(err)
			}
			result := singleRefreshReviewResult(t, report)
			if result.RawValidationStatus != CatalogRefreshValidStructured {
				t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshValidStructured)
			}
			if result.Protocol != refreshReviewTestProtocol(tc.kind) {
				t.Fatalf("protocol = %q, want %q", result.Protocol, refreshReviewTestProtocol(tc.kind))
			}
		})
	}
}

func TestCatalogSpecRefreshReviewRejectsStructuredNonOpenRPCArtifact(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("openrpc", catalog.SpecKindOpenRPC)
	ref.RegisteredArtifactPath = "openrpc/openrpc.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"title":"Not OpenRPC","methods":[]}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RawValidationStatus != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshInvalid)
	}
	if !strings.Contains(result.RawValidationError, "does not validate as OpenRPC") {
		t.Fatalf("validation error = %q, want OpenRPC validation error", result.RawValidationError)
	}
}

func TestCatalogSpecRefreshReviewNonOpenAPIMachineSpec(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("stone", catalog.SpecKindDropboxStone)
	ref.RegisteredArtifactPath = "openapi/stone.tar.gz"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, "stone module content")

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RawValidationStatus != CatalogRefreshSkippedValidation {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshSkippedValidation)
	}
	if result.Protocol != catalog.SpecProtocolDropboxStone {
		t.Fatalf("protocol = %q, want dropbox-stone", result.Protocol)
	}
}

func TestCatalogSpecRefreshReviewDiscoversUnregisteredSavedFile(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	writeRefreshReviewArtifact(t, dir, "openapi/demo.json", `{"openapi":"3.1.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RegisteredArtifactPath != "" {
		t.Fatalf("registered artifact path = %q, want empty", result.RegisteredArtifactPath)
	}
	if result.RawValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshValidOpenAPI)
	}
	if result.SavedPath == "" {
		t.Fatalf("saved path is empty")
	}
}

func TestCatalogSpecRefreshReviewDiscoversAlternateSavedFilename(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("calendar-discovery-v3", catalog.SpecKindGoogleDiscovery)
	ref.ProviderID = "google-calendar"
	ref.ProviderName = "Google Calendar"
	writeRefreshReviewArtifact(t, dir, "google-discovery/google-calendar-discovery-v3.json", `{"title":"Calendar","version":"v3","resources":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RawValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshValidStructured)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.SavedPath), "google-discovery/google-calendar-discovery-v3.json") {
		t.Fatalf("saved path = %q", result.SavedPath)
	}
	if !hasRefreshReviewFollowUp(result, "Register the saved artifact path") {
		t.Fatalf("missing registration follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewDiscoversUnregisteredSmithySavedFile(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("aws-s3-smithy-model", catalog.SpecKindSmithyJSON)
	ref.ProviderID = "aws-s3"
	ref.ProviderName = "Amazon S3"
	writeRefreshReviewArtifact(t, dir, "aws-smithy/aws-s3-smithy-model.json", `{"smithy":"2.0","shapes":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.RawValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("status = %q, want %q", result.RawValidationStatus, CatalogRefreshValidStructured)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.SavedPath), "aws-smithy/aws-s3-smithy-model.json") {
		t.Fatalf("saved path = %q", result.SavedPath)
	}
	if !hasRefreshReviewFollowUp(result, "Register the saved artifact path") {
		t.Fatalf("missing registration follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewReportsStaleVerificationDate(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.VerifiedAt = "2024-01-01"
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)
	asOf, err := time.Parse("2006-01-02", "2026-05-19")
	if err != nil {
		t.Fatal(err)
	}

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir:              dir,
		AsOf:                  asOf,
		StaleVerificationDays: 365,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if !result.VerificationStale {
		t.Fatalf("verification stale = false, want true")
	}
	if !hasRefreshReviewFollowUp(result, "stale verification date 2024-01-01") {
		t.Fatalf("missing stale follow-up: %#v", result.ManualFollowUps)
	}
}

func TestCatalogSpecRefreshReviewRejectsSymlinkArtifact(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"openapi":"3.0.0","info":{"title":"Outside","version":"1.0.0"},"paths":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "openapi", "demo.json")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{CacheDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", result.Status, CatalogRefreshInvalid)
	}
	if !strings.Contains(result.StatusError, "symlink") {
		t.Fatalf("artifact error = %q, want symlink", result.StatusError)
	}
}

func TestCatalogSpecRefreshReviewRejectsOversizedArtifact(t *testing.T) {
	dir := t.TempDir()
	ref := refreshReviewTestRef("demo", catalog.SpecKindOpenAPI)
	ref.RegisteredArtifactPath = "openapi/demo.json"
	writeRefreshReviewArtifact(t, dir, ref.RegisteredArtifactPath, `{"openapi":"3.0.0","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)

	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{ref}, CatalogSpecRefreshReviewOptions{
		CacheDir: dir,
		MaxBytes: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := singleRefreshReviewResult(t, report)
	if result.Status != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", result.Status, CatalogRefreshInvalid)
	}
	if !strings.Contains(result.StatusError, "over limit 8") {
		t.Fatalf("artifact error = %q, want size limit", result.StatusError)
	}
	if result.Bytes <= 8 {
		t.Fatalf("bytes = %d, want original size over limit", result.Bytes)
	}
}

func TestCatalogSpecRefreshReviewDeterministicOrdering(t *testing.T) {
	report, err := BuildCatalogSpecRefreshReviewReport([]catalog.RefreshableSpecReference{
		refreshReviewTestRef("zeta", catalog.SpecKindOpenAPI),
		refreshReviewTestRef("alpha", catalog.SpecKindOpenAPI),
	}, CatalogSpecRefreshReviewOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := report.Results[0].ProviderID+"/"+report.Results[0].SpecRefID, "alpha/alpha"; got != want {
		t.Fatalf("first result = %q, want %q", got, want)
	}
	if got, want := report.Results[1].ProviderID+"/"+report.Results[1].SpecRefID, "zeta/zeta"; got != want {
		t.Fatalf("second result = %q, want %q", got, want)
	}
}

func refreshReviewTestRef(id string, kind catalog.SpecKind) catalog.RefreshableSpecReference {
	return catalog.RefreshableSpecReference{
		ProviderID:      id,
		ProviderName:    id,
		SpecRefID:       id,
		Kind:            kind,
		URL:             "https://example.com/" + id + ".json",
		SourceAuthority: catalog.SourceAuthorityOfficialProvider,
		VerifiedAt:      "2026-05-19",
	}
}

func writeRefreshReviewArtifact(t *testing.T, cacheDir, artifactPath, content string) {
	t.Helper()
	fullPath := filepath.Join(cacheDir, filepath.FromSlash(artifactPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func singleRefreshReviewResult(t *testing.T, report CatalogSpecRefreshReviewReport) CatalogSpecRefreshReviewResult {
	t.Helper()
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("result len = %d, want %d", got, want)
	}
	return report.Results[0]
}

func hasRefreshReviewFollowUp(result CatalogSpecRefreshReviewResult, text string) bool {
	for _, followUp := range result.ManualFollowUps {
		if strings.Contains(followUp, text) {
			return true
		}
	}
	return false
}

func refreshReviewTestProtocol(kind catalog.SpecKind) catalog.SpecProtocol {
	switch kind {
	case catalog.SpecKindGoogleDiscovery:
		return catalog.SpecProtocolGoogleDiscovery
	case catalog.SpecKindOpenAPIIndex:
		return catalog.SpecProtocolOpenAPIIndex
	case catalog.SpecKindSmithyJSON:
		return catalog.SpecProtocolSmithy
	case catalog.SpecKindAsyncAPI:
		return catalog.SpecProtocolAsyncAPI
	case catalog.SpecKindOpenRPC:
		return catalog.SpecProtocolOpenRPC
	case catalog.SpecKindGraphQL:
		return catalog.SpecProtocolGraphQL
	case catalog.SpecKindGRPCProtobuf:
		return catalog.SpecProtocolGRPCProtobuf
	case catalog.SpecKindOData:
		return catalog.SpecProtocolOData
	default:
		return catalog.SpecProtocolUnknown
	}
}
