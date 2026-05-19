package apitools

import (
	"encoding/json"
	"os"
	"path/filepath"
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

type endpointOverlayDecisions struct {
	built     map[string]string
	noOverlay map[string]struct{}
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
