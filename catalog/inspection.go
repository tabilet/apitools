package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityProvenance labels where inspection-view security metadata came from.
type SecurityProvenance string

const (
	SecurityProvenanceUpstream       SecurityProvenance = "upstream"
	SecurityProvenanceOverlay        SecurityProvenance = "overlay"
	SecurityProvenanceClassification SecurityProvenance = "classification"
	SecurityProvenanceUnresolved     SecurityProvenance = "unresolved"
)

// SecurityInspectionConflictType identifies advisory issues found while
// building a read-only overlay inspection view.
type SecurityInspectionConflictType string

const (
	SecurityInspectionConflictDuplicateScheme     SecurityInspectionConflictType = "duplicate-scheme-name"
	SecurityInspectionConflictMissingScheme       SecurityInspectionConflictType = "missing-referenced-scheme"
	SecurityInspectionConflictOverlayOnlyAddition SecurityInspectionConflictType = "overlay-only-addition"
	SecurityInspectionConflictUnresolvedOperation SecurityInspectionConflictType = "unresolved-operation-match"
)

// SecurityInspectionOptions controls metadata-only overlay inspection.
type SecurityInspectionOptions struct {
	Catalog                 Catalog
	SecurityClassifications []SecurityClassification
	ProviderKey             string
	KnownOperations         []OperationMatch
}

// SecurityInspectionView is a read-only advisory view of catalog security
// metadata and overlay effects. It does not mutate or export OpenAPI documents.
type SecurityInspectionView struct {
	ProviderID        string                            `json:"provider_id"`
	DisplayName       string                            `json:"display_name,omitempty"`
	Status            AuthCompletenessStatus            `json:"status"`
	Classification    *SecurityInspectionClassification `json:"classification,omitempty"`
	SpecReferences    []SpecReference                   `json:"spec_references,omitempty"`
	SecuritySchemes   []EffectiveSecurityScheme         `json:"security_schemes,omitempty"`
	RootSecurity      []EffectiveSecurityRequirement    `json:"root_security,omitempty"`
	OperationSecurity []EffectiveOperationSecurity      `json:"operation_security,omitempty"`
	Conflicts         []SecurityInspectionConflict      `json:"conflicts,omitempty"`
	SourceNotes       []SecurityInspectionSourceNote    `json:"source_notes,omitempty"`
}

// SecurityInspectionClassification records source-backed upstream catalog
// classification metadata in an inspection view.
type SecurityInspectionClassification struct {
	Status     AuthCompletenessStatus `json:"status"`
	SpecRefID  string                 `json:"spec_ref_id,omitempty"`
	Provenance SecurityProvenance     `json:"provenance"`
	SourceRefs []string               `json:"source_refs,omitempty"`
	SourceNote string                 `json:"source_note,omitempty"`
}

// EffectiveSecurityScheme is a security scheme plus provenance and source
// labels. It is metadata only and never contains credential values.
type EffectiveSecurityScheme struct {
	Scheme     SecurityScheme     `json:"scheme"`
	Provenance SecurityProvenance `json:"provenance"`
	OverlayID  string             `json:"overlay_id,omitempty"`
	SpecRefID  string             `json:"spec_ref_id,omitempty"`
	SourceRefs []string           `json:"source_refs,omitempty"`
	SourceNote string             `json:"source_note,omitempty"`
}

// EffectiveSecurityRequirement is an OpenAPI-style security requirement plus
// provenance and source labels.
type EffectiveSecurityRequirement struct {
	Requirement SecurityRequirement `json:"requirement"`
	Provenance  SecurityProvenance  `json:"provenance"`
	OverlayID   string              `json:"overlay_id,omitempty"`
	SpecRefID   string              `json:"spec_ref_id,omitempty"`
	SourceRefs  []string            `json:"source_refs,omitempty"`
	SourceNote  string              `json:"source_note,omitempty"`
}

// EffectiveOperationSecurity is operation-level security metadata plus
// provenance-labeled security requirements.
type EffectiveOperationSecurity struct {
	Match      OperationMatch                 `json:"match"`
	Provenance SecurityProvenance             `json:"provenance"`
	OverlayID  string                         `json:"overlay_id,omitempty"`
	SpecRefID  string                         `json:"spec_ref_id,omitempty"`
	Security   []EffectiveSecurityRequirement `json:"security,omitempty"`
}

// SecurityInspectionConflict records an advisory issue or review point. These
// conflicts do not make the view authoritative and do not imply execution.
type SecurityInspectionConflict struct {
	Type       SecurityInspectionConflictType `json:"type"`
	ProviderID string                         `json:"provider_id"`
	OverlayID  string                         `json:"overlay_id,omitempty"`
	SpecRefID  string                         `json:"spec_ref_id,omitempty"`
	Scheme     string                         `json:"scheme,omitempty"`
	Match      OperationMatch                 `json:"match,omitzero,omitempty"`
	Message    string                         `json:"message"`
}

// SecurityInspectionSourceNote preserves source notes with provenance labels.
type SecurityInspectionSourceNote struct {
	Provenance SecurityProvenance `json:"provenance"`
	OverlayID  string             `json:"overlay_id,omitempty"`
	SpecRefID  string             `json:"spec_ref_id,omitempty"`
	SourceRefs []string           `json:"source_refs,omitempty"`
	SourceNote string             `json:"source_note"`
}

// BuiltInSecurityInspectionView builds an inspection view for a built-in
// provider using built-in classifications and security overlays.
func BuiltInSecurityInspectionView(providerKey string) (SecurityInspectionView, error) {
	return BuildSecurityInspectionView(SecurityInspectionOptions{ProviderKey: providerKey})
}

// BuildSecurityInspectionView combines provider metadata, source-backed
// classifications, and security overlays into a deterministic read-only
// inspection view. It reports overlay effects and conflicts without writing or
// mutating OpenAPI documents.
func BuildSecurityInspectionView(options SecurityInspectionOptions) (SecurityInspectionView, error) {
	catalog := options.Catalog
	classifications := options.SecurityClassifications
	if catalog.isZero() {
		catalog = BuiltInCatalog()
		if classifications == nil {
			classifications = BuiltInSecurityClassifications()
		}
	}
	if err := ValidateProviders(catalog.Providers); err != nil {
		return SecurityInspectionView{}, err
	}
	provider, ok := catalog.FindProvider(options.ProviderKey)
	if !ok {
		return SecurityInspectionView{}, fmt.Errorf("unknown provider %q", options.ProviderKey)
	}
	providerByID := map[string]Provider{}
	for _, p := range catalog.Providers {
		providerByID[p.ID] = p
	}

	classification, hasClassification, err := inspectionClassificationForProvider(classifications, providerByID, provider.ID)
	if err != nil {
		return SecurityInspectionView{}, err
	}
	overlays, err := inspectionOverlaysForProvider(catalog.SecurityOverlays, providerByID, provider.ID)
	if err != nil {
		return SecurityInspectionView{}, err
	}

	view := SecurityInspectionView{
		ProviderID:     provider.ID,
		DisplayName:    provider.DisplayName,
		Status:         AuthStatusUnknown,
		SpecReferences: append([]SpecReference(nil), provider.SpecReferences...),
	}
	if hasClassification {
		view.Status = classification.Status
		inspection := SecurityInspectionClassification{
			Status:     classification.Status,
			SpecRefID:  classification.SpecRefID,
			Provenance: SecurityProvenanceClassification,
			SourceRefs: append([]string(nil), classification.SourceRefs...),
			SourceNote: classification.SourceNote,
		}
		view.Classification = &inspection
		view.SourceNotes = append(view.SourceNotes, SecurityInspectionSourceNote{
			Provenance: SecurityProvenanceClassification,
			SpecRefID:  classification.SpecRefID,
			SourceRefs: append([]string(nil), classification.SourceRefs...),
			SourceNote: classification.SourceNote,
		})
	}

	schemeByName := map[string]EffectiveSecurityScheme{}
	knownOperations := cloneOperationMatches(options.KnownOperations)
	for _, overlay := range overlays {
		view.Status = overlay.Status
		view.SourceNotes = append(view.SourceNotes, SecurityInspectionSourceNote{
			Provenance: SecurityProvenanceOverlay,
			OverlayID:  overlay.ID,
			SpecRefID:  overlay.SpecRefID,
			SourceRefs: append([]string(nil), overlay.SourceRefs...),
			SourceNote: overlay.SourceNote,
		})
		for _, scheme := range overlay.SecuritySchemes {
			effective := EffectiveSecurityScheme{
				Scheme:     cloneSecurityScheme(scheme),
				Provenance: SecurityProvenanceOverlay,
				OverlayID:  overlay.ID,
				SpecRefID:  overlay.SpecRefID,
				SourceRefs: append([]string(nil), overlay.SourceRefs...),
				SourceNote: overlay.SourceNote,
			}
			if previous, exists := schemeByName[scheme.Name]; exists {
				view.Conflicts = append(view.Conflicts, SecurityInspectionConflict{
					Type:       SecurityInspectionConflictDuplicateScheme,
					ProviderID: provider.ID,
					OverlayID:  overlay.ID,
					SpecRefID:  overlay.SpecRefID,
					Scheme:     scheme.Name,
					Message:    fmt.Sprintf("overlay scheme %q duplicates scheme from %s", scheme.Name, previous.OverlayID),
				})
			}
			schemeByName[scheme.Name] = effective
			view.SecuritySchemes = append(view.SecuritySchemes, effective)
			view.Conflicts = append(view.Conflicts, SecurityInspectionConflict{
				Type:       SecurityInspectionConflictOverlayOnlyAddition,
				ProviderID: provider.ID,
				OverlayID:  overlay.ID,
				SpecRefID:  overlay.SpecRefID,
				Scheme:     scheme.Name,
				Message:    fmt.Sprintf("overlay adds advisory security scheme %q", scheme.Name),
			})
		}
		for _, requirement := range overlay.RootSecurity {
			effective := effectiveRequirement(requirement, overlay)
			view.RootSecurity = append(view.RootSecurity, effective)
			if _, exists := schemeByName[requirement.Scheme]; !exists {
				view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, OperationMatch{}))
			}
		}
		for _, operation := range overlay.OperationSecurity {
			provenance := SecurityProvenanceOverlay
			resolvedOperation := operationMatchResolved(operation.Match, knownOperations)
			if !resolvedOperation {
				provenance = SecurityProvenanceUnresolved
			}
			effective := EffectiveOperationSecurity{
				Match:      cloneOperationMatch(operation.Match),
				Provenance: provenance,
				OverlayID:  overlay.ID,
				SpecRefID:  overlay.SpecRefID,
			}
			for _, requirement := range operation.Security {
				effective.Security = append(effective.Security, effectiveRequirement(requirement, overlay))
				if _, exists := schemeByName[requirement.Scheme]; !exists {
					view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, operation.Match))
				}
			}
			if !resolvedOperation {
				view.Conflicts = append(view.Conflicts, SecurityInspectionConflict{
					Type:       SecurityInspectionConflictUnresolvedOperation,
					ProviderID: provider.ID,
					OverlayID:  overlay.ID,
					SpecRefID:  overlay.SpecRefID,
					Match:      cloneOperationMatch(operation.Match),
					Message:    "overlay operation security target was not resolved against known operations",
				})
			}
			view.OperationSecurity = append(view.OperationSecurity, effective)
		}
	}

	sortSecurityInspectionView(&view)
	return cloneSecurityInspectionView(view), nil
}

func inspectionClassificationForProvider(classifications []SecurityClassification, providers map[string]Provider, providerID string) (SecurityClassification, bool, error) {
	var found SecurityClassification
	var ok bool
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providers); err != nil {
			return SecurityClassification{}, false, fmt.Errorf("security classification[%d]: %w", i, err)
		}
		if classification.ProviderID != providerID {
			continue
		}
		if ok {
			return SecurityClassification{}, false, fmt.Errorf("security classification %q: duplicate provider", providerID)
		}
		found = classification
		ok = true
	}
	return found, ok, nil
}

func inspectionOverlaysForProvider(overlays []SecurityOverlay, providers map[string]Provider, providerID string) ([]SecurityOverlay, error) {
	var out []SecurityOverlay
	seenIDs := map[string]struct{}{}
	for i, overlay := range overlays {
		if err := validateInspectionOverlayEnvelope(overlay, providers); err != nil {
			return nil, fmt.Errorf("security overlay[%d]: %w", i, err)
		}
		if _, exists := seenIDs[overlay.ID]; exists {
			return nil, fmt.Errorf("security overlay %q: duplicate id", overlay.ID)
		}
		seenIDs[overlay.ID] = struct{}{}
		if overlay.ProviderID == providerID {
			out = append(out, cloneSecurityOverlay(overlay))
		}
	}
	sortSecurityOverlays(out)
	return out, nil
}

func validateInspectionOverlayEnvelope(overlay SecurityOverlay, providers map[string]Provider) error {
	if strings.TrimSpace(overlay.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(overlay.ID) {
		return fmt.Errorf("invalid id %q", overlay.ID)
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
	for i, scheme := range overlay.SecuritySchemes {
		if err := validateSecurityScheme(scheme); err != nil {
			return fmt.Errorf("overlay %q scheme[%d]: %w", overlay.ID, i, err)
		}
	}
	for i, requirement := range overlay.RootSecurity {
		if err := validateSecurityRequirementShape(requirement); err != nil {
			return fmt.Errorf("overlay %q root security[%d]: %w", overlay.ID, i, err)
		}
	}
	for i, operation := range overlay.OperationSecurity {
		if err := validateOperationMatch(operation.Match); err != nil {
			return fmt.Errorf("overlay %q operation security[%d]: %w", overlay.ID, i, err)
		}
		for j, requirement := range operation.Security {
			if err := validateSecurityRequirementShape(requirement); err != nil {
				return fmt.Errorf("overlay %q operation security[%d].security[%d]: %w", overlay.ID, i, j, err)
			}
		}
	}
	return nil
}

func validateSecurityRequirementShape(requirement SecurityRequirement) error {
	if !validSchemeName(requirement.Scheme) {
		return fmt.Errorf("invalid security requirement scheme %q", requirement.Scheme)
	}
	return validateUniqueStrings("scope", requirement.Scopes)
}

func effectiveRequirement(requirement SecurityRequirement, overlay SecurityOverlay) EffectiveSecurityRequirement {
	return EffectiveSecurityRequirement{
		Requirement: cloneSecurityRequirement(requirement),
		Provenance:  SecurityProvenanceOverlay,
		OverlayID:   overlay.ID,
		SpecRefID:   overlay.SpecRefID,
		SourceRefs:  append([]string(nil), overlay.SourceRefs...),
		SourceNote:  overlay.SourceNote,
	}
}

func missingSchemeConflict(providerID string, overlay SecurityOverlay, scheme string, match OperationMatch) SecurityInspectionConflict {
	return SecurityInspectionConflict{
		Type:       SecurityInspectionConflictMissingScheme,
		ProviderID: providerID,
		OverlayID:  overlay.ID,
		SpecRefID:  overlay.SpecRefID,
		Scheme:     scheme,
		Match:      cloneOperationMatch(match),
		Message:    fmt.Sprintf("security requirement references missing scheme %q", scheme),
	}
}

func operationMatchResolved(match OperationMatch, known []OperationMatch) bool {
	if len(known) == 0 {
		return false
	}
	for _, candidate := range known {
		if strings.TrimSpace(match.OperationID) != "" && strings.TrimSpace(match.OperationID) == strings.TrimSpace(candidate.OperationID) {
			return true
		}
		if strings.TrimSpace(match.Method) != "" || strings.TrimSpace(match.Path) != "" {
			if strings.EqualFold(strings.TrimSpace(match.Method), strings.TrimSpace(candidate.Method)) && strings.TrimSpace(match.Path) == strings.TrimSpace(candidate.Path) {
				return true
			}
		}
		if len(match.Tags) > 0 && containsAllStrings(candidate.Tags, match.Tags) {
			return true
		}
	}
	return false
}

func containsAllStrings(values, required []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		seen[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range required {
		if _, ok := seen[strings.TrimSpace(value)]; !ok {
			return false
		}
	}
	return true
}

func sortSecurityInspectionView(view *SecurityInspectionView) {
	sort.SliceStable(view.SpecReferences, func(i, j int) bool {
		return view.SpecReferences[i].ID < view.SpecReferences[j].ID
	})
	sort.SliceStable(view.SecuritySchemes, func(i, j int) bool {
		if view.SecuritySchemes[i].Scheme.Name == view.SecuritySchemes[j].Scheme.Name {
			return view.SecuritySchemes[i].OverlayID < view.SecuritySchemes[j].OverlayID
		}
		return view.SecuritySchemes[i].Scheme.Name < view.SecuritySchemes[j].Scheme.Name
	})
	sort.SliceStable(view.RootSecurity, func(i, j int) bool {
		if view.RootSecurity[i].Requirement.Scheme == view.RootSecurity[j].Requirement.Scheme {
			return view.RootSecurity[i].OverlayID < view.RootSecurity[j].OverlayID
		}
		return view.RootSecurity[i].Requirement.Scheme < view.RootSecurity[j].Requirement.Scheme
	})
	sort.SliceStable(view.OperationSecurity, func(i, j int) bool {
		left := operationMatchSortKey(view.OperationSecurity[i].Match)
		right := operationMatchSortKey(view.OperationSecurity[j].Match)
		if left == right {
			return view.OperationSecurity[i].OverlayID < view.OperationSecurity[j].OverlayID
		}
		return left < right
	})
	sort.SliceStable(view.Conflicts, func(i, j int) bool {
		left := conflictSortKey(view.Conflicts[i])
		right := conflictSortKey(view.Conflicts[j])
		return left < right
	})
	sort.SliceStable(view.SourceNotes, func(i, j int) bool {
		left := sourceNoteSortKey(view.SourceNotes[i])
		right := sourceNoteSortKey(view.SourceNotes[j])
		return left < right
	})
}

func operationMatchSortKey(match OperationMatch) string {
	return strings.Join([]string{
		match.OperationID,
		strings.ToUpper(match.Method),
		match.Path,
		strings.Join(sortedStrings(match.Tags), ","),
	}, "\x00")
}

func conflictSortKey(conflict SecurityInspectionConflict) string {
	return strings.Join([]string{
		string(conflict.Type),
		conflict.ProviderID,
		conflict.OverlayID,
		conflict.SpecRefID,
		conflict.Scheme,
		operationMatchSortKey(conflict.Match),
		conflict.Message,
	}, "\x00")
}

func sourceNoteSortKey(note SecurityInspectionSourceNote) string {
	return strings.Join([]string{
		string(note.Provenance),
		note.OverlayID,
		note.SpecRefID,
		note.SourceNote,
	}, "\x00")
}

func cloneSecurityInspectionView(view SecurityInspectionView) SecurityInspectionView {
	out := view
	if view.Classification != nil {
		classification := *view.Classification
		classification.SourceRefs = append([]string(nil), classification.SourceRefs...)
		out.Classification = &classification
	}
	out.SpecReferences = append([]SpecReference(nil), view.SpecReferences...)
	out.SecuritySchemes = cloneEffectiveSecuritySchemes(view.SecuritySchemes)
	out.RootSecurity = cloneEffectiveSecurityRequirements(view.RootSecurity)
	out.OperationSecurity = cloneEffectiveOperationSecurity(view.OperationSecurity)
	out.Conflicts = cloneSecurityInspectionConflicts(view.Conflicts)
	out.SourceNotes = cloneSecurityInspectionSourceNotes(view.SourceNotes)
	return out
}

func cloneEffectiveSecuritySchemes(in []EffectiveSecurityScheme) []EffectiveSecurityScheme {
	out := make([]EffectiveSecurityScheme, len(in))
	for i, scheme := range in {
		out[i] = scheme
		out[i].Scheme = cloneSecurityScheme(scheme.Scheme)
		out[i].SourceRefs = append([]string(nil), scheme.SourceRefs...)
	}
	return out
}

func cloneEffectiveSecurityRequirements(in []EffectiveSecurityRequirement) []EffectiveSecurityRequirement {
	out := make([]EffectiveSecurityRequirement, len(in))
	for i, requirement := range in {
		out[i] = requirement
		out[i].Requirement = cloneSecurityRequirement(requirement.Requirement)
		out[i].SourceRefs = append([]string(nil), requirement.SourceRefs...)
	}
	return out
}

func cloneEffectiveOperationSecurity(in []EffectiveOperationSecurity) []EffectiveOperationSecurity {
	out := make([]EffectiveOperationSecurity, len(in))
	for i, operation := range in {
		out[i] = operation
		out[i].Match = cloneOperationMatch(operation.Match)
		out[i].Security = cloneEffectiveSecurityRequirements(operation.Security)
	}
	return out
}

func cloneSecurityInspectionConflicts(in []SecurityInspectionConflict) []SecurityInspectionConflict {
	out := make([]SecurityInspectionConflict, len(in))
	for i, conflict := range in {
		out[i] = conflict
		out[i].Match = cloneOperationMatch(conflict.Match)
	}
	return out
}

func cloneSecurityInspectionSourceNotes(in []SecurityInspectionSourceNote) []SecurityInspectionSourceNote {
	out := make([]SecurityInspectionSourceNote, len(in))
	for i, note := range in {
		out[i] = note
		out[i].SourceRefs = append([]string(nil), note.SourceRefs...)
	}
	return out
}

func cloneSecurityScheme(scheme SecurityScheme) SecurityScheme {
	scheme.Flows = cloneOAuthFlows(scheme.Flows)
	scheme.Scopes = append([]string(nil), scheme.Scopes...)
	return scheme
}

func cloneSecurityRequirement(requirement SecurityRequirement) SecurityRequirement {
	requirement.Scopes = append([]string(nil), requirement.Scopes...)
	return requirement
}

func cloneOperationMatch(match OperationMatch) OperationMatch {
	match.Tags = append([]string(nil), match.Tags...)
	return match
}

func cloneOperationMatches(in []OperationMatch) []OperationMatch {
	out := make([]OperationMatch, len(in))
	for i, match := range in {
		out[i] = cloneOperationMatch(match)
	}
	return out
}
