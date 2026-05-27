package apitools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/apitools/catalog"
)

type uws14CorpusArtifact struct {
	providerID string
	specRefID  string
	kind       catalog.SpecKind
	protocol   catalog.SpecProtocol
	url        string
	path       string
}

func TestUWS14SourceArtifactCorpusValidates(t *testing.T) {
	for _, artifact := range uws14SourceArtifactCorpus() {
		t.Run(artifact.specRefID, func(t *testing.T) {
			content := readUWS14CorpusArtifact(t, artifact.path)
			ref := artifact.refreshableRef()

			status, metadata, correctionNotes, err := validateCatalogRefreshContentWithCorrections(context.Background(), ref, content, artifact.url)
			if err != nil {
				t.Fatalf("validateCatalogRefreshContentWithCorrections() error = %v", err)
			}
			if status != CatalogRefreshValidStructured {
				t.Fatalf("status = %q, want %q", status, CatalogRefreshValidStructured)
			}
			if got := catalogRefreshProtocolClassification(ref, metadata).Protocol; got != artifact.protocol {
				t.Fatalf("protocol = %q, want %q", got, artifact.protocol)
			}
			if metadata.OperationCount == 0 {
				t.Fatalf("operation count is zero")
			}
			if len(correctionNotes) != 0 {
				t.Fatalf("correction notes = %#v, want none", correctionNotes)
			}
			if len(content) == 0 {
				t.Fatalf("artifact bytes = 0")
			}
			sum := sha256.Sum256(content)
			if digest := hex.EncodeToString(sum[:]); len(digest) != 64 {
				t.Fatalf("sha256 length = %d, want 64", len(digest))
			}
		})
	}
}

func TestUWS14SourceArtifactCorpusRefreshReview(t *testing.T) {
	artifacts := uws14SourceArtifactCorpus()
	refs := make([]catalog.RefreshableSpecReference, 0, len(artifacts))
	wantProtocol := map[string]catalog.SpecProtocol{}
	for _, artifact := range artifacts {
		ref := artifact.refreshableRef()
		refs = append(refs, ref)
		wantProtocol[ref.ProviderID+"/"+ref.SpecRefID] = artifact.protocol
	}

	report, err := BuildCatalogSpecRefreshReviewReport(refs, CatalogSpecRefreshReviewOptions{
		CacheDir:              "catalog-openapi-cache",
		AsOf:                  time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
		StaleVerificationDays: 365,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(report.Results), len(artifacts); got != want {
		t.Fatalf("result len = %d, want %d", got, want)
	}
	for _, result := range report.Results {
		key := result.ProviderID + "/" + result.SpecRefID
		protocol, ok := wantProtocol[key]
		if !ok {
			t.Fatalf("unexpected result %s", key)
		}
		if !result.Exists {
			t.Fatalf("%s exists = false, want true", key)
		}
		if result.RegisteredArtifactPath == "" {
			t.Fatalf("%s registered artifact path is empty", key)
		}
		if result.ValidationStatus != CatalogRefreshValidStructured {
			t.Fatalf("%s status = %q, want %q; error=%q", key, result.ValidationStatus, CatalogRefreshValidStructured, result.ValidationError)
		}
		if result.Protocol != protocol {
			t.Fatalf("%s protocol = %q, want %q", key, result.Protocol, protocol)
		}
		if result.Bytes == 0 {
			t.Fatalf("%s bytes = 0", key)
		}
		if len(result.SHA256) != 64 {
			t.Fatalf("%s sha256 length = %d, want 64", key, len(result.SHA256))
		}
		if result.Metadata.OperationCount == 0 {
			t.Fatalf("%s operation count is zero", key)
		}
	}
}

func readUWS14CorpusArtifact(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("catalog-openapi-cache", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func (artifact uws14CorpusArtifact) refreshableRef() catalog.RefreshableSpecReference {
	return catalog.RefreshableSpecReference{
		ProviderID:             artifact.providerID,
		ProviderName:           artifact.providerID,
		SpecRefID:              artifact.specRefID,
		Kind:                   artifact.kind,
		Protocol:               artifact.protocol,
		UWSSourceType:          catalog.SpecProtocolClassification{Protocol: artifact.protocol}.UWSSourceType(),
		URL:                    artifact.url,
		SourceAuthority:        catalog.SourceAuthorityOfficialGitHub,
		VerifiedAt:             "2026-05-27",
		RegisteredArtifactPath: artifact.path,
	}
}

func uws14SourceArtifactCorpus() []uws14CorpusArtifact {
	return []uws14CorpusArtifact{
		{
			providerID: "graphql-examples",
			specRefID:  "graphiql-starwars-schema",
			kind:       catalog.SpecKindGraphQL,
			protocol:   catalog.SpecProtocolGraphQL,
			url:        "https://raw.githubusercontent.com/graphql/graphiql/main/packages/graphql-language-service/src/interface/__tests__/__schema__/StarWarsSchema.graphql",
			path:       "graphql/graphiql-starwars-schema.graphql",
		},
		{
			providerID: "graphql-examples",
			specRefID:  "graphiql-schema-kitchen-sink",
			kind:       catalog.SpecKindGraphQL,
			protocol:   catalog.SpecProtocolGraphQL,
			url:        "https://raw.githubusercontent.com/graphql/graphiql/main/packages/codemirror-graphql/src/__tests__/schema-kitchen-sink.graphql",
			path:       "graphql/graphiql-schema-kitchen-sink.graphql",
		},
		{
			providerID: "openrpc-examples",
			specRefID:  "openrpc-petstore",
			kind:       catalog.SpecKindOpenRPC,
			protocol:   catalog.SpecProtocolOpenRPC,
			url:        "https://raw.githubusercontent.com/open-rpc/examples/master/service-descriptions/petstore-openrpc.json",
			path:       "openrpc/openrpc-petstore.json",
		},
		{
			providerID: "openrpc-examples",
			specRefID:  "openrpc-simple-math",
			kind:       catalog.SpecKindOpenRPC,
			protocol:   catalog.SpecProtocolOpenRPC,
			url:        "https://raw.githubusercontent.com/open-rpc/examples/master/service-descriptions/simple-math-openrpc.json",
			path:       "openrpc/openrpc-simple-math.json",
		},
		{
			providerID: "grpc-protobuf-examples",
			specRefID:  "google-pubsub-v1",
			kind:       catalog.SpecKindGRPCProtobuf,
			protocol:   catalog.SpecProtocolGRPCProtobuf,
			url:        "https://raw.githubusercontent.com/googleapis/googleapis/master/google/pubsub/v1/pubsub.proto",
			path:       "grpc-protobuf/google-pubsub-v1.proto",
		},
		{
			providerID: "grpc-protobuf-examples",
			specRefID:  "opentelemetry-trace-service-v1",
			kind:       catalog.SpecKindGRPCProtobuf,
			protocol:   catalog.SpecProtocolGRPCProtobuf,
			url:        "https://raw.githubusercontent.com/open-telemetry/opentelemetry-proto/main/opentelemetry/proto/collector/trace/v1/trace_service.proto",
			path:       "grpc-protobuf/opentelemetry-trace-service-v1.proto",
		},
		{
			providerID: "odata-examples",
			specRefID:  "odata-csdl-16-1",
			kind:       catalog.SpecKindOData,
			protocol:   catalog.SpecProtocolOData,
			url:        "https://raw.githubusercontent.com/oasis-tcs/odata-csdl-schemas/master/examples/csdl-16.1.json",
			path:       "odata/odata-csdl-16-1.json",
		},
		{
			providerID: "odata-examples",
			specRefID:  "odata-operations-service",
			kind:       catalog.SpecKindOData,
			protocol:   catalog.SpecProtocolOData,
			url:        "https://raw.githubusercontent.com/OData/odata.net/main/test/EndToEndTests/Common/Microsoft.OData.E2E.TestCommon/Common/Client/Operations/OperationsServiceCsdl.xml",
			path:       "odata/odata-operations-service.xml",
		},
	}
}
