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
