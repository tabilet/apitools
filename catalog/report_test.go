package catalog

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildSecurityReportAggregatesPerSpecAsMixed(t *testing.T) {
	provider := securityReportProviderWithTwoSpecs()
	classifications := []SecurityClassification{
		securityReportClassification("demo", "demo-openapi", AuthStatusComplete),
		securityReportClassification("demo", "demo-events-openapi", AuthStatusAbsent),
	}
	report, err := BuildSecurityReport([]Provider{provider}, nil, classifications)
	if err != nil {
		t.Fatal(err)
	}
	row, ok := report.FindProvider("demo")
	if !ok {
		t.Fatal("missing demo security report")
	}
	if row.Status != AuthStatusMixed {
		t.Fatalf("provider status = %q, want %q", row.Status, AuthStatusMixed)
	}
	if got, want := len(row.Dispositions), 2; got != want {
		t.Fatalf("dispositions = %d, want %d", got, want)
	}
	disposition, ok := row.ResolveDisposition("demo-openapi")
	if !ok || disposition.Status != AuthStatusComplete || disposition.Scope != SecurityDispositionSpec {
		t.Fatalf("openapi disposition = %#v, %v", disposition, ok)
	}
	disposition, ok = row.ResolveDisposition("demo-events-openapi")
	if !ok || disposition.Status != AuthStatusAbsent {
		t.Fatalf("events disposition = %#v, %v", disposition, ok)
	}
}

func TestBuildSecurityReportUsesScopedOverlayAsEffectiveDisposition(t *testing.T) {
	provider := qualityProvider("demo")
	classification := qualityClassification("demo")
	overlay := qualityOverlay("demo")
	overlay.Status = AuthStatusPresentIncomplete

	report, err := BuildSecurityReport([]Provider{provider}, []SecurityOverlay{overlay}, []SecurityClassification{classification})
	if err != nil {
		t.Fatal(err)
	}
	row, _ := report.FindProvider("demo")
	disposition, ok := row.ResolveDisposition("demo-openapi")
	if !ok {
		t.Fatal("missing scoped disposition")
	}
	if disposition.Status != AuthStatusPresentIncomplete || disposition.ClassificationStatus != AuthStatusComplete || disposition.OverlayStatus != AuthStatusPresentIncomplete {
		t.Fatalf("effective disposition = %#v", disposition)
	}
}

func TestBuildSecurityReportReportsSameScopeOverlayConflict(t *testing.T) {
	provider := qualityProvider("demo")
	first := qualityOverlay("demo")
	first.Status = AuthStatusComplete
	second := qualityOverlay("demo")
	second.ID = "demo-second-auth-overlay"
	second.Status = AuthStatusAbsent

	report, err := BuildSecurityReport([]Provider{provider}, []SecurityOverlay{second, first}, nil)
	if err != nil {
		t.Fatal(err)
	}
	row, _ := report.FindProvider("demo")
	if row.Status != AuthStatusConflict {
		t.Fatalf("provider status = %q, want %q", row.Status, AuthStatusConflict)
	}
	disposition, ok := row.ResolveDisposition("demo-openapi")
	if !ok || disposition.Status != AuthStatusConflict {
		t.Fatalf("conflict disposition = %#v, %v", disposition, ok)
	}
	if got, want := len(disposition.ConflictStatuses), 2; got != want {
		t.Fatalf("conflict statuses = %#v, want %d", disposition.ConflictStatuses, want)
	}
}

func TestBuildSecurityReportIsIndependentOfEvidenceOrder(t *testing.T) {
	provider := securityReportProviderWithTwoSpecs()
	classifications := []SecurityClassification{
		securityReportClassification("demo", "demo-openapi", AuthStatusComplete),
		securityReportClassification("demo", "demo-events-openapi", AuthStatusAbsent),
	}
	overlays := []SecurityOverlay{qualityOverlay("demo")}

	first, err := BuildSecurityReport([]Provider{provider}, overlays, classifications)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildSecurityReport([]Provider{provider}, []SecurityOverlay{overlays[0]}, []SecurityClassification{classifications[1], classifications[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("security reports depend on evidence order:\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestResolveDispositionFallsBackToProviderScope(t *testing.T) {
	provider := securityReportProviderWithTwoSpecs()
	classification := securityReportClassification("demo", "", AuthStatusComplete)
	report, err := BuildSecurityReport([]Provider{provider}, nil, []SecurityClassification{classification})
	if err != nil {
		t.Fatal(err)
	}
	row, _ := report.FindProvider("demo")
	disposition, ok := row.ResolveDisposition("demo-events-openapi")
	if !ok || disposition.Scope != SecurityDispositionProvider || disposition.Status != AuthStatusComplete {
		t.Fatalf("provider fallback disposition = %#v, %v", disposition, ok)
	}
}

func TestValidateCatalogBundleRejectsConflictingSecurityDispositions(t *testing.T) {
	bundle := reviewedCatalogBundleForTest(t)
	conflict := cloneSecurityOverlay(bundle.SecurityOverlays[0])
	conflict.ID += "-conflict"
	conflict.Status = AuthStatusComplete
	bundle.SecurityOverlays = append(bundle.SecurityOverlays, conflict)
	sortSecurityOverlays(bundle.SecurityOverlays)

	err := ValidateCatalogBundle(bundle)
	if err == nil || !strings.Contains(err.Error(), "conflicting security dispositions") {
		t.Fatalf("conflicting-disposition error = %v", err)
	}
}

func TestDerivedSecurityStatusesAreRejectedAsEvidence(t *testing.T) {
	provider := qualityProvider("demo")
	classification := qualityClassification("demo")
	classification.Status = AuthStatusMixed
	if err := ValidateSecurityClassifications([]SecurityClassification{classification}, []Provider{provider}); err == nil {
		t.Fatal("mixed classification status accepted")
	}
	overlay := qualityOverlay("demo")
	overlay.Status = AuthStatusConflict
	if err := ValidateSecurityOverlays([]SecurityOverlay{overlay}, []Provider{provider}); err == nil {
		t.Fatal("conflict overlay status accepted")
	}
}

func TestResolveProviderUsesSelectedSpecDisposition(t *testing.T) {
	provider := securityReportProviderWithTwoSpecs()
	classifications := []SecurityClassification{
		securityReportClassification("demo", "demo-openapi", AuthStatusComplete),
		securityReportClassification("demo", "demo-events-openapi", AuthStatusAbsent),
	}
	resolved, err := ResolveProvider(ResolveProviderOptions{
		Catalog:                 Catalog{Providers: []Provider{provider}},
		SecurityClassifications: classifications,
		ProviderKey:             "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.SecurityReport.Status != AuthStatusMixed {
		t.Fatalf("provider aggregate status = %q, want %q", resolved.SecurityReport.Status, AuthStatusMixed)
	}
	if resolved.OpenAPI.SpecRefID != "demo-openapi" {
		t.Fatalf("selected spec = %q", resolved.OpenAPI.SpecRefID)
	}
	if resolved.SecurityStatus != AuthStatusComplete {
		t.Fatalf("selected security status = %q, want %q", resolved.SecurityStatus, AuthStatusComplete)
	}
	if resolved.Security.Source != ResolutionSourceSecurityClassification || resolved.Security.SpecRefID != "demo-openapi" {
		t.Fatalf("selected security source = %#v", resolved.Security)
	}
}

func securityReportProviderWithTwoSpecs() Provider {
	provider := qualityProvider("demo")
	second := provider.SpecReferences[0]
	second.ID = "demo-events-openapi"
	second.URL = "https://example.com/events-openapi.json"
	provider.SpecReferences = append(provider.SpecReferences, second)
	return provider
}

func securityReportClassification(providerID, specRefID string, status AuthCompletenessStatus) SecurityClassification {
	return SecurityClassification{
		ProviderID: providerID,
		SpecRefID:  specRefID,
		Status:     status,
		SourceRefs: []string{"https://example.com/security"},
		SourceNote: "Reviewed test security evidence.",
	}
}
