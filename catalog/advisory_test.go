package catalog

import (
	"strings"
	"testing"
)

func TestBuiltInProviderAdvisoryReportDeterministic(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	if got, want := len(report.Providers), len(BuiltInProviders()); got != want {
		t.Fatalf("providers len = %d, want %d", got, want)
	}
	for i := 1; i < len(report.Providers); i++ {
		if report.Providers[i-1].ProviderID > report.Providers[i].ProviderID {
			t.Fatalf("providers not sorted at %d: %s before %s", i, report.Providers[i-1].ProviderID, report.Providers[i].ProviderID)
		}
	}
	if report.Providers[0].ProviderID != "action-network" {
		t.Fatalf("first provider = %q, want action-network", report.Providers[0].ProviderID)
	}
}

func TestBuiltInProviderAdvisoryReportSelectsProviderByAlias(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "grafana http api"})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	if got, want := len(report.Providers), 1; got != want {
		t.Fatalf("providers len = %d, want %d", got, want)
	}
	row := report.Providers[0]
	if row.ProviderID != "grafana" {
		t.Fatalf("provider id = %q, want grafana", row.ProviderID)
	}
	if row.AuthStatus != AuthStatusComplete {
		t.Fatalf("auth status = %q, want %q", row.AuthStatus, AuthStatusComplete)
	}
	if row.ResolvedOpenAPI.SpecRefID != "grafana-http-api-openapi-v3" {
		t.Fatalf("resolved spec = %q, want grafana-http-api-openapi-v3", row.ResolvedOpenAPI.SpecRefID)
	}
	for _, ref := range row.SpecReferences {
		if ref.ID == "grafana-http-api-openapi-v3" {
			if ref.Protocol != SpecProtocolOpenAPI || ref.ProtocolVersion != "3" {
				t.Fatalf("grafana protocol = %q %q, want openapi 3", ref.Protocol, ref.ProtocolVersion)
			}
			return
		}
	}
	t.Fatal("missing grafana spec reference")
}

func TestBuiltInProviderAdvisoryReportJoinsArtifactPath(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{
		ProviderKey: "slack",
		Artifacts: []CatalogSpecArtifact{
			{ProviderID: "slack", SpecRefID: "slack-web-openapi-v2", Kind: "openapi", Path: "openapi/slack-web-openapi-v2.json"},
			{ProviderID: "github", SpecRefID: "github-rest-api-openapi", Kind: "openapi", Path: "openapi/github.json"},
		},
	})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	row := report.Providers[0]
	if got, want := len(row.RegisteredArtifactPaths), 1; got != want {
		t.Fatalf("registered artifact len = %d, want %d", got, want)
	}
	if row.RegisteredArtifactPaths[0].Path != "openapi/slack-web-openapi-v2.json" {
		t.Fatalf("registered path = %q", row.RegisteredArtifactPaths[0].Path)
	}
	var found bool
	for _, ref := range row.SpecReferences {
		if ref.ID == "slack-web-openapi-v2" {
			found = true
			if ref.RegisteredArtifactPath != "openapi/slack-web-openapi-v2.json" {
				t.Fatalf("spec registered path = %q", ref.RegisteredArtifactPath)
			}
			if ref.Protocol != SpecProtocolSwagger || ref.ProtocolVersion != "2.0" {
				t.Fatalf("slack protocol = %q %q, want swagger 2.0", ref.Protocol, ref.ProtocolVersion)
			}
		}
	}
	if !found {
		t.Fatal("missing slack spec reference")
	}
}

func TestBuiltInProviderAdvisoryReportJoinsEndpointOverlay(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{
		ProviderKey: "activecampaign",
		Artifacts: []CatalogSpecArtifact{
			{
				ProviderID:  "activecampaign",
				SpecRefID:   "activecampaign-api-v3-overlay",
				ArtifactID:  "activecampaign-api-v3-overlay",
				Kind:        "advisory-overlay",
				Path:        "advisory-overlays/activecampaign-api-v3-overlay.json",
				OverlayPath: "advisory-overlays/activecampaign-api-v3-overlay.json",
				BuilderPath: "overlay-builders/build_m21_human_docs_overlays.go",
				Metadata:    map[string]string{"derived_from_docs": "true", "official_openapi": "false"},
			},
			{
				ProviderID:  "webflow",
				SpecRefID:   "webflow-data-api-v2-overlay",
				ArtifactID:  "webflow-data-api-v2-overlay",
				Kind:        "advisory-overlay",
				Path:        "advisory-overlays/webflow-data-api-v2-overlay.json",
				OverlayPath: "advisory-overlays/webflow-data-api-v2-overlay.json",
				BuilderPath: "overlay-builders/build_m21_human_docs_overlays.go",
			},
		},
	})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	row := report.Providers[0]
	if got, want := len(row.EndpointOverlays), 1; got != want {
		t.Fatalf("endpoint overlays len = %d, want %d", got, want)
	}
	overlay := row.EndpointOverlays[0]
	if overlay.ArtifactID != "activecampaign-api-v3-overlay" {
		t.Fatalf("endpoint overlay artifact = %q", overlay.ArtifactID)
	}
	if overlay.Path != "advisory-overlays/activecampaign-api-v3-overlay.json" {
		t.Fatalf("endpoint overlay path = %q", overlay.Path)
	}
	if overlay.BuilderPath != "overlay-builders/build_m21_human_docs_overlays.go" {
		t.Fatalf("endpoint overlay builder = %q", overlay.BuilderPath)
	}
	if overlay.Metadata["official_openapi"] != "false" {
		t.Fatalf("endpoint overlay metadata = %#v", overlay.Metadata)
	}
	if len(row.RegisteredArtifactPaths) != 0 {
		t.Fatalf("registered spec artifact paths = %#v, want none", row.RegisteredArtifactPaths)
	}
	assertHasFollowUp(t, row.ManualFollowUps, "Review registered advisory endpoint overlay metadata")
	assertNoFollowUp(t, row.ManualFollowUps, "OpenAPI-only workflows likely need a user-provided or generated OpenAPI document before import.")
}

func TestBuiltInProviderAdvisoryReportPreservesDropboxStoneAdvisoryOverlay(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{
		ProviderKey: "dropbox",
		Artifacts: []CatalogSpecArtifact{
			{
				ProviderID:  "dropbox",
				SpecRefID:   "dropbox-core-api-overlay",
				ArtifactID:  "dropbox-core-api-overlay",
				Kind:        "advisory-overlay",
				Path:        "advisory-overlays/dropbox-core-api-overlay.json",
				OverlayPath: "advisory-overlays/dropbox-core-api-overlay.json",
				BuilderPath: "overlay-builders/build_dropbox_overlay.go",
				Metadata: map[string]string{
					"derived_from_docs":                  "true",
					"derived_from_official_machine_spec": "true",
					"official_openapi":                   "false",
					"source_protocol":                    "dropbox-stone",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	row := report.Providers[0]
	if row.OpenAPIAvailability != SpecAvailabilityUnavailable {
		t.Fatalf("dropbox openapi availability = %q", row.OpenAPIAvailability)
	}
	var foundStone bool
	for _, ref := range row.SpecReferences {
		if ref.ID == "dropbox-api-stone-spec" {
			foundStone = true
			if ref.Protocol != SpecProtocolDropboxStone {
				t.Fatalf("dropbox protocol = %q, want %q", ref.Protocol, SpecProtocolDropboxStone)
			}
		}
	}
	if !foundStone {
		t.Fatal("missing dropbox stone spec reference")
	}
	if got, want := len(row.EndpointOverlays), 1; got != want {
		t.Fatalf("endpoint overlays len = %d, want %d", got, want)
	}
	overlay := row.EndpointOverlays[0]
	if overlay.Metadata["source_protocol"] != string(SpecProtocolDropboxStone) || overlay.Metadata["official_openapi"] != "false" {
		t.Fatalf("dropbox overlay metadata = %#v", overlay.Metadata)
	}
	assertHasFollowUp(t, row.ManualFollowUps, "Review registered advisory endpoint overlay metadata")
}

func TestHighValueSaaSCurationSetAdvisoryCoverage(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{
		Artifacts: []CatalogSpecArtifact{
			m46EndpointOverlayArtifact("airtable", "airtable-web-api-overlay"),
			m46EndpointOverlayArtifact("quickbooks", "quickbooks-online-accounting-api-overlay"),
			m46EndpointOverlayArtifact("salesforce", "salesforce-rest-core-overlay"),
			m46EndpointOverlayArtifact("servicenow", "servicenow-rest-api-overlay"),
			m46EndpointOverlayArtifact("shopify", "shopify-admin-rest-overlay"),
		},
	})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	rows := map[string]ProviderAdvisory{}
	for _, row := range report.Providers {
		rows[row.ProviderID] = row
	}
	tests := []struct {
		provider        string
		openAPI         SpecAvailability
		specKind        SpecKind
		endpointOverlay string
	}{
		{provider: "airtable", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs, endpointOverlay: "airtable-web-api-overlay"},
		{provider: "asana", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "box", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "discord", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "github", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "gitlab", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "hubspot", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPIIndex},
		{provider: "jira-cloud", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "linear", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs},
		{provider: "notion", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "openai", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "pagerduty", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "quickbooks", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs, endpointOverlay: "quickbooks-online-accounting-api-overlay"},
		{provider: "salesforce", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs, endpointOverlay: "salesforce-rest-core-overlay"},
		{provider: "sendgrid", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "servicenow", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs, endpointOverlay: "servicenow-rest-api-overlay"},
		{provider: "shopify", openAPI: SpecAvailabilityUnavailable, specKind: SpecKindHumanDocs, endpointOverlay: "shopify-admin-rest-overlay"},
		{provider: "slack", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "stripe", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "trello", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "twilio", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "xero", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "zendesk", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
		{provider: "zoho", openAPI: SpecAvailabilityKnown, specKind: SpecKindOpenAPI},
	}
	for _, test := range tests {
		row, ok := rows[test.provider]
		if !ok {
			t.Fatalf("missing M46 provider %s", test.provider)
		}
		if row.OpenAPIAvailability != test.openAPI {
			t.Fatalf("%s OpenAPI availability = %q, want %q", test.provider, row.OpenAPIAvailability, test.openAPI)
		}
		if !advisoryHasSpecKind(row, test.specKind) {
			t.Fatalf("%s missing spec kind %q", test.provider, test.specKind)
		}
		if row.AuthStatus == "" || row.AuthStatus == AuthStatusUnknown {
			t.Fatalf("%s auth status = %q, want reviewed status", test.provider, row.AuthStatus)
		}
		if test.openAPI == SpecAvailabilityUnavailable && row.UserOpenAPINeed != UserOpenAPINeedLikely {
			t.Fatalf("%s user OpenAPI need = %q, want likely", test.provider, row.UserOpenAPINeed)
		}
		if test.endpointOverlay != "" && !advisoryHasEndpointOverlay(row, test.endpointOverlay) {
			t.Fatalf("%s missing endpoint overlay %q", test.provider, test.endpointOverlay)
		}
		if test.provider == "linear" && len(row.EndpointOverlays) != 0 {
			t.Fatalf("linear endpoint overlays = %#v, want none", row.EndpointOverlays)
		}
	}
}

func TestDockerAPISourceCoverageAdvisoryRows(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	rows := map[string]ProviderAdvisory{}
	for _, row := range report.Providers {
		rows[row.ProviderID] = row
	}
	tests := []struct {
		provider        string
		wantNeed        UserOpenAPINeed
		wantAuth        AuthCompletenessStatus
		wantSpecRefID   string
		wantSourceValue string
	}{
		{
			provider:        "docker-engine",
			wantNeed:        UserOpenAPINeedNotExpected,
			wantAuth:        AuthStatusPresentIncomplete,
			wantSpecRefID:   "docker-engine-api-v1-54-openapi",
			wantSourceValue: "https://docs.docker.com/reference/api/engine/version/v1.54.yaml",
		},
		{
			provider:        "docker-hub",
			wantNeed:        UserOpenAPINeedNotExpected,
			wantAuth:        AuthStatusComplete,
			wantSpecRefID:   "docker-hub-api-openapi",
			wantSourceValue: "https://docs.docker.com/reference/api/hub/latest.yaml",
		},
		{
			provider:        "docker-registry",
			wantNeed:        UserOpenAPINeedPossible,
			wantAuth:        AuthStatusPresentIncomplete,
			wantSpecRefID:   "docker-registry-hub-supported-openapi",
			wantSourceValue: "https://docs.docker.com/reference/api/registry/latest.yaml",
		},
	}
	for _, test := range tests {
		row, ok := rows[test.provider]
		if !ok {
			t.Fatalf("missing Docker provider %s", test.provider)
		}
		if row.OpenAPIAvailability != SpecAvailabilityKnown {
			t.Fatalf("%s OpenAPI availability = %q, want %q", test.provider, row.OpenAPIAvailability, SpecAvailabilityKnown)
		}
		if row.UserOpenAPINeed != test.wantNeed {
			t.Fatalf("%s user OpenAPI need = %q, want %q", test.provider, row.UserOpenAPINeed, test.wantNeed)
		}
		if row.AuthStatus != test.wantAuth {
			t.Fatalf("%s auth status = %q, want %q", test.provider, row.AuthStatus, test.wantAuth)
		}
		if !advisoryHasSpecKind(row, SpecKindOpenAPI) {
			t.Fatalf("%s missing OpenAPI spec reference", test.provider)
		}
		if row.ResolvedOpenAPI.SpecRefID != test.wantSpecRefID || row.ResolvedOpenAPI.Value != test.wantSourceValue {
			t.Fatalf("%s resolved OpenAPI = %#v", test.provider, row.ResolvedOpenAPI)
		}
	}
}

func TestKubernetesClusterAPISourceCoverageAdvisoryRow(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "kubernetes"})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	if got, want := len(report.Providers), 1; got != want {
		t.Fatalf("providers len = %d, want %d", got, want)
	}
	row := report.Providers[0]
	if row.OpenAPIAvailability != SpecAvailabilityUnavailable {
		t.Fatalf("OpenAPI availability = %q, want %q", row.OpenAPIAvailability, SpecAvailabilityUnavailable)
	}
	if row.UserOpenAPINeed != UserOpenAPINeedLikely {
		t.Fatalf("user OpenAPI need = %q, want %q", row.UserOpenAPINeed, UserOpenAPINeedLikely)
	}
	if row.AuthStatus != AuthStatusPresentIncomplete {
		t.Fatalf("auth status = %q, want %q", row.AuthStatus, AuthStatusPresentIncomplete)
	}
	if !advisoryHasSpecKind(row, SpecKindHumanDocs) {
		t.Fatalf("missing Kubernetes human docs reference: %#v", row.SpecReferences)
	}
	if row.ResolvedOpenAPI.Source != ResolutionSourceBuiltInSpecReference || row.ResolvedOpenAPI.SpecRefID != "kubernetes-api-overview" {
		t.Fatalf("resolved OpenAPI = %#v", row.ResolvedOpenAPI)
	}
	if len(row.EndpointOverlays) != 0 {
		t.Fatalf("endpoint overlays = %#v, want none", row.EndpointOverlays)
	}
	assertHasFollowUp(t, row.ManualFollowUps, "OpenAPI-only workflows likely need a user-provided or generated OpenAPI document before import.")
}

func TestBuiltInProviderAdvisoryReportMissingCacheHasNoArtifactPath(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "slack"})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	row := report.Providers[0]
	if len(row.RegisteredArtifactPaths) != 0 {
		t.Fatalf("registered artifact paths = %#v, want none", row.RegisteredArtifactPaths)
	}
	if len(row.EndpointOverlays) != 0 {
		t.Fatalf("endpoint overlays = %#v, want none", row.EndpointOverlays)
	}
	for _, ref := range row.SpecReferences {
		if ref.RegisteredArtifactPath != "" {
			t.Fatalf("%s registered artifact path = %q, want empty", ref.ID, ref.RegisteredArtifactPath)
		}
	}
}

func TestBuiltInProviderAdvisoryReportUnknownProvider(t *testing.T) {
	if _, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "missing"}); err == nil {
		t.Fatal("BuiltInProviderAdvisoryReport() expected unknown provider error")
	}
}

func TestProviderAdvisoryReportReturnsCopies(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "slack"})
	if err != nil {
		t.Fatal(err)
	}
	report.Providers[0].Aliases[0] = "mutated"
	report.Providers[0].SpecReferences[0].RegisteredArtifactPath = "mutated"
	report.Providers[0].EndpointOverlays = append(report.Providers[0].EndpointOverlays, AdvisoryEndpointOverlay{
		ArtifactID: "mutated",
		Metadata:   map[string]string{"official_openapi": "false"},
	})
	report.Providers[0].EndpointOverlays[0].Metadata["official_openapi"] = "mutated"

	fresh, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "slack"})
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Providers[0].Aliases[0] == "mutated" {
		t.Fatal("advisory report leaked alias slice")
	}
	if fresh.Providers[0].SpecReferences[0].RegisteredArtifactPath == "mutated" {
		t.Fatal("advisory report leaked spec reference slice")
	}
	if len(fresh.Providers[0].EndpointOverlays) != 0 {
		t.Fatal("advisory report leaked endpoint overlay slice")
	}
}

func assertHasFollowUp(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if contains(value, want) {
			return
		}
	}
	t.Fatalf("manual follow-ups missing %q: %#v", want, values)
}

func assertNoFollowUp(t *testing.T, values []string, unwanted string) {
	t.Helper()
	for _, value := range values {
		if contains(value, unwanted) {
			t.Fatalf("manual follow-ups unexpectedly contain %q: %#v", unwanted, values)
		}
	}
}

func contains(value, substr string) bool {
	return strings.Contains(value, substr)
}

func m46EndpointOverlayArtifact(providerID, artifactID string) CatalogSpecArtifact {
	return CatalogSpecArtifact{
		ProviderID:  providerID,
		SpecRefID:   artifactID,
		ArtifactID:  artifactID,
		Kind:        "advisory-overlay",
		Path:        "advisory-overlays/" + artifactID + ".json",
		OverlayPath: "advisory-overlays/" + artifactID + ".json",
		BuilderPath: "overlay-builders/build_" + providerID + "_overlay.go",
		Metadata:    map[string]string{"derived_from_docs": "true", "official_openapi": "false"},
	}
}

func advisoryHasSpecKind(row ProviderAdvisory, kind SpecKind) bool {
	for _, ref := range row.SpecReferences {
		if ref.Kind == kind {
			return true
		}
	}
	return false
}

func advisoryHasEndpointOverlay(row ProviderAdvisory, artifactID string) bool {
	for _, overlay := range row.EndpointOverlays {
		if overlay.ArtifactID == artifactID {
			return true
		}
	}
	return false
}
