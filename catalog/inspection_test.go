package catalog

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBuiltInSecurityInspectionViewPreservesProvenance(t *testing.T) {
	view, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatalf("BuiltInSecurityInspectionView() error = %v", err)
	}
	if view.ProviderID != "github" {
		t.Fatalf("ProviderID = %q, want github", view.ProviderID)
	}
	if view.Status != AuthStatusOverlayRequired {
		t.Fatalf("Status = %q, want %q", view.Status, AuthStatusOverlayRequired)
	}
	if view.Classification == nil {
		t.Fatalf("Classification is nil")
	}
	if view.Classification.Provenance != SecurityProvenanceClassification {
		t.Fatalf("classification provenance = %q, want %q", view.Classification.Provenance, SecurityProvenanceClassification)
	}

	schemeByName := map[string]EffectiveSecurityScheme{}
	for _, scheme := range view.SecuritySchemes {
		schemeByName[scheme.Scheme.Name] = scheme
	}
	githubBearer, ok := schemeByName["githubBearer"]
	if !ok {
		t.Fatalf("missing githubBearer scheme in %#v", view.SecuritySchemes)
	}
	if githubBearer.Provenance != SecurityProvenanceOverlay || githubBearer.OverlayID != "github-rest-api-auth-overlay" {
		t.Fatalf("githubBearer provenance = %q overlay = %q", githubBearer.Provenance, githubBearer.OverlayID)
	}

	if !hasInspectionConflict(view.Conflicts, SecurityInspectionConflictOverlayOnlyAddition, "githubBearer") {
		t.Fatalf("missing overlay-only conflict for githubBearer in %#v", view.Conflicts)
	}
	if len(view.RootSecurity) == 0 || view.RootSecurity[0].Provenance != SecurityProvenanceOverlay {
		t.Fatalf("root security provenance not preserved: %#v", view.RootSecurity)
	}
}

func TestBuiltInSecurityInspectionViewPreservesCombinedRequirementSets(t *testing.T) {
	view, err := BuiltInSecurityInspectionView("copper")
	if err != nil {
		t.Fatalf("BuiltInSecurityInspectionView() error = %v", err)
	}
	if len(view.RootSecuritySets) != 1 {
		t.Fatalf("RootSecuritySets len = %d, want 1: %#v", len(view.RootSecuritySets), view.RootSecuritySets)
	}
	set := view.RootSecuritySets[0]
	if set.Provenance != SecurityProvenanceOverlay || set.OverlayID != "copper-api-auth-overlay" {
		t.Fatalf("RootSecuritySets provenance = %q overlay = %q", set.Provenance, set.OverlayID)
	}
	got := make([]string, 0, len(set.Requirements))
	for _, requirement := range set.Requirements {
		got = append(got, requirement.Requirement.Scheme)
	}
	want := []string{"copperAccessToken", "copperApplication", "copperUserEmail"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Copper combined requirement set = %#v, want %#v", got, want)
	}
}

func TestSecurityInspectionReportsConflicts(t *testing.T) {
	provider := inspectionTestProvider()
	catalog := Catalog{
		Providers: []Provider{provider},
		SecurityOverlays: []SecurityOverlay{
			{
				ID:              "example-first-overlay",
				ProviderID:      provider.ID,
				SpecRefID:       provider.SpecReferences[0].ID,
				Status:          AuthStatusOverlayRequired,
				SecuritySchemes: []SecurityScheme{bearerScheme("sharedAuth")},
				RootSecurity:    []SecurityRequirement{{Scheme: "sharedAuth"}},
				SourceRefs:      []string{"https://example.com/auth-one"},
				SourceNote:      "First overlay source note.",
			},
			{
				ID:              "example-second-overlay",
				ProviderID:      provider.ID,
				SpecRefID:       provider.SpecReferences[0].ID,
				Status:          AuthStatusOverlayRequired,
				SecuritySchemes: []SecurityScheme{bearerScheme("sharedAuth")},
				RootSecurity:    []SecurityRequirement{{Scheme: "missingAuth"}},
				OperationSecurity: []OperationSecurity{
					{
						Match:    OperationMatch{Method: "GET", Path: "/missing"},
						Security: []SecurityRequirement{{Scheme: "missingAuth"}},
					},
				},
				SourceRefs: []string{"https://example.com/auth-two"},
				SourceNote: "Second overlay source note.",
			},
		},
	}
	view, err := BuildSecurityInspectionView(SecurityInspectionOptions{
		Catalog:         catalog,
		ProviderKey:     provider.ID,
		KnownOperations: []OperationMatch{{Method: "GET", Path: "/known"}},
	})
	if err != nil {
		t.Fatalf("BuildSecurityInspectionView() error = %v", err)
	}
	if !hasInspectionConflict(view.Conflicts, SecurityInspectionConflictDuplicateScheme, "sharedAuth") {
		t.Fatalf("missing duplicate scheme conflict in %#v", view.Conflicts)
	}
	if !hasInspectionConflict(view.Conflicts, SecurityInspectionConflictMissingScheme, "missingAuth") {
		t.Fatalf("missing missing scheme conflict in %#v", view.Conflicts)
	}
	if !hasInspectionConflict(view.Conflicts, SecurityInspectionConflictUnresolvedOperation, "") {
		t.Fatalf("missing unresolved operation conflict in %#v", view.Conflicts)
	}
	if len(view.OperationSecurity) != 1 || view.OperationSecurity[0].Provenance != SecurityProvenanceUnresolved {
		t.Fatalf("unresolved operation provenance = %#v, want unresolved", view.OperationSecurity)
	}
	if len(view.SecuritySchemes) != 2 {
		t.Fatalf("SecuritySchemes len = %d, want 2", len(view.SecuritySchemes))
	}

	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("Marshal inspection view: %v", err)
	}
	if !strings.Contains(string(raw), `"match":{"method":"GET","path":"/missing"}`) {
		t.Fatalf("unresolved operation conflict JSON missing match:\n%s", string(raw))
	}
}

func TestSecurityInspectionSupportsPerSpecClassifications(t *testing.T) {
	provider := securityReportProviderWithTwoSpecs()
	view, err := BuildSecurityInspectionView(SecurityInspectionOptions{
		Catalog:     Catalog{Providers: []Provider{provider}},
		ProviderKey: "demo",
		SecurityClassifications: []SecurityClassification{
			securityReportClassification("demo", "demo-openapi", AuthStatusComplete),
			securityReportClassification("demo", "demo-events-openapi", AuthStatusAbsent),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != AuthStatusMixed || len(view.Classifications) != 2 {
		t.Fatalf("inspection classifications/status = %#v/%q", view.Classifications, view.Status)
	}
	if view.Classification == nil || view.Classification.SpecRefID != "demo-events-openapi" {
		t.Fatalf("legacy classification = %#v", view.Classification)
	}
}

func TestSecurityInspectionResolvesKnownOperationMatches(t *testing.T) {
	provider := inspectionTestProvider()
	catalog := Catalog{
		Providers: []Provider{provider},
		SecurityOverlays: []SecurityOverlay{
			{
				ID:              "example-operation-overlay",
				ProviderID:      provider.ID,
				SpecRefID:       provider.SpecReferences[0].ID,
				Status:          AuthStatusOverlayRequired,
				SecuritySchemes: []SecurityScheme{bearerScheme("operationAuth")},
				OperationSecurity: []OperationSecurity{
					{
						Match:    OperationMatch{Method: "GET", Path: "/known"},
						Security: []SecurityRequirement{{Scheme: "operationAuth"}},
					},
				},
				SourceRefs: []string{"https://example.com/auth"},
				SourceNote: "Operation overlay source note.",
			},
		},
	}
	view, err := BuildSecurityInspectionView(SecurityInspectionOptions{
		Catalog:         catalog,
		ProviderKey:     provider.ID,
		KnownOperations: []OperationMatch{{Method: "GET", Path: "/known"}},
	})
	if err != nil {
		t.Fatalf("BuildSecurityInspectionView() error = %v", err)
	}
	if hasInspectionConflict(view.Conflicts, SecurityInspectionConflictUnresolvedOperation, "") {
		t.Fatalf("unexpected unresolved operation conflict in %#v", view.Conflicts)
	}
	if len(view.OperationSecurity) != 1 || view.OperationSecurity[0].Security[0].Provenance != SecurityProvenanceOverlay {
		t.Fatalf("operation security provenance not preserved: %#v", view.OperationSecurity)
	}
	if view.OperationSecurity[0].Provenance != SecurityProvenanceOverlay {
		t.Fatalf("resolved operation provenance = %q, want %q", view.OperationSecurity[0].Provenance, SecurityProvenanceOverlay)
	}
}

func TestSecurityInspectionViewReturnsCopies(t *testing.T) {
	view, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatal(err)
	}
	view.SecuritySchemes[0].Scheme.Name = "mutated"
	view.SecuritySchemes[0].SourceRefs[0] = "https://example.com/mutated"
	view.RootSecurity[0].Requirement.Scheme = "mutated"
	view.Classification.SourceRefs[0] = "https://example.com/mutated"

	fresh, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.SecuritySchemes[0].Scheme.Name == "mutated" {
		t.Fatalf("SecuritySchemes leaked nested scheme copy")
	}
	if fresh.SecuritySchemes[0].SourceRefs[0] == "https://example.com/mutated" {
		t.Fatalf("SecuritySchemes leaked source refs copy")
	}
	if fresh.RootSecurity[0].Requirement.Scheme == "mutated" {
		t.Fatalf("RootSecurity leaked requirement copy")
	}
	copper, err := BuiltInSecurityInspectionView("copper")
	if err != nil {
		t.Fatal(err)
	}
	copper.RootSecuritySets[0].Requirements[0].Requirement.Scheme = "mutated"
	freshCopper, err := BuiltInSecurityInspectionView("copper")
	if err != nil {
		t.Fatal(err)
	}
	if freshCopper.RootSecuritySets[0].Requirements[0].Requirement.Scheme == "mutated" {
		t.Fatalf("RootSecuritySets leaked requirement copy")
	}
	if fresh.Classification.SourceRefs[0] == "https://example.com/mutated" {
		t.Fatalf("Classification leaked source refs copy")
	}
}

func TestSecurityInspectionViewIsDeterministic(t *testing.T) {
	first, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("inspection view not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestSecurityInspectionConflictOmitZeroMatchJSON(t *testing.T) {
	view, err := BuiltInSecurityInspectionView("github")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("Marshal inspection view: %v", err)
	}
	if strings.Contains(string(raw), `"match":{}`) {
		t.Fatalf("zero conflict match should be omitted:\n%s", string(raw))
	}
	if !strings.Contains(string(raw), `"type":"overlay-only-addition"`) {
		t.Fatalf("expected overlay-only conflict in JSON:\n%s", string(raw))
	}
}

func hasInspectionConflict(conflicts []SecurityInspectionConflict, conflictType SecurityInspectionConflictType, scheme string) bool {
	for _, conflict := range conflicts {
		if conflict.Type != conflictType {
			continue
		}
		if scheme != "" && conflict.Scheme != scheme {
			continue
		}
		return true
	}
	return false
}

func inspectionTestProvider() Provider {
	return Provider{
		ID:                              "example",
		DisplayName:                     "Example",
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     "example",
		OfficialOpenAPIAvailability:     SpecAvailabilityKnown,
		OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
		UserOpenAPINeed:                 UserOpenAPINeedPossible,
		SpecReferences: []SpecReference{
			{
				ID:              "example-openapi",
				Kind:            SpecKindOpenAPI,
				URL:             "https://example.com/openapi.yaml",
				SourceAuthority: SourceAuthorityOfficialProvider,
				SourceNote:      "Example official OpenAPI source.",
			},
		},
	}
}
