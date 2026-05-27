package apitools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestRefreshCatalogSpecReferencesDownloadsValidOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Refresh Test","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{"description":"ok"}}}}}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("len(results) = %d, want %d", got, want)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidOpenAPI {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolOpenAPI || result.ProtocolVersion != "3.0.0" {
		t.Fatalf("protocol = %q %q, want openapi 3.0.0", result.Protocol, result.ProtocolVersion)
	}
	if result.Metadata.Title != "Refresh Test" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if result.ArtifactPath != "openapi/test-openapi.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("saved artifact: %v", err)
	}
	if result.SHA256 == "" || result.Bytes == 0 {
		t.Fatalf("missing hash/bytes: %#v", result)
	}
	if len(result.ManualFollowUps) == 0 {
		t.Fatalf("missing manual follow-ups")
	}
}

func TestRefreshCatalogSpecReferencesSavesParseableInvalidOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"Parseable Invalid","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-parseable-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshParseableOpenAPIInvalid {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.ValidationError == "" {
		t.Fatalf("ValidationError is empty")
	}
	if result.Metadata.Title != "Parseable Invalid" || result.Metadata.OpenAPI != "3.0.0" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("saved artifact: %v", err)
	}
	if !hasRefreshFollowUp(result, "strict validation errors") {
		t.Fatalf("missing strict validation follow-up: %#v", result.ManualFollowUps)
	}
}

func TestRefreshCatalogSpecReferencesSavesParseableInvalidOpenAPIWithoutInfoVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openapi":"3.1.0","info":{"title":"Missing Info Version"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-missing-info-version-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshParseableOpenAPIInvalid {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Metadata.Title != "Missing Info Version" || result.Metadata.OpenAPI != "3.1.0" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
}

func TestRefreshCatalogSpecReferencesSavesParseableInvalidSwagger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"swagger":"2.0","info":{"title":"Parseable Swagger","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{}}}}}}`))
	}))
	defer server.Close()

	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-parseable-swagger",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/swagger.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshParseableSwaggerInvalid {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolSwagger || result.ProtocolVersion != "2.0" {
		t.Fatalf("protocol = %q %q, want swagger 2.0", result.Protocol, result.ProtocolVersion)
	}
	if result.ValidationError == "" {
		t.Fatalf("ValidationError is empty")
	}
	if result.Metadata.Title != "Parseable Swagger" || result.Metadata.Swagger != "2.0" || result.Metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", result.Metadata)
	}
}

func TestRefreshCatalogSpecReferencesAppliesHighLevelCorrection(t *testing.T) {
	content := `{
		"openapi":"3.0.0",
		"info":{"title":"Contacts API","version":"1.0"},
		"paths":{
			"/contacts/search/duplicate":{"get":{"responses":{"200":{"description":""},"400":{"description":"Bad Request","content":{"application/json":{"schema":{"$ref":"../common/common-schemas.json#/components/schemas/BadRequestDTO"}}}}}}}
		},
		"components":{"schemas":{"NestedArray":{"type":"array","items":{"type":"array"}}}}
	}`
	status, metadata, notes, err := validateCatalogRefreshContentWithCorrections(context.Background(), catalog.RefreshableSpecReference{
		ProviderID: "highlevel",
		SpecRefID:  "highlevel-contacts-openapi",
		Kind:       catalog.SpecKindOpenAPI,
	}, []byte(content), "https://raw.githubusercontent.com/GoHighLevel/highlevel-api-docs/main/apps/contacts.json")
	if err != nil {
		t.Fatalf("validateCatalogRefreshContentWithCorrections() error = %v", err)
	}
	if status != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", status, CatalogRefreshValidOpenAPI)
	}
	if metadata.Title != "Contacts API" || metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "HighLevel") {
		t.Fatalf("correction notes = %#v", notes)
	}
}

func TestRefreshCatalogSpecReferencesAppliesStravaCorrection(t *testing.T) {
	content := `{
		"swagger":"2.0",
		"info":{"title":"Strava API v3","version":"3.0.0"},
		"paths":{"/activities":{"get":{"responses":{"200":{"description":"ok","schema":{"$ref":"https://developers.strava.com/swagger/activity.json#/DetailedActivity"}}}}}},
		"securityDefinitions":{"strava_oauth":{"type":"oauth2","flow":"accessCode","authorizationUrl":"https://www.strava.com/oauth/authorize","tokenUrl":"https://www.strava.com/oauth/token","scopes":{"read":"Read data"}}},
		"security":[{"strava_oauth":["public"]}]
	}`
	status, metadata, notes, err := validateCatalogRefreshContentWithCorrections(context.Background(), catalog.RefreshableSpecReference{
		ProviderID: "strava",
		SpecRefID:  "strava-api-v3-swagger",
		Kind:       catalog.SpecKindOpenAPI,
	}, []byte(content), "https://developers.strava.com/swagger/swagger.json")
	if err != nil {
		t.Fatalf("validateCatalogRefreshContentWithCorrections() error = %v", err)
	}
	if status != CatalogRefreshValidSwagger {
		t.Fatalf("status = %q, want %q", status, CatalogRefreshValidSwagger)
	}
	if metadata.Title != "Strava API v3" || metadata.Swagger != "2.0" || metadata.OperationCount != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "Strava") {
		t.Fatalf("correction notes = %#v", notes)
	}
}

func TestRefreshCatalogSpecReferencesAppliesSpotifyCorrection(t *testing.T) {
	content := `
openapi: "3.0.3"
info:
  title: Spotify Web API
  version: "1.0.0"
paths:
  /me/tracks:
    put:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required:
                - uris
              properties:
                ids:
                  type: array
                  items:
                    type: string
      responses:
        "200":
          description: ok
  /playlists/{playlist_id}/images:
    put:
      parameters:
        - name: playlist_id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          image/jpeg:
            schema:
              type: string
              format: byte
              required: true
      responses:
        "202":
          description: ok
components:
  x-spotify-policy:
    policies:
      $ref: ../policies.yaml
  schemas:
    PagingObject:
      type: object
      required:
        - href
        - items
      properties:
        href:
          type: string
`
	status, metadata, notes, err := validateCatalogRefreshContentWithCorrections(context.Background(), catalog.RefreshableSpecReference{
		ProviderID: "spotify",
		SpecRefID:  "spotify-web-api-openapi",
		Kind:       catalog.SpecKindOpenAPI,
	}, []byte(content), "https://developer.spotify.com/reference/web-api/open-api-schema.yaml")
	if err != nil {
		t.Fatalf("validateCatalogRefreshContentWithCorrections() error = %v", err)
	}
	if status != CatalogRefreshValidOpenAPI {
		t.Fatalf("status = %q, want %q", status, CatalogRefreshValidOpenAPI)
	}
	if metadata.Title != "Spotify Web API" || metadata.OperationCount != 2 {
		t.Fatalf("metadata = %#v", metadata)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "Spotify") {
		t.Fatalf("correction notes = %#v", notes)
	}
}

func TestRefreshCatalogSpecReferencesDoesNotApplyCorrectionsToOtherProviders(t *testing.T) {
	content := `{"openapi":"3.0.0","info":{"title":"External Ref","version":"1.0.0"},"paths":{"/items":{"get":{"responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"../common/common-schemas.json#/components/schemas/BadRequestDTO"}}}}}}}}`
	status, _, notes, err := validateCatalogRefreshContentWithCorrections(context.Background(), catalog.RefreshableSpecReference{
		ProviderID: "other",
		SpecRefID:  "other-openapi",
		Kind:       catalog.SpecKindOpenAPI,
	}, []byte(content), "https://example.com/openapi.json")
	if err == nil {
		t.Fatalf("validateCatalogRefreshContentWithCorrections() expected error")
	}
	if status != CatalogRefreshInvalid {
		t.Fatalf("status = %q, want %q", status, CatalogRefreshInvalid)
	}
	if len(notes) != 0 {
		t.Fatalf("correction notes = %#v, want none", notes)
	}
}

func TestRefreshCatalogSpecReferencesRejectsUnparseableOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"info":{"title":"Not OpenAPI","version":"1.0.0"},"paths":{}}`))
	}))
	defer server.Close()

	_, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-invalid",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "does not validate as OpenAPI or Swagger") {
		t.Fatalf("err = %v, want OpenAPI validation error", err)
	}
}

func TestRefreshCatalogSpecReferencesRejectsUnsafeHost(t *testing.T) {
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := (&Client{HTTPClient: server.Client()}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "test",
		SpecRefID:  "test-openapi",
		Kind:       catalog.SpecKindOpenAPI,
		URL:        server.URL + "/openapi.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "refusing private URL host") {
		t.Fatalf("err = %v, want unsafe host rejection", err)
	}
	if requested {
		t.Fatal("unsafe refresh reached server")
	}
}

func hasRefreshFollowUp(result CatalogSpecRefreshResult, text string) bool {
	for _, followUp := range result.ManualFollowUps {
		if strings.Contains(followUp, text) {
			return true
		}
	}
	return false
}

func TestRefreshCatalogSpecReferencesValidatesStructuredNonOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"discoveryVersion":"v1","title":"Discovery Test","version":"v1"}`))
	}))
	defer server.Close()

	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID:             "google-test",
		SpecRefID:              "google-test-discovery",
		Kind:                   catalog.SpecKindGoogleDiscovery,
		URL:                    server.URL + "/discovery/rest",
		RegisteredArtifactPath: "openapi/google-test-discovery.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolGoogleDiscovery {
		t.Fatalf("protocol = %q, want google-discovery", result.Protocol)
	}
	if result.ArtifactPath != "google-discovery/google-test-discovery.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if result.Metadata.Title != "Discovery Test" {
		t.Fatalf("Metadata.Title = %q", result.Metadata.Title)
	}
}

func TestRefreshCatalogSpecReferencesSavesSmithyArtifactUnderAWSSmithy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"smithy":"2.0","shapes":{}}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID:             "aws-s3",
		SpecRefID:              "aws-s3-smithy-model",
		Kind:                   catalog.SpecKindSmithyJSON,
		URL:                    server.URL + "/aws-s3-smithy-model.json",
		RegisteredArtifactPath: "openapi/aws-s3-smithy-model.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolSmithy {
		t.Fatalf("protocol = %q, want smithy", result.Protocol)
	}
	if result.ArtifactPath != "aws-smithy/aws-s3-smithy-model.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("expected saved artifact: %v", err)
	}
}

func TestRefreshCatalogSpecReferencesSavesAsyncAPIArtifactUnderAsyncAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`asyncapi: 3.0.0
info:
  title: Events
  version: 1.0.0
operations: {}
`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID:             "events",
		SpecRefID:              "events-asyncapi",
		Kind:                   catalog.SpecKindAsyncAPI,
		URL:                    server.URL + "/events.yaml",
		RegisteredArtifactPath: "openapi/events-asyncapi.yaml",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolAsyncAPI {
		t.Fatalf("protocol = %q, want asyncapi", result.Protocol)
	}
	if result.ArtifactPath != "asyncapi/events-asyncapi.yaml" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("expected saved artifact: %v", err)
	}
}

func TestRefreshCatalogSpecReferencesSavesOpenRPCArtifactUnderOpenRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"openrpc":"1.3.2","info":{"title":"Pet RPC","version":"1.0.0"},"methods":[{"name":"pet.get"}]}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID:             "pet",
		SpecRefID:              "pet-openrpc",
		Kind:                   catalog.SpecKindOpenRPC,
		URL:                    server.URL + "/pet-openrpc.json",
		RegisteredArtifactPath: "openapi/pet-openrpc.json",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolOpenRPC {
		t.Fatalf("protocol = %q, want openrpc", result.Protocol)
	}
	if result.ArtifactPath != "openrpc/pet-openrpc.json" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("expected saved artifact: %v", err)
	}
}

func TestRefreshCatalogSpecReferencesSavesGraphQLArtifactUnderGraphQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`type Query { issue(id: ID!): Issue }`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "issues",
		SpecRefID:  "issues-graphql",
		Kind:       catalog.SpecKindGraphQL,
		URL:        server.URL + "/schema.graphql",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolGraphQL {
		t.Fatalf("protocol = %q, want graphql", result.Protocol)
	}
	if result.ArtifactPath != "graphql/issues-graphql.graphql" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if result.Metadata.OperationCount != 1 {
		t.Fatalf("operation count = %d, want 1", result.Metadata.OperationCount)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("expected saved artifact: %v", err)
	}
}

func TestRefreshCatalogSpecReferencesSavesGRPCProtobufArtifactUnderGRPCProtobuf(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`syntax = "proto3"; package issues.v1; service IssueService { rpc GetIssue(GetIssueRequest) returns (Issue); } message GetIssueRequest { string id = 1; }`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	report, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "issues",
		SpecRefID:  "issues-proto",
		Kind:       catalog.SpecKindGRPCProtobuf,
		URL:        server.URL + "/issues.proto",
	}}, CatalogSpecRefreshOptions{CacheDir: cacheDir})
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.ValidationStatus != CatalogRefreshValidStructured {
		t.Fatalf("ValidationStatus = %q", result.ValidationStatus)
	}
	if result.Protocol != catalog.SpecProtocolGRPCProtobuf {
		t.Fatalf("protocol = %q, want grpc-protobuf", result.Protocol)
	}
	if result.ArtifactPath != "grpc-protobuf/issues-proto.proto" {
		t.Fatalf("ArtifactPath = %q", result.ArtifactPath)
	}
	if result.Metadata.OperationCount != 1 {
		t.Fatalf("operation count = %d, want 1", result.Metadata.OperationCount)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))); err != nil {
		t.Fatalf("expected saved artifact: %v", err)
	}
}

func TestRefreshCatalogSpecReferencesRejectsStructuredNonOpenRPCArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"title":"Not OpenRPC","methods":[]}`))
	}))
	defer server.Close()

	_, err := (&Client{HTTPClient: server.Client(), AllowUnsafeHosts: true}).RefreshCatalogSpecReferences(context.Background(), []catalog.RefreshableSpecReference{{
		ProviderID: "pet",
		SpecRefID:  "pet-openrpc",
		Kind:       catalog.SpecKindOpenRPC,
		URL:        server.URL + "/pet-openrpc.json",
	}}, CatalogSpecRefreshOptions{CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "does not validate as OpenRPC") {
		t.Fatalf("RefreshCatalogSpecReferences() error = %v, want OpenRPC validation error", err)
	}
}
