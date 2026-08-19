package catalog

import "testing"

func TestCatalogIndexProvidesIndependentIndexedLookups(t *testing.T) {
	index := BuiltInCatalogIndex()
	provider, ok := index.FindProvider("grafana http api")
	if !ok || provider.ID != "grafana" {
		t.Fatalf("provider = %#v, %v", provider, ok)
	}
	ref, ok := index.FindSpecReference("grafana", "grafana-http-api-openapi-v3")
	if !ok || ref.ID != "grafana-http-api-openapi-v3" {
		t.Fatalf("spec ref = %#v, %v", ref, ok)
	}
	overlays := index.SecurityOverlaysForProvider("slack")
	if len(overlays) == 0 {
		t.Fatal("missing indexed Slack overlays")
	}
	security, ok := index.SecurityForProvider("grafana")
	if !ok || security.Status != AuthStatusComplete {
		t.Fatalf("security = %#v, %v", security, ok)
	}

	provider.SpecReferences[0].ID = "mutated"
	overlays[0].ID = "mutated"
	security.Dispositions[0].SourceRefs[0] = "mutated"
	freshProvider, _ := index.FindProvider("grafana")
	freshOverlays := index.SecurityOverlaysForProvider("slack")
	freshSecurity, _ := index.SecurityForProvider("grafana")
	if freshProvider.SpecReferences[0].ID == "mutated" || freshOverlays[0].ID == "mutated" || freshSecurity.Dispositions[0].SourceRefs[0] == "mutated" {
		t.Fatal("catalog index leaked mutable slices")
	}
}

func TestNewCatalogIndexRejectsInvalidCatalog(t *testing.T) {
	provider := qualityProvider("demo")
	provider.SpecReferences[0].ID = ""
	if _, err := NewCatalogIndex(Catalog{Providers: []Provider{provider}}, nil); err == nil {
		t.Fatal("NewCatalogIndex accepted invalid catalog")
	}
}

func TestCatalogIndexRejectsProviderAliasForExactSpecLookup(t *testing.T) {
	if _, ok := BuiltInCatalogIndex().FindSpecReference("Grafana HTTP API", "grafana-http-api-openapi-v3"); ok {
		t.Fatal("FindSpecReference accepted provider alias")
	}
}
