package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// ResolutionSource identifies where resolved catalog metadata came from.
type ResolutionSource string

const (
	ResolutionSourceNone                   ResolutionSource = "none"
	ResolutionSourceUserOpenAPI            ResolutionSource = "user-openapi"
	ResolutionSourceUserSecurityOverlay    ResolutionSource = "user-security-overlay"
	ResolutionSourceProjectLocalOpenAPI    ResolutionSource = "project-local-openapi"
	ResolutionSourceBuiltInSpecReference   ResolutionSource = "built-in-spec-reference"
	ResolutionSourceBuiltInSecurityOverlay ResolutionSource = "built-in-security-overlay"
	ResolutionSourceSecurityClassification ResolutionSource = "security-classification"
)

// ResolveProviderOptions controls metadata-only provider resolution.
type ResolveProviderOptions struct {
	Catalog                 Catalog
	SecurityClassifications []SecurityClassification
	ProviderKey             string
	UserOpenAPI             string
	UserSecurityOverlay     string
	ProjectLocalOpenAPI     string
}

// ResolvedReference records the selected metadata source for a resolved item.
type ResolvedReference struct {
	Source     ResolutionSource `json:"source"`
	Value      string           `json:"value,omitempty"`
	SpecRefID  string           `json:"spec_ref_id,omitempty"`
	OverlayID  string           `json:"overlay_id,omitempty"`
	SourceNote string           `json:"source_note,omitempty"`
}

// ResolvedProvider is a metadata-only provider resolution result.
type ResolvedProvider struct {
	Provider                Provider               `json:"provider"`
	OpenAPI                 ResolvedReference      `json:"openapi"`
	Security                ResolvedReference      `json:"security"`
	SecurityStatus          AuthCompletenessStatus `json:"security_status"`
	SecurityReport          ProviderSecurityReport `json:"security_report,omitempty"`
	CatalogSpecReferences   []SpecReference        `json:"catalog_spec_references,omitempty"`
	CatalogSecurityOverlays []SecurityOverlay      `json:"catalog_security_overlays,omitempty"`
}

// ResolveProvider resolves provider spec and auth/security metadata with the
// precedence documented in status-M5: explicit user OpenAPI, explicit user
// security overlay, project-local OpenAPI, built-in spec reference, then
// built-in security overlay. It does not read files, fetch URLs, parse specs,
// execute operations, or resolve credentials.
func ResolveProvider(options ResolveProviderOptions) (ResolvedProvider, error) {
	cat := options.Catalog
	classifications := options.SecurityClassifications
	var index *CatalogIndex
	if cat.isZero() {
		cat = BuiltInCatalog()
		if classifications == nil {
			index = BuiltInCatalogIndex()
		}
	}
	if index == nil {
		var err error
		index, err = NewCatalogIndex(cat, classifications)
		if err != nil {
			return ResolvedProvider{}, err
		}
	}
	return resolveProviderWithIndex(index, options)
}

func resolveProviderWithIndex(index *CatalogIndex, options ResolveProviderOptions) (ResolvedProvider, error) {
	provider, ok := index.FindProvider(options.ProviderKey)
	if !ok {
		return ResolvedProvider{}, fmt.Errorf("unknown provider %q", options.ProviderKey)
	}
	if err := validateResolutionRef("user OpenAPI", options.UserOpenAPI); err != nil {
		return ResolvedProvider{}, err
	}
	if err := validateResolutionRef("user security overlay", options.UserSecurityOverlay); err != nil {
		return ResolvedProvider{}, err
	}
	if err := validateResolutionRef("project-local OpenAPI", options.ProjectLocalOpenAPI); err != nil {
		return ResolvedProvider{}, err
	}

	overlays := index.SecurityOverlaysForProvider(provider.ID)
	securityReport, _ := index.SecurityForProvider(provider.ID)
	resolved := ResolvedProvider{
		Provider:                cloneProvider(provider),
		OpenAPI:                 ResolvedReference{Source: ResolutionSourceNone},
		Security:                ResolvedReference{Source: ResolutionSourceNone},
		SecurityStatus:          AuthStatusUnknown,
		SecurityReport:          cloneProviderSecurityReport(securityReport),
		CatalogSpecReferences:   append([]SpecReference(nil), provider.SpecReferences...),
		CatalogSecurityOverlays: cloneSecurityOverlays(overlays),
	}

	if ref := strings.TrimSpace(options.UserOpenAPI); ref != "" {
		resolved.OpenAPI = ResolvedReference{Source: ResolutionSourceUserOpenAPI, Value: ref}
		resolved.SecurityStatus = AuthStatusUnknown
		if overlay := strings.TrimSpace(options.UserSecurityOverlay); overlay != "" {
			resolved.Security = ResolvedReference{Source: ResolutionSourceUserSecurityOverlay, Value: overlay}
		}
		return resolved, nil
	}
	if overlay := strings.TrimSpace(options.UserSecurityOverlay); overlay != "" {
		resolved.Security = ResolvedReference{Source: ResolutionSourceUserSecurityOverlay, Value: overlay}
		resolved.SecurityStatus = AuthStatusUnknown
	}
	if ref := strings.TrimSpace(options.ProjectLocalOpenAPI); ref != "" {
		resolved.OpenAPI = ResolvedReference{Source: ResolutionSourceProjectLocalOpenAPI, Value: ref}
		if resolved.Security.Source == ResolutionSourceNone {
			resolved.SecurityStatus = AuthStatusUnknown
		}
		return resolved, nil
	}

	if spec, ok := preferredSpecReference(provider); ok {
		resolved.OpenAPI = ResolvedReference{
			Source:     ResolutionSourceBuiltInSpecReference,
			Value:      spec.URL,
			SpecRefID:  spec.ID,
			SourceNote: spec.SourceNote,
		}
	}
	if resolved.Security.Source == ResolutionSourceNone {
		selectedSecurity := ProviderSecurityReport{
			ProviderID: securityReport.ProviderID,
			Status:     AuthStatusUnknown,
		}
		if disposition, ok := securityReport.ResolveDisposition(resolved.OpenAPI.SpecRefID); ok {
			selectedSecurity = providerReportForDisposition(securityReport.ProviderID, disposition)
		} else if resolved.OpenAPI.SpecRefID == "" {
			selectedSecurity = securityReport
		}
		resolved.SecurityStatus = selectedSecurity.Status
		if selectedSecurity.Status == AuthStatusMixed || selectedSecurity.Status == AuthStatusConflict {
			return resolved, nil
		}
		if len(selectedSecurity.OverlayIDs) > 0 {
			overlayID := selectedSecurity.OverlayIDs[0]
			resolved.Security = ResolvedReference{
				Source:     ResolutionSourceBuiltInSecurityOverlay,
				OverlayID:  overlayID,
				SourceNote: overlaySourceNote(overlays, overlayID),
			}
		} else if selectedSecurity.Status != "" && selectedSecurity.Status != AuthStatusUnknown && selectedSecurity.Status != AuthStatusMixed && selectedSecurity.Status != AuthStatusConflict {
			resolved.Security = ResolvedReference{
				Source:     ResolutionSourceSecurityClassification,
				SpecRefID:  firstString(selectedSecurity.SpecRefIDs),
				SourceNote: firstString(selectedSecurity.SourceNotes),
			}
		}
	}
	return resolved, nil
}

func (c Catalog) isZero() bool {
	return len(c.Providers) == 0 && len(c.SecurityOverlays) == 0
}

func validateResolutionRef(label, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, "://") && !validHTTPSURL(trimmed) && !strings.HasPrefix(trimmed, "http://") {
		return fmt.Errorf("%s reference must be a path or http(s) URL", label)
	}
	return nil
}

func preferredSpecReference(provider Provider) (SpecReference, bool) {
	for _, kind := range []SpecKind{
		SpecKindOpenAPI,
		SpecKindOpenAPIIndex,
		SpecKindDropboxStone,
		SpecKindGoogleDiscovery,
		SpecKindSmithyJSON,
		SpecKindHumanDocs,
	} {
		for _, ref := range provider.SpecReferences {
			if ref.Kind == kind {
				return ref, true
			}
		}
	}
	return SpecReference{}, false
}

func cloneProviderSecurityReport(report ProviderSecurityReport) ProviderSecurityReport {
	dispositions := report.Dispositions
	report.Dispositions = make([]SecurityDisposition, len(dispositions))
	for i, disposition := range dispositions {
		report.Dispositions[i] = cloneSecurityDisposition(disposition)
	}
	report.OverlayIDs = append([]string(nil), report.OverlayIDs...)
	report.SpecRefIDs = append([]string(nil), report.SpecRefIDs...)
	report.SourceRefs = append([]string(nil), report.SourceRefs...)
	report.SourceNotes = append([]string(nil), report.SourceNotes...)
	return report
}

func cloneSecurityDisposition(disposition SecurityDisposition) SecurityDisposition {
	disposition.ConflictStatuses = append([]AuthCompletenessStatus(nil), disposition.ConflictStatuses...)
	disposition.OverlayIDs = append([]string(nil), disposition.OverlayIDs...)
	disposition.SourceRefs = append([]string(nil), disposition.SourceRefs...)
	disposition.SourceNotes = append([]string(nil), disposition.SourceNotes...)
	return disposition
}

func providerReportForDisposition(providerID string, disposition SecurityDisposition) ProviderSecurityReport {
	report := ProviderSecurityReport{
		ProviderID:   providerID,
		Status:       disposition.Status,
		Dispositions: []SecurityDisposition{cloneSecurityDisposition(disposition)},
		OverlayIDs:   append([]string(nil), disposition.OverlayIDs...),
		SourceRefs:   append([]string(nil), disposition.SourceRefs...),
		SourceNotes:  append([]string(nil), disposition.SourceNotes...),
	}
	report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, disposition.SpecRefID)
	return report
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func overlaySourceNote(overlays []SecurityOverlay, overlayID string) string {
	for _, overlay := range overlays {
		if overlay.ID == overlayID {
			return overlay.SourceNote
		}
	}
	return ""
}

// SecurityReportRows returns provider security reports in deterministic order.
func SecurityReportRows(report SecurityReport) []ProviderSecurityReport {
	out := make([]ProviderSecurityReport, len(report.Providers))
	for i, provider := range report.Providers {
		out[i] = cloneProviderSecurityReport(provider)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ProviderID < out[j].ProviderID
	})
	return out
}
