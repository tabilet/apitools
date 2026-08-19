package catalog

import (
	"reflect"
	"testing"
	"time"
)

func TestBuiltInCatalogQualityReportHasNoCurrentFindings(t *testing.T) {
	report := BuiltInCatalogQualityReport(CatalogQualityOptions{AsOf: mustQualityDate(t, "2026-05-18")})
	if report.ErrorCount() != 0 || report.WarningCount() != 0 {
		t.Fatalf("BuiltInCatalogQualityReport() findings = %#v, want none", report.Findings)
	}
}

func TestCatalogQualityReportFindsSourceAndSecurityIssues(t *testing.T) {
	provider := qualityProvider("demo")
	provider.SourceHints = nil
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{Providers: []Provider{provider}},
		AsOf:    mustQualityDate(t, "2026-05-18"),
	})
	if !hasQualityFinding(report, CatalogQualityWarning, "missing-provider-source-hints", "demo") {
		t.Fatalf("missing source hint warning in %#v", report.Findings)
	}
	if !hasQualityFinding(report, CatalogQualityError, "missing-security-status", "demo") {
		t.Fatalf("missing security status error in %#v", report.Findings)
	}
}

func TestCatalogQualityReportFindsStaleVerificationDates(t *testing.T) {
	provider := qualityProvider("demo")
	provider.SpecReferences[0].VerifiedAt = "2024-01-01"
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{Providers: []Provider{provider}},
		SecurityClassifications: []SecurityClassification{{
			ProviderID: "demo",
			SpecRefID:  "demo-openapi",
			Status:     AuthStatusComplete,
			SourceRefs: []string{"https://example.com/openapi.json"},
			SourceNote: "Demo OpenAPI includes security metadata.",
		}},
		AsOf:                  mustQualityDate(t, "2026-05-18"),
		StaleVerificationDays: 365,
	})
	if !hasQualityFinding(report, CatalogQualityWarning, "stale-verification-date", "demo") {
		t.Fatalf("missing stale verification warning in %#v", report.Findings)
	}
	if report.ErrorCount() != 0 {
		t.Fatalf("errors = %#v, want none", report.Findings)
	}
}

func TestCatalogQualityReportFindsOverlayReferenceMismatch(t *testing.T) {
	provider := qualityProvider("demo")
	overlay := qualityOverlay("demo")
	overlay.SpecRefID = "missing-spec"
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{
			Providers:        []Provider{provider},
			SecurityOverlays: []SecurityOverlay{overlay},
		},
		AsOf: mustQualityDate(t, "2026-05-18"),
	})
	if !hasQualityFinding(report, CatalogQualityError, "invalid-security-overlays", "") {
		t.Fatalf("missing overlay validation error in %#v", report.Findings)
	}
}

func TestCatalogQualityReportFindsConflictingScopedDispositions(t *testing.T) {
	provider := qualityProvider("demo")
	first := qualityOverlay("demo")
	first.Status = AuthStatusComplete
	second := qualityOverlay("demo")
	second.ID = "demo-second-auth-overlay"
	second.Status = AuthStatusAbsent
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{
			Providers:        []Provider{provider},
			SecurityOverlays: []SecurityOverlay{first, second},
		},
		AsOf: mustQualityDate(t, "2026-05-18"),
	})
	if !hasQualityFinding(report, CatalogQualityError, "conflicting-security-status", "demo") {
		t.Fatalf("missing conflicting security status in %#v", report.Findings)
	}
}

func TestCatalogQualityReportFindsUnresolvedOverlayOperations(t *testing.T) {
	provider := qualityProvider("demo")
	overlay := qualityOverlay("demo")
	overlay.OperationSecurity = []OperationSecurity{{
		Match:    OperationMatch{OperationID: "missingOperation"},
		Security: []SecurityRequirement{{Scheme: "demoBearer"}},
	}}
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{
			Providers:        []Provider{provider},
			SecurityOverlays: []SecurityOverlay{overlay},
		},
		AsOf: mustQualityDate(t, "2026-05-18"),
		KnownOperationsByProvider: map[string][]OperationMatch{
			"demo": {{OperationID: "knownOperation"}},
		},
	})
	if !hasQualityFinding(report, CatalogQualityError, "unresolved-overlay-operation", "demo") {
		t.Fatalf("missing unresolved operation error in %#v", report.Findings)
	}
}

func TestCatalogQualityReportOrderIsDeterministic(t *testing.T) {
	alpha := qualityProvider("alpha")
	zeta := qualityProvider("zeta")
	alpha.SpecReferences[0].VerifiedAt = "2024-01-01"
	zeta.SpecReferences[0].VerifiedAt = "2024-01-01"
	report := BuildCatalogQualityReport(CatalogQualityOptions{
		Catalog: Catalog{Providers: []Provider{zeta, alpha}},
		SecurityClassifications: []SecurityClassification{
			qualityClassification("zeta"),
			qualityClassification("alpha"),
		},
		AsOf:                  mustQualityDate(t, "2026-05-18"),
		StaleVerificationDays: 365,
	})
	var got []string
	for _, finding := range report.Findings {
		got = append(got, finding.ProviderID+":"+finding.Code)
	}
	want := []string{"alpha:stale-verification-date", "zeta:stale-verification-date"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("finding order = %#v, want %#v", got, want)
	}
}

func qualityProvider(id string) Provider {
	return Provider{
		ID:                              id,
		DisplayName:                     id,
		SourceHints:                     []string{"test"},
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     id,
		OfficialOpenAPIAvailability:     SpecAvailabilityKnown,
		OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
		UserOpenAPINeed:                 UserOpenAPINeedNotExpected,
		SpecReferences: []SpecReference{{
			ID:              id + "-openapi",
			Kind:            SpecKindOpenAPI,
			URL:             "https://example.com/openapi.json",
			SourceAuthority: SourceAuthorityOfficialProvider,
			VerifiedAt:      "2026-05-18",
			SourceNote:      "Official test OpenAPI document.",
		}},
	}
}

func qualityOverlay(providerID string) SecurityOverlay {
	return SecurityOverlay{
		ID:         providerID + "-auth-overlay",
		ProviderID: providerID,
		SpecRefID:  providerID + "-openapi",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:   "demoBearer",
			Type:   SecuritySchemeHTTP,
			Scheme: "bearer",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "demoBearer"}},
		SourceRefs:   []string{"https://example.com/auth"},
		SourceNote:   "Official docs describe bearer auth.",
	}
}

func qualityClassification(providerID string) SecurityClassification {
	return SecurityClassification{
		ProviderID: providerID,
		SpecRefID:  providerID + "-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://example.com/openapi.json"},
		SourceNote: "OpenAPI includes security metadata.",
	}
}

func hasQualityFinding(report CatalogQualityReport, severity CatalogQualitySeverity, code, providerID string) bool {
	for _, finding := range report.Findings {
		if finding.Severity == severity && finding.Code == code && finding.ProviderID == providerID {
			return true
		}
	}
	return false
}

func mustQualityDate(t *testing.T, value string) time.Time {
	t.Helper()
	out, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
