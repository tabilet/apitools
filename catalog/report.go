package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityClassification records source-backed auth/security completeness for
// providers whose upstream machine specs are already sufficient or still need
// review.
type SecurityClassification struct {
	ProviderID string                 `json:"provider_id"`
	SpecRefID  string                 `json:"spec_ref_id,omitempty"`
	Status     AuthCompletenessStatus `json:"status"`
	SourceRefs []string               `json:"source_refs,omitempty"`
	SourceNote string                 `json:"source_note"`
}

// SecurityReport is a deterministic metadata-only auth/security report for a
// provider catalog.
type SecurityReport struct {
	Providers []ProviderSecurityReport `json:"providers,omitempty"`
}

// ProviderSecurityReport records the effective catalog security status for one
// provider.
type ProviderSecurityReport struct {
	ProviderID  string                 `json:"provider_id"`
	Status      AuthCompletenessStatus `json:"status"`
	OverlayIDs  []string               `json:"overlay_ids,omitempty"`
	SpecRefIDs  []string               `json:"spec_ref_ids,omitempty"`
	SourceRefs  []string               `json:"source_refs,omitempty"`
	SourceNotes []string               `json:"source_notes,omitempty"`
}

// BuiltInSecurityClassifications returns source-backed security status records
// for built-in providers. Callers receive independent copies.
func BuiltInSecurityClassifications() []SecurityClassification {
	out := cloneSecurityClassifications(builtInSecurityClassifications)
	sortSecurityClassifications(out)
	return out
}

// BuiltInSecurityReport returns the built-in provider security report.
func BuiltInSecurityReport() (SecurityReport, error) {
	return BuildSecurityReport(BuiltInProviders(), BuiltInSecurityOverlays(), BuiltInSecurityClassifications())
}

// BuildSecurityReport combines provider metadata, security overlays, and
// source-backed classifications into a deterministic report.
func BuildSecurityReport(providers []Provider, overlays []SecurityOverlay, classifications []SecurityClassification) (SecurityReport, error) {
	if err := ValidateProviders(providers); err != nil {
		return SecurityReport{}, err
	}
	if err := ValidateSecurityOverlays(overlays, providers); err != nil {
		return SecurityReport{}, err
	}
	overlays = cloneSecurityOverlays(overlays)
	sortSecurityOverlays(overlays)
	providerByID := map[string]Provider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}

	classificationByProvider := map[string]SecurityClassification{}
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providerByID); err != nil {
			return SecurityReport{}, fmt.Errorf("security classification[%d]: %w", i, err)
		}
		if _, exists := classificationByProvider[classification.ProviderID]; exists {
			return SecurityReport{}, fmt.Errorf("security classification %q: duplicate provider", classification.ProviderID)
		}
		classificationByProvider[classification.ProviderID] = classification
	}

	overlaysByProvider := map[string][]SecurityOverlay{}
	for _, overlay := range overlays {
		overlaysByProvider[overlay.ProviderID] = append(overlaysByProvider[overlay.ProviderID], overlay)
	}

	var reports []ProviderSecurityReport
	for _, provider := range cloneProviders(providers) {
		report := ProviderSecurityReport{
			ProviderID: provider.ID,
			Status:     AuthStatusUnknown,
		}
		if classification, ok := classificationByProvider[provider.ID]; ok {
			report.Status = classification.Status
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, classification.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, classification.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, classification.SourceNote)
		}
		for _, overlay := range overlaysByProvider[provider.ID] {
			report.Status = overlay.Status
			report.OverlayIDs = appendIfNotEmpty(report.OverlayIDs, overlay.ID)
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, overlay.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, overlay.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, overlay.SourceNote)
		}
		report.OverlayIDs = sortedUniqueStrings(report.OverlayIDs)
		report.SpecRefIDs = sortedUniqueStrings(report.SpecRefIDs)
		report.SourceRefs = sortedUniqueStrings(report.SourceRefs)
		report.SourceNotes = sortedUniqueStrings(report.SourceNotes)
		reports = append(reports, report)
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].ProviderID < reports[j].ProviderID
	})
	return SecurityReport{Providers: reports}, nil
}

// FindProvider returns a provider security report by provider ID.
func (r SecurityReport) FindProvider(providerID string) (ProviderSecurityReport, bool) {
	normalized := normalizeKey(providerID)
	for _, provider := range r.Providers {
		if normalizeKey(provider.ProviderID) == normalized {
			return provider, true
		}
	}
	return ProviderSecurityReport{}, false
}

func validateSecurityClassification(classification SecurityClassification, providers map[string]Provider) error {
	if strings.TrimSpace(classification.ProviderID) == "" {
		return fmt.Errorf("missing provider id")
	}
	if !validID(classification.ProviderID) {
		return fmt.Errorf("invalid provider id %q", classification.ProviderID)
	}
	provider, ok := providers[classification.ProviderID]
	if !ok {
		return fmt.Errorf("unknown provider %q", classification.ProviderID)
	}
	if strings.TrimSpace(classification.SpecRefID) != "" {
		if !validID(classification.SpecRefID) {
			return fmt.Errorf("invalid spec ref id %q", classification.SpecRefID)
		}
		if !providerHasSpecRef(provider, classification.SpecRefID) {
			return fmt.Errorf("unknown spec ref %q for provider %q", classification.SpecRefID, classification.ProviderID)
		}
	}
	if !validAuthCompletenessStatus(classification.Status) {
		return fmt.Errorf("invalid status %q", classification.Status)
	}
	if strings.TrimSpace(classification.SourceNote) == "" {
		return fmt.Errorf("missing source note")
	}
	if len(classification.SourceRefs) == 0 {
		return fmt.Errorf("missing source refs")
	}
	for i, ref := range classification.SourceRefs {
		if !validHTTPSURL(ref) {
			return fmt.Errorf("source ref[%d]: must be https", i)
		}
	}
	return validateUniqueStrings("source_ref", classification.SourceRefs)
}

func sortSecurityClassifications(classifications []SecurityClassification) {
	sort.SliceStable(classifications, func(i, j int) bool {
		return classifications[i].ProviderID < classifications[j].ProviderID
	})
}

func cloneSecurityClassifications(in []SecurityClassification) []SecurityClassification {
	out := make([]SecurityClassification, len(in))
	for i, classification := range in {
		out[i] = classification
		out[i].SourceRefs = append([]string(nil), classification.SourceRefs...)
	}
	return out
}

func appendIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, value)
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
