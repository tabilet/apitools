package apitools

import (
	"reflect"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestBuildCatalogStatsReportIsDeterministic(t *testing.T) {
	alpha := catalogStatsProvider("alpha", catalog.SpecKindOpenAPI)
	zeta := catalogStatsProvider("zeta", catalog.SpecKindGraphQL)
	artifacts := []catalog.CatalogSpecArtifact{
		{ProviderID: "zeta", SpecRefID: "zeta-source", Kind: "graphql"},
		{ProviderID: "alpha", SpecRefID: "alpha-source", Kind: "openapi"},
	}
	results := []CatalogSpecRefreshReviewResult{
		{ProviderID: "zeta", SpecRefID: "zeta-source", Protocol: catalog.SpecProtocolGraphQL, Status: CatalogRefreshMissingFile, RegisteredArtifactPath: "graphql/zeta.graphql"},
		{ProviderID: "alpha", SpecRefID: "alpha-source", Protocol: catalog.SpecProtocolOpenAPI, Status: CatalogRefreshReviewAvailable, RawValidationStatus: CatalogRefreshValidOpenAPI, Exists: true, Bytes: 42},
	}

	first := BuildCatalogStatsReport([]catalog.Provider{zeta, alpha}, artifacts, CatalogSpecRefreshReviewReport{Results: results})
	second := BuildCatalogStatsReport([]catalog.Provider{alpha, zeta}, []catalog.CatalogSpecArtifact{artifacts[1], artifacts[0]}, CatalogSpecRefreshReviewReport{Results: []CatalogSpecRefreshReviewResult{results[1], results[0]}})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("stats depend on input order:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.ProviderCount != 2 || first.ArtifactRegistry.ArtifactCount != 2 || first.Refresh.RefreshableSpecCount != 2 || first.Refresh.RegisteredArtifactCount != 1 || first.Refresh.ExistingArtifactCount != 1 || first.Refresh.MissingRegisteredFileCount != 1 || first.Refresh.TotalBytes != 42 {
		t.Fatalf("unexpected stats totals = %#v", first)
	}
}

func TestCatalogStatsProtocolOrderCoversEverySupportedFamily(t *testing.T) {
	want := []catalog.SpecProtocol{
		catalog.SpecProtocolOpenAPI,
		catalog.SpecProtocolSwagger,
		catalog.SpecProtocolSmithy,
		catalog.SpecProtocolGoogleDiscovery,
		catalog.SpecProtocolAsyncAPI,
		catalog.SpecProtocolOpenRPC,
		catalog.SpecProtocolGraphQL,
		catalog.SpecProtocolGRPCProtobuf,
		catalog.SpecProtocolOData,
		catalog.SpecProtocolDropboxStone,
		catalog.SpecProtocolOpenAPIIndex,
		catalog.SpecProtocolHumanDocs,
		catalog.SpecProtocolUnknown,
	}
	if got := catalogProtocolOrder(); !reflect.DeepEqual(got, want) {
		t.Fatalf("protocol order = %#v, want %#v", got, want)
	}
}

func TestCatalogStatsPrefersMachineSourceOverHumanDocs(t *testing.T) {
	provider := catalogStatsProvider("demo", catalog.SpecKindHumanDocs)
	provider.SpecReferences = append(provider.SpecReferences, catalog.SpecReference{ID: "demo-smithy", Kind: catalog.SpecKindSmithyJSON})
	report := BuildCatalogStatsReport([]catalog.Provider{provider}, nil, CatalogSpecRefreshReviewReport{})
	for _, bucket := range report.Protocols {
		if bucket.Protocol == catalog.SpecProtocolSmithy {
			if bucket.Count != 1 || !reflect.DeepEqual(bucket.ProviderIDs, []string{"demo"}) {
				t.Fatalf("Smithy bucket = %#v", bucket)
			}
			return
		}
	}
	t.Fatal("missing Smithy protocol bucket")
}

func catalogStatsProvider(id string, kind catalog.SpecKind) catalog.Provider {
	return catalog.Provider{
		ID: id,
		SpecReferences: []catalog.SpecReference{{
			ID:   id + "-source",
			Kind: kind,
		}},
	}
}
