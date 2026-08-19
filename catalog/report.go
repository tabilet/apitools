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

// SecurityDispositionScope identifies whether security evidence applies to a
// whole provider or one catalog spec reference.
type SecurityDispositionScope string

const (
	SecurityDispositionProvider SecurityDispositionScope = "provider"
	SecurityDispositionSpec     SecurityDispositionScope = "spec"
)

// SecurityDisposition records the explicit effective status and provenance
// for one provider or provider/spec scope. A classification is baseline
// evidence; a scoped overlay is an explicit reviewed supplementation and owns
// the effective OverlayStatus. Different overlay statuses in the same scope
// produce conflict rather than relying on input order.
type SecurityDisposition struct {
	Scope                SecurityDispositionScope `json:"scope"`
	SpecRefID            string                   `json:"spec_ref_id,omitempty"`
	Status               AuthCompletenessStatus   `json:"status"`
	ClassificationStatus AuthCompletenessStatus   `json:"classification_status,omitempty"`
	OverlayStatus        AuthCompletenessStatus   `json:"overlay_status,omitempty"`
	ConflictStatuses     []AuthCompletenessStatus `json:"conflict_statuses,omitempty"`
	OverlayIDs           []string                 `json:"overlay_ids,omitempty"`
	SourceRefs           []string                 `json:"source_refs,omitempty"`
	SourceNotes          []string                 `json:"source_notes,omitempty"`
}

// ProviderSecurityReport records the effective catalog security status for one
// provider.
type ProviderSecurityReport struct {
	ProviderID   string                 `json:"provider_id"`
	Status       AuthCompletenessStatus `json:"status"`
	Dispositions []SecurityDisposition  `json:"dispositions,omitempty"`
	OverlayIDs   []string               `json:"overlay_ids,omitempty"`
	SpecRefIDs   []string               `json:"spec_ref_ids,omitempty"`
	SourceRefs   []string               `json:"source_refs,omitempty"`
	SourceNotes  []string               `json:"source_notes,omitempty"`
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
	return BuiltInCatalogIndex().SecurityReport(), nil
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
	if err := ValidateSecurityClassifications(classifications, providers); err != nil {
		return SecurityReport{}, err
	}
	overlays = cloneSecurityOverlays(overlays)
	sortSecurityOverlays(overlays)
	classifications = cloneSecurityClassifications(classifications)
	sortSecurityClassifications(classifications)

	classificationByScope := make(map[string]SecurityClassification, len(classifications))
	overlaysByScope := map[string][]SecurityOverlay{}
	for _, classification := range classifications {
		classificationByScope[securityScopeKey(classification.ProviderID, classification.SpecRefID)] = classification
	}
	for _, overlay := range overlays {
		key := securityScopeKey(overlay.ProviderID, overlay.SpecRefID)
		overlaysByScope[key] = append(overlaysByScope[key], overlay)
	}

	var reports []ProviderSecurityReport
	for _, provider := range cloneProviders(providers) {
		report := ProviderSecurityReport{
			ProviderID: provider.ID,
			Status:     AuthStatusUnknown,
		}
		for _, specRefID := range securityScopesForProvider(provider.ID, classificationByScope, overlaysByScope) {
			disposition := buildSecurityDisposition(
				specRefID,
				classificationByScope[securityScopeKey(provider.ID, specRefID)],
				overlaysByScope[securityScopeKey(provider.ID, specRefID)],
			)
			report.Dispositions = append(report.Dispositions, disposition)
			report.OverlayIDs = append(report.OverlayIDs, disposition.OverlayIDs...)
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, disposition.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, disposition.SourceRefs...)
			report.SourceNotes = append(report.SourceNotes, disposition.SourceNotes...)
		}
		report.Status = aggregateSecurityDispositionStatus(report.Dispositions)
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

func securityScopeKey(providerID, specRefID string) string {
	return strings.TrimSpace(providerID) + "\x00" + strings.TrimSpace(specRefID)
}

func securityScopesForProvider(providerID string, classifications map[string]SecurityClassification, overlays map[string][]SecurityOverlay) []string {
	prefix := providerID + "\x00"
	seen := map[string]struct{}{}
	for key := range classifications {
		if strings.HasPrefix(key, prefix) {
			seen[strings.TrimPrefix(key, prefix)] = struct{}{}
		}
	}
	for key := range overlays {
		if strings.HasPrefix(key, prefix) {
			seen[strings.TrimPrefix(key, prefix)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for specRefID := range seen {
		out = append(out, specRefID)
	}
	sort.Strings(out)
	return out
}

func buildSecurityDisposition(specRefID string, classification SecurityClassification, overlays []SecurityOverlay) SecurityDisposition {
	disposition := SecurityDisposition{
		Scope:     SecurityDispositionSpec,
		SpecRefID: strings.TrimSpace(specRefID),
		Status:    AuthStatusUnknown,
	}
	if disposition.SpecRefID == "" {
		disposition.Scope = SecurityDispositionProvider
	}
	if classification.ProviderID != "" {
		disposition.ClassificationStatus = classification.Status
		disposition.Status = classification.Status
		disposition.SourceRefs = append(disposition.SourceRefs, classification.SourceRefs...)
		disposition.SourceNotes = appendIfNotEmpty(disposition.SourceNotes, classification.SourceNote)
	}
	overlayStatusSet := map[AuthCompletenessStatus]struct{}{}
	for _, overlay := range overlays {
		overlayStatusSet[overlay.Status] = struct{}{}
		disposition.OverlayIDs = append(disposition.OverlayIDs, overlay.ID)
		disposition.SourceRefs = append(disposition.SourceRefs, overlay.SourceRefs...)
		disposition.SourceNotes = appendIfNotEmpty(disposition.SourceNotes, overlay.SourceNote)
	}
	overlayStatuses := sortedAuthStatuses(overlayStatusSet)
	switch len(overlayStatuses) {
	case 1:
		disposition.OverlayStatus = overlayStatuses[0]
		disposition.Status = overlayStatuses[0]
	case 2:
		disposition.Status = AuthStatusConflict
		disposition.ConflictStatuses = overlayStatuses
	default:
		if len(overlayStatuses) > 2 {
			disposition.Status = AuthStatusConflict
			disposition.ConflictStatuses = overlayStatuses
		}
	}
	disposition.OverlayIDs = sortedUniqueStrings(disposition.OverlayIDs)
	disposition.SourceRefs = sortedUniqueStrings(disposition.SourceRefs)
	disposition.SourceNotes = sortedUniqueStrings(disposition.SourceNotes)
	return disposition
}

func sortedAuthStatuses(statuses map[AuthCompletenessStatus]struct{}) []AuthCompletenessStatus {
	out := make([]AuthCompletenessStatus, 0, len(statuses))
	for status := range statuses {
		out = append(out, status)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func aggregateSecurityDispositionStatus(dispositions []SecurityDisposition) AuthCompletenessStatus {
	if len(dispositions) == 0 {
		return AuthStatusUnknown
	}
	statuses := map[AuthCompletenessStatus]struct{}{}
	for _, disposition := range dispositions {
		if disposition.Status == AuthStatusConflict {
			return AuthStatusConflict
		}
		statuses[disposition.Status] = struct{}{}
	}
	if len(statuses) > 1 {
		return AuthStatusMixed
	}
	for status := range statuses {
		return status
	}
	return AuthStatusUnknown
}

// FindProvider returns a provider security report by provider ID.
func (r SecurityReport) FindProvider(providerID string) (ProviderSecurityReport, bool) {
	normalized := normalizeKey(providerID)
	for _, provider := range r.Providers {
		if normalizeKey(provider.ProviderID) == normalized {
			return cloneProviderSecurityReport(provider), true
		}
	}
	return ProviderSecurityReport{}, false
}

// ResolveDisposition finds exact spec-scoped evidence, then provider-wide
// evidence. It does not fall back to another spec's disposition.
func (r ProviderSecurityReport) ResolveDisposition(specRefID string) (SecurityDisposition, bool) {
	wanted := strings.TrimSpace(specRefID)
	for _, disposition := range r.Dispositions {
		if disposition.Scope == SecurityDispositionSpec && disposition.SpecRefID == wanted {
			return cloneSecurityDisposition(disposition), true
		}
	}
	for _, disposition := range r.Dispositions {
		if disposition.Scope == SecurityDispositionProvider {
			return cloneSecurityDisposition(disposition), true
		}
	}
	return SecurityDisposition{}, false
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
	if !validAuthEvidenceStatus(classification.Status) {
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
		if classifications[i].ProviderID == classifications[j].ProviderID {
			return classifications[i].SpecRefID < classifications[j].SpecRefID
		}
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
