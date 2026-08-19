package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityOverlay records supplemental auth/security metadata separately from
// upstream provider specs. Overlays are advisory catalog metadata, not provider
// truth, and never contain credential values.
type SecurityOverlay struct {
	ID                string                   `json:"id"`
	ProviderID        string                   `json:"provider_id"`
	SpecRefID         string                   `json:"spec_ref_id,omitempty"`
	Status            AuthCompletenessStatus   `json:"status"`
	SecuritySchemes   []SecurityScheme         `json:"security_schemes,omitempty"`
	RootSecurity      []SecurityRequirement    `json:"root_security,omitempty"`
	RootSecuritySets  []SecurityRequirementSet `json:"root_security_sets,omitempty"`
	OperationSecurity []OperationSecurity      `json:"operation_security,omitempty"`
	SourceRefs        []string                 `json:"source_refs,omitempty"`
	SourceNote        string                   `json:"source_note"`
}

// BuiltInSecurityOverlays returns built-in security overlays in deterministic
// order. Callers receive independent copies.
func BuiltInSecurityOverlays() []SecurityOverlay {
	out := cloneSecurityOverlays(builtInSecurityOverlays)
	sortSecurityOverlays(out)
	return out
}

// SecurityOverlaysForProvider returns built-in security overlays for a provider
// in deterministic order.
func SecurityOverlaysForProvider(providerID string) []SecurityOverlay {
	provider, ok := FindBuiltInProvider(providerID)
	if !ok {
		return nil
	}
	indexes := generatedSecurityOverlayIndexesByProvider[provider.ID]
	out := make([]SecurityOverlay, 0, len(indexes))
	for _, index := range indexes {
		out = append(out, cloneSecurityOverlay(builtInSecurityOverlays[index]))
	}
	return out
}

// ValidateSecurityOverlays validates security overlays against provider catalog
// metadata.
func ValidateSecurityOverlays(overlays []SecurityOverlay, providers []Provider) error {
	providerByID := map[string]Provider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	seenIDs := map[string]struct{}{}
	for i, overlay := range overlays {
		if err := validateSecurityOverlay(overlay, providerByID); err != nil {
			return fmt.Errorf("security overlay[%d]: %w", i, err)
		}
		if _, exists := seenIDs[overlay.ID]; exists {
			return fmt.Errorf("security overlay %q: duplicate id", overlay.ID)
		}
		seenIDs[overlay.ID] = struct{}{}
	}
	return nil
}

func validateSecurityOverlay(overlay SecurityOverlay, providers map[string]Provider) error {
	if strings.TrimSpace(overlay.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(overlay.ID) {
		return fmt.Errorf("invalid id %q", overlay.ID)
	}
	if strings.TrimSpace(overlay.ProviderID) == "" {
		return fmt.Errorf("overlay %q: missing provider id", overlay.ID)
	}
	if !validID(overlay.ProviderID) {
		return fmt.Errorf("overlay %q: invalid provider id %q", overlay.ID, overlay.ProviderID)
	}
	provider, ok := providers[overlay.ProviderID]
	if !ok {
		return fmt.Errorf("overlay %q: unknown provider %q", overlay.ID, overlay.ProviderID)
	}
	if strings.TrimSpace(overlay.SpecRefID) != "" {
		if !validID(overlay.SpecRefID) {
			return fmt.Errorf("overlay %q: invalid spec ref id %q", overlay.ID, overlay.SpecRefID)
		}
		if !providerHasSpecRef(provider, overlay.SpecRefID) {
			return fmt.Errorf("overlay %q: unknown spec ref %q for provider %q", overlay.ID, overlay.SpecRefID, overlay.ProviderID)
		}
	}
	if !validAuthCompletenessStatus(overlay.Status) {
		return fmt.Errorf("overlay %q: invalid status %q", overlay.ID, overlay.Status)
	}
	if strings.TrimSpace(overlay.SourceNote) == "" {
		return fmt.Errorf("overlay %q: missing source note", overlay.ID)
	}
	if len(overlay.SourceRefs) == 0 {
		return fmt.Errorf("overlay %q: missing source refs", overlay.ID)
	}
	for i, ref := range overlay.SourceRefs {
		if !validHTTPSURL(ref) {
			return fmt.Errorf("overlay %q source ref[%d]: must be https", overlay.ID, i)
		}
	}
	if err := validateUniqueStrings("source_ref", overlay.SourceRefs); err != nil {
		return fmt.Errorf("overlay %q: %w", overlay.ID, err)
	}

	schemes := map[string]struct{}{}
	for i, scheme := range overlay.SecuritySchemes {
		if err := validateSecurityScheme(scheme); err != nil {
			return fmt.Errorf("overlay %q scheme[%d]: %w", overlay.ID, i, err)
		}
		if _, exists := schemes[scheme.Name]; exists {
			return fmt.Errorf("overlay %q scheme[%d]: duplicate security scheme %q", overlay.ID, i, scheme.Name)
		}
		schemes[scheme.Name] = struct{}{}
	}
	for i, requirement := range overlay.RootSecurity {
		if err := validateSecurityRequirement(requirement, schemes); err != nil {
			return fmt.Errorf("overlay %q root security[%d]: %w", overlay.ID, i, err)
		}
	}
	for i, set := range overlay.RootSecuritySets {
		if err := validateSecurityRequirementSet(set, schemes); err != nil {
			return fmt.Errorf("overlay %q root security sets[%d]: %w", overlay.ID, i, err)
		}
	}
	for i, operation := range overlay.OperationSecurity {
		if err := validateOperationSecurity(operation, schemes); err != nil {
			return fmt.Errorf("overlay %q operation security[%d]: %w", overlay.ID, i, err)
		}
	}
	return nil
}

func providerHasSpecRef(provider Provider, id string) bool {
	for _, ref := range provider.SpecReferences {
		if ref.ID == id {
			return true
		}
	}
	return false
}

func sortSecurityOverlays(overlays []SecurityOverlay) {
	sort.SliceStable(overlays, func(i, j int) bool {
		if overlays[i].ProviderID == overlays[j].ProviderID {
			return overlays[i].ID < overlays[j].ID
		}
		return overlays[i].ProviderID < overlays[j].ProviderID
	})
}

func cloneSecurityOverlays(in []SecurityOverlay) []SecurityOverlay {
	out := make([]SecurityOverlay, len(in))
	for i, overlay := range in {
		out[i] = cloneSecurityOverlay(overlay)
	}
	return out
}

func cloneSecurityOverlay(overlay SecurityOverlay) SecurityOverlay {
	overlay.SecuritySchemes = cloneSecuritySchemes(overlay.SecuritySchemes)
	overlay.RootSecurity = cloneSecurityRequirements(overlay.RootSecurity)
	overlay.RootSecuritySets = cloneSecurityRequirementSets(overlay.RootSecuritySets)
	overlay.OperationSecurity = cloneOperationSecurity(overlay.OperationSecurity)
	overlay.SourceRefs = append([]string(nil), overlay.SourceRefs...)
	return overlay
}
