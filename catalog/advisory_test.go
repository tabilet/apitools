package catalog

import "testing"

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
	if report.Providers[0].ProviderID != "airtable" {
		t.Fatalf("first provider = %q, want airtable", report.Providers[0].ProviderID)
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

func TestBuiltInProviderAdvisoryReportMissingCacheHasNoArtifactPath(t *testing.T) {
	report, err := BuiltInProviderAdvisoryReport(ProviderAdvisoryOptions{ProviderKey: "slack"})
	if err != nil {
		t.Fatalf("BuiltInProviderAdvisoryReport() error = %v", err)
	}
	row := report.Providers[0]
	if len(row.RegisteredArtifactPaths) != 0 {
		t.Fatalf("registered artifact paths = %#v, want none", row.RegisteredArtifactPaths)
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
}
