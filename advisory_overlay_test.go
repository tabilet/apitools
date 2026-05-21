package apitools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestTrackedAdvisoryOverlayArtifactsLoad(t *testing.T) {
	paths, err := filepath.Glob("catalog-openapi-cache/advisory-overlays/*-overlay.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no advisory overlay artifacts found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			index, err := LoadOperationIndex(path)
			if err != nil {
				t.Fatalf("LoadOperationIndex(%q) error = %v", path, err)
			}
			if len(index.OperationIDs) == 0 {
				t.Fatalf("%s has no indexed operations", path)
			}
		})
	}
}

func TestTrackedAdvisoryOverlaySecurityMatchesCatalogRequirements(t *testing.T) {
	paths, err := filepath.Glob("catalog-openapi-cache/advisory-overlays/*-overlay.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no advisory overlay artifacts found")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Components struct {
					SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
				} `json:"components"`
				Security []map[string][]string `json:"security"`
				Paths    map[string]map[string]struct {
					OperationID string                `json:"operationId"`
					Security    []map[string][]string `json:"security"`
				} `json:"paths"`
				Overlay struct {
					ProviderID string `json:"provider_id"`
				} `json:"x-apitools-overlay"`
			}
			if err := json.Unmarshal(content, &doc); err != nil {
				t.Fatalf("%s JSON parse error: %v", path, err)
			}
			if doc.Overlay.ProviderID == "" {
				t.Fatalf("%s missing x-apitools-overlay.provider_id", path)
			}

			assertSecurityRequirementsUseDeclaredSchemes(t, "root security", doc.Security, doc.Components.SecuritySchemes)
			expectedSets := catalogRootSecuritySetsForProvider(doc.Overlay.ProviderID)
			for route, pathItem := range doc.Paths {
				for method, operation := range pathItem {
					if operation.OperationID == "" {
						continue
					}
					context := method + " " + route
					assertSecurityRequirementsUseDeclaredSchemes(t, context, operation.Security, doc.Components.SecuritySchemes)
					if len(expectedSets) > 0 {
						assertOperationSecurityIncludesCatalogRequirementSet(t, context, operation.Security, expectedSets)
					}
				}
			}
		})
	}
}

func TestHumanDocsEndpointOverlayDecisionsHaveTrackedArtifacts(t *testing.T) {
	decisions := loadHumanDocsEndpointOverlayDecisions(t)
	registrations := loadAdvisoryOverlayRegistrations(t)

	providerIDByDecisionKey := map[string]string{}
	for _, provider := range catalog.BuiltInProviders() {
		providerIDByDecisionKey[overlayDecisionKey(provider.DisplayName)] = provider.ID
	}

	builtByProviderID := map[string]struct{}{}
	noOverlayByProviderID := map[string]struct{}{}
	for providerName, overlayFile := range decisions.built {
		providerID, ok := providerIDByDecisionKey[overlayDecisionKey(providerName)]
		if !ok {
			t.Fatalf("overlay decision for unknown provider %q", providerName)
		}
		builtByProviderID[providerID] = struct{}{}

		overlayPath := filepath.Join("catalog-openapi-cache", "advisory-overlays", overlayFile)
		content, err := os.ReadFile(overlayPath)
		if err != nil {
			t.Fatalf("%s decision references missing overlay file %s: %v", providerName, overlayPath, err)
		}

		var doc struct {
			Overlay struct {
				ProviderID      string `json:"provider_id"`
				OfficialOpenAPI bool   `json:"official_openapi"`
				DerivedFromDocs bool   `json:"derived_from_docs"`
			} `json:"x-apitools-overlay"`
		}
		if err := json.Unmarshal(content, &doc); err != nil {
			t.Fatalf("%s overlay JSON parse error: %v", overlayPath, err)
		}
		if doc.Overlay.ProviderID != providerID {
			t.Fatalf("%s provider_id = %q, want %q", overlayPath, doc.Overlay.ProviderID, providerID)
		}
		if doc.Overlay.OfficialOpenAPI {
			t.Fatalf("%s marks official_openapi true", overlayPath)
		}
		if !doc.Overlay.DerivedFromDocs {
			t.Fatalf("%s missing derived_from_docs true", overlayPath)
		}

		index, err := LoadOperationIndex(overlayPath)
		if err != nil {
			t.Fatalf("LoadOperationIndex(%q) error = %v", overlayPath, err)
		}
		if len(index.OperationIDs) == 0 {
			t.Fatalf("%s has no indexed operations", overlayPath)
		}

		registration, ok := registrations["advisory-overlays/"+overlayFile]
		if !ok {
			t.Fatalf("%s missing advisory overlay artifact registry row", overlayFile)
		}
		if registration.providerID != providerID {
			t.Fatalf("%s registry provider = %q, want %q", overlayFile, registration.providerID, providerID)
		}
		if registration.builderPath == "" {
			t.Fatalf("%s registry row missing builderPath", overlayFile)
		}
		if _, err := os.Stat(filepath.Join("catalog-openapi-cache", registration.builderPath)); err != nil {
			t.Fatalf("%s builder path %q missing: %v", overlayFile, registration.builderPath, err)
		}
	}

	for providerName := range decisions.noOverlay {
		providerID, ok := providerIDByDecisionKey[overlayDecisionKey(providerName)]
		if !ok {
			t.Fatalf("no-overlay decision for unknown provider %q", providerName)
		}
		noOverlayByProviderID[providerID] = struct{}{}
	}

	var missing []string
	for _, provider := range catalog.BuiltInProviders() {
		if provider.OfficialOpenAPIAvailability != catalog.SpecAvailabilityUnavailable ||
			provider.OfficialMachineSpecAvailability != catalog.SpecAvailabilityUnknown ||
			!providerHasHumanDocsReference(provider) {
			continue
		}
		if _, ok := builtByProviderID[provider.ID]; ok {
			continue
		}
		if _, ok := noOverlayByProviderID[provider.ID]; ok {
			continue
		}
		missing = append(missing, provider.ID)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("human-docs-primary providers missing endpoint overlay or no-overlay decision: %v", missing)
	}
}

func TestClockifyAdvisoryOverlayAuthAlternatives(t *testing.T) {
	content, err := os.ReadFile("catalog-openapi-cache/advisory-overlays/clockify-api-v1-overlay.json")
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Security []map[string][]string `json:"security"`
		Paths    map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Security    []map[string][]string `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("Clockify overlay JSON parse error: %v", err)
	}

	assertClockifyAuthAlternatives(t, "root security", doc.Security)
	operationCount := 0
	for path, pathItem := range doc.Paths {
		for method, operation := range pathItem {
			if operation.OperationID == "" {
				continue
			}
			operationCount++
			assertClockifyAuthAlternatives(t, method+" "+path, operation.Security)
		}
	}
	if operationCount == 0 {
		t.Fatal("Clockify overlay has no operations")
	}
}

func TestCopperAdvisoryOverlayCombinesRequiredHeaders(t *testing.T) {
	content, err := os.ReadFile("catalog-openapi-cache/advisory-overlays/copper-developer-api-overlay.json")
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string                `json:"operationId"`
			Security    []map[string][]string `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("Copper overlay JSON parse error: %v", err)
	}

	operationCount := 0
	for path, pathItem := range doc.Paths {
		for method, operation := range pathItem {
			if operation.OperationID == "" {
				continue
			}
			operationCount++
			assertCopperRequiredHeaders(t, method+" "+path, operation.Security)
		}
	}
	if operationCount == 0 {
		t.Fatal("Copper overlay has no operations")
	}
}

func TestM57ReviewFindingRegressions(t *testing.T) {
	t.Run("apollo search uses query parameters", func(t *testing.T) {
		doc := loadOverlayDoc(t, "catalog-openapi-cache/advisory-overlays/apollo-api-overlay.json")
		for path, wantParam := range map[string]string{
			"/mixed_people/api_search": "q_keywords",
			"/mixed_companies/search":  "q_organization_name",
		} {
			operation := doc.Paths[path]["post"]
			if operation.RequestBody != nil {
				t.Fatalf("%s has requestBody, want documented query parameters", path)
			}
			if !hasQueryParameter(operation.Parameters, wantParam, false) {
				t.Fatalf("%s parameters = %#v, want optional query parameter %q", path, operation.Parameters, wantParam)
			}
		}
	})

	t.Run("marketo required parameters are modeled", func(t *testing.T) {
		doc := loadOverlayDoc(t, "catalog-openapi-cache/advisory-overlays/marketo-rest-api-overlay.json")
		leads := doc.Paths["/rest/v1/leads.json"]["get"]
		for _, name := range []string{"filterType", "filterValues"} {
			if !hasQueryParameter(leads.Parameters, name, true) {
				t.Fatalf("Marketo leads parameters = %#v, want required query parameter %q", leads.Parameters, name)
			}
		}
		activities := doc.Paths["/rest/v1/activities.json"]["get"]
		for _, name := range []string{"activityTypeIds", "nextPageToken"} {
			if !hasQueryParameter(activities.Parameters, name, true) {
				t.Fatalf("Marketo activities parameters = %#v, want required query parameter %q", activities.Parameters, name)
			}
		}
	})

	t.Run("acrobat sign preserves agreement read scope", func(t *testing.T) {
		doc := loadOverlayDoc(t, "catalog-openapi-cache/advisory-overlays/adobe-acrobat-sign-api-overlay.json")
		scheme := doc.Components.SecuritySchemes["adobeAcrobatSignOAuth2"]
		if len(scheme) == 0 || !strings.Contains(string(scheme), "agreement_read") {
			t.Fatalf("Acrobat Sign security scheme = %s, want agreement_read scope", scheme)
		}
		for _, path := range []string{"/agreements", "/agreements/{agreementId}"} {
			operation := doc.Paths[path]["get"]
			if !hasSecurityScope(operation.Security, "adobeAcrobatSignOAuth2", "agreement_read") {
				t.Fatalf("%s security = %#v, want adobeAcrobatSignOAuth2 agreement_read", path, operation.Security)
			}
		}

		overlays := catalog.SecurityOverlaysForProvider("adobe-acrobat-sign")
		if len(overlays) != 1 {
			t.Fatalf("Acrobat Sign overlays = %#v, want one", overlays)
		}
		if !catalogOverlayHasOperationScope(overlays[0], "GET", "/agreements", "adobeAcrobatSignOAuth2", "agreement_read") {
			t.Fatalf("Acrobat Sign catalog overlay missing GET /agreements agreement_read scope: %#v", overlays[0].OperationSecurity)
		}
	})
}

type endpointOverlayDecisions struct {
	built     map[string]string
	noOverlay map[string]struct{}
}

type overlayDoc struct {
	Components struct {
		SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
	} `json:"components"`
	Paths map[string]map[string]struct {
		OperationID string                `json:"operationId"`
		Parameters  []overlayParameter    `json:"parameters"`
		RequestBody json.RawMessage       `json:"requestBody"`
		Security    []map[string][]string `json:"security"`
	} `json:"paths"`
}

type overlayParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

func loadOverlayDoc(t *testing.T, path string) overlayDoc {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc overlayDoc
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatalf("%s JSON parse error: %v", path, err)
	}
	return doc
}

func hasQueryParameter(parameters []overlayParameter, name string, required bool) bool {
	for _, parameter := range parameters {
		if parameter.Name == name && parameter.In == "query" && parameter.Required == required {
			return true
		}
	}
	return false
}

func hasSecurityScope(requirements []map[string][]string, scheme, scope string) bool {
	for _, requirement := range requirements {
		scopes, ok := requirement[scheme]
		if !ok {
			continue
		}
		for _, got := range scopes {
			if got == scope {
				return true
			}
		}
	}
	return false
}

func catalogOverlayHasOperationScope(overlay catalog.SecurityOverlay, method, path, scheme, scope string) bool {
	for _, operation := range overlay.OperationSecurity {
		if operation.Match.Method != method || operation.Match.Path != path {
			continue
		}
		for _, set := range operation.SecuritySets {
			for _, requirement := range set.Requirements {
				if requirement.Scheme != scheme {
					continue
				}
				for _, got := range requirement.Scopes {
					if got == scope {
						return true
					}
				}
			}
		}
	}
	return false
}

func loadHumanDocsEndpointOverlayDecisions(t *testing.T) endpointOverlayDecisions {
	t.Helper()
	content, err := os.ReadFile("catalog-openapi-cache/advisory-overlays/human-doc-overlay-decisions.md")
	if err != nil {
		t.Fatal(err)
	}
	decisions := endpointOverlayDecisions{
		built:     map[string]string{},
		noOverlay: map[string]struct{}{},
	}
	section := ""
	for _, line := range strings.Split(string(content), "\n") {
		switch strings.TrimSpace(line) {
		case "## Built":
			section = "built"
			continue
		case "## No Endpoint Overlay":
			section = "no-overlay"
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		providerName := strings.TrimSpace(parts[1])
		if providerName == "" || providerName == "Provider" || strings.HasPrefix(providerName, "---") {
			continue
		}
		switch section {
		case "built":
			overlayFile := strings.Trim(strings.TrimSpace(parts[2]), "`")
			if overlayFile == "" {
				t.Fatalf("built decision for %s missing overlay filename", providerName)
			}
			decisions.built[providerName] = overlayFile
		case "no-overlay":
			decisions.noOverlay[providerName] = struct{}{}
		}
	}
	return decisions
}

type advisoryOverlayRegistration struct {
	providerID   string
	builderPath  string
	artifactPath string
}

func loadAdvisoryOverlayRegistrations(t *testing.T) map[string]advisoryOverlayRegistration {
	t.Helper()
	content, err := os.ReadFile("catalog-openapi-cache/artifact-registry/register_catalog_artifacts.go")
	if err != nil {
		t.Fatal(err)
	}
	entryRE := regexp.MustCompile(`(?s)\{\s*providerID:\s*"([^"]+)",\s*artifactID:\s*"([^"]+)",\s*path:\s*"([^"]+)",\s*builderPath:\s*"([^"]+)",`)
	out := map[string]advisoryOverlayRegistration{}
	for _, match := range entryRE.FindAllStringSubmatch(string(content), -1) {
		out[match[3]] = advisoryOverlayRegistration{
			providerID:   match[1],
			artifactPath: match[3],
			builderPath:  match[4],
		}
	}
	return out
}

func providerHasHumanDocsReference(provider catalog.Provider) bool {
	for _, ref := range provider.SpecReferences {
		if ref.Kind == catalog.SpecKindHumanDocs {
			return true
		}
	}
	return false
}

func overlayDecisionKey(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func assertClockifyAuthAlternatives(t *testing.T, context string, requirements []map[string][]string) {
	t.Helper()
	if len(requirements) != 2 {
		t.Fatalf("%s requirement count = %d, want 2", context, len(requirements))
	}
	seenAPIKey := false
	seenAddonToken := false
	for _, req := range requirements {
		if _, hasAPIKey := req["clockifyAPIKey"]; hasAPIKey {
			seenAPIKey = true
			if _, hasAddonToken := req["clockifyAddonToken"]; hasAddonToken {
				t.Fatalf("%s combines clockifyAPIKey and clockifyAddonToken in one requirement: %#v", context, req)
			}
		}
		if _, hasAddonToken := req["clockifyAddonToken"]; hasAddonToken {
			seenAddonToken = true
		}
		if len(req) != 1 {
			t.Fatalf("%s requirement object = %#v, want exactly one Clockify auth scheme", context, req)
		}
	}
	if !seenAPIKey || !seenAddonToken {
		t.Fatalf("%s requirements = %#v, want clockifyAPIKey and clockifyAddonToken alternatives", context, requirements)
	}
}

func assertCopperRequiredHeaders(t *testing.T, context string, requirements []map[string][]string) {
	t.Helper()
	if len(requirements) != 1 {
		t.Fatalf("%s requirement count = %d, want 1 combined requirement", context, len(requirements))
	}
	requirement := requirements[0]
	want := []string{"copperAccessToken", "copperApplication", "copperUserEmail"}
	if len(requirement) != len(want) {
		t.Fatalf("%s requirement = %#v, want exactly Copper's three required header schemes", context, requirement)
	}
	for _, scheme := range want {
		scopes, ok := requirement[scheme]
		if !ok {
			t.Fatalf("%s requirement = %#v, missing %s", context, requirement, scheme)
		}
		if len(scopes) != 0 {
			t.Fatalf("%s %s scopes = %#v, want none", context, scheme, scopes)
		}
	}
}

func assertSecurityRequirementsUseDeclaredSchemes(t *testing.T, context string, requirements []map[string][]string, declared map[string]json.RawMessage) {
	t.Helper()
	for _, requirement := range requirements {
		if len(requirement) == 0 {
			t.Fatalf("%s has empty security requirement", context)
		}
		for scheme := range requirement {
			if _, ok := declared[scheme]; !ok {
				t.Fatalf("%s references undeclared security scheme %q", context, scheme)
			}
		}
	}
}

func catalogRootSecuritySetsForProvider(providerID string) [][]string {
	var out [][]string
	for _, overlay := range catalog.SecurityOverlaysForProvider(providerID) {
		for _, set := range overlay.RootSecuritySets {
			var schemes []string
			for _, requirement := range set.Requirements {
				schemes = append(schemes, requirement.Scheme)
			}
			sort.Strings(schemes)
			out = append(out, schemes)
		}
	}
	return out
}

func assertOperationSecurityIncludesCatalogRequirementSet(t *testing.T, context string, requirements []map[string][]string, expectedSets [][]string) {
	t.Helper()
	for _, requirement := range requirements {
		var got []string
		for scheme := range requirement {
			got = append(got, scheme)
		}
		sort.Strings(got)
		for _, want := range expectedSets {
			if reflect.DeepEqual(got, want) {
				return
			}
		}
	}
	t.Fatalf("%s security = %#v, want one requirement matching catalog root security sets %#v", context, requirements, expectedSets)
}
