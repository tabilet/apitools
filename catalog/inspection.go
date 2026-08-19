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
	ProviderID      string                             `json:"provider_id"`
	DisplayName     string                             `json:"display_name,omitempty"`
	Status          AuthCompletenessStatus             `json:"status"`
	Classifications []SecurityInspectionClassification `json:"classifications,omitempty"`
	// Classification retains the first deterministic classification for
	// compatibility with callers written before per-spec dispositions.
	Classification    *SecurityInspectionClassification `json:"classification,omitempty"`
	SpecReferences    []SpecReference                   `json:"spec_references,omitempty"`
	SecuritySchemes   []EffectiveSecurityScheme         `json:"security_schemes,omitempty"`
	RootSecurity      []EffectiveSecurityRequirement    `json:"root_security,omitempty"`
	RootSecuritySets  []EffectiveSecurityRequirementSet `json:"root_security_sets,omitempty"`
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

// EffectiveSecurityRequirementSet is one provenance-labeled OpenAPI-style
// security requirement object. Requirements in the same set are ANDed.
type EffectiveSecurityRequirementSet struct {
	Requirements []EffectiveSecurityRequirement `json:"requirements,omitempty"`
	Provenance   SecurityProvenance             `json:"provenance"`
	OverlayID    string                         `json:"overlay_id,omitempty"`
	SpecRefID    string                         `json:"spec_ref_id,omitempty"`
	SourceRefs   []string                       `json:"source_refs,omitempty"`
	SourceNote   string                         `json:"source_note,omitempty"`
}

// EffectiveOperationSecurity is operation-level security metadata plus
// provenance-labeled security requirements.
type EffectiveOperationSecurity struct {
	Match        OperationMatch                    `json:"match"`
	Provenance   SecurityProvenance                `json:"provenance"`
	OverlayID    string                            `json:"overlay_id,omitempty"`
	SpecRefID    string                            `json:"spec_ref_id,omitempty"`
	Security     []EffectiveSecurityRequirement    `json:"security,omitempty"`
	SecuritySets []EffectiveSecurityRequirementSet `json:"security_sets,omitempty"`
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

	providerClassifications, err := inspectionClassificationsForProvider(classifications, providerByID, provider.ID)
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
		Status:         inspectionSecurityStatus(provider.ID, providerClassifications, overlays),
		SpecReferences: append([]SpecReference(nil), provider.SpecReferences...),
	}
	for _, classification := range providerClassifications {
		inspection := SecurityInspectionClassification{
			Status:     classification.Status,
			SpecRefID:  classification.SpecRefID,
			Provenance: SecurityProvenanceClassification,
			SourceRefs: append([]string(nil), classification.SourceRefs...),
			SourceNote: classification.SourceNote,
		}
		view.Classifications = append(view.Classifications, inspection)
		view.SourceNotes = append(view.SourceNotes, SecurityInspectionSourceNote{
			Provenance: SecurityProvenanceClassification,
			SpecRefID:  classification.SpecRefID,
			SourceRefs: append([]string(nil), classification.SourceRefs...),
			SourceNote: classification.SourceNote,
		})
	}
	if len(view.Classifications) > 0 {
		classification := view.Classifications[0]
		classification.SourceRefs = append([]string(nil), classification.SourceRefs...)
		view.Classification = &classification
	}

	schemeByName := map[string]EffectiveSecurityScheme{}
	knownOperations := cloneOperationMatches(options.KnownOperations)
	for _, overlay := range overlays {
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
		if len(overlay.RootSecuritySets) == 0 {
			for _, requirement := range overlay.RootSecurity {
				effective := effectiveRequirement(requirement, overlay)
				view.RootSecurity = append(view.RootSecurity, effective)
				if _, exists := schemeByName[requirement.Scheme]; !exists {
					view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, OperationMatch{}))
				}
			}
		}
		for _, set := range overlay.RootSecuritySets {
			effective := effectiveRequirementSet(set, overlay)
			view.RootSecuritySets = append(view.RootSecuritySets, effective)
			for _, requirement := range set.Requirements {
				if _, exists := schemeByName[requirement.Scheme]; !exists {
					view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, OperationMatch{}))
				}
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
			if len(operation.SecuritySets) == 0 {
				for _, requirement := range operation.Security {
					effective.Security = append(effective.Security, effectiveRequirement(requirement, overlay))
					if _, exists := schemeByName[requirement.Scheme]; !exists {
						view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, operation.Match))
					}
				}
			}
			for _, set := range operation.SecuritySets {
				effective.SecuritySets = append(effective.SecuritySets, effectiveRequirementSet(set, overlay))
				for _, requirement := range set.Requirements {
					if _, exists := schemeByName[requirement.Scheme]; !exists {
						view.Conflicts = append(view.Conflicts, missingSchemeConflict(provider.ID, overlay, requirement.Scheme, operation.Match))
					}
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

func inspectionSecurityStatus(providerID string, classifications []SecurityClassification, overlays []SecurityOverlay) AuthCompletenessStatus {
	classificationByScope := map[string]SecurityClassification{}
	overlaysByScope := map[string][]SecurityOverlay{}
	for _, classification := range classifications {
		classificationByScope[securityScopeKey(providerID, classification.SpecRefID)] = classification
	}
	for _, overlay := range overlays {
		key := securityScopeKey(providerID, overlay.SpecRefID)
		overlaysByScope[key] = append(overlaysByScope[key], overlay)
	}
	var dispositions []SecurityDisposition
	for _, specRefID := range securityScopesForProvider(providerID, classificationByScope, overlaysByScope) {
		dispositions = append(dispositions, buildSecurityDisposition(
			specRefID,
			classificationByScope[securityScopeKey(providerID, specRefID)],
			overlaysByScope[securityScopeKey(providerID, specRefID)],
		))
	}
	return aggregateSecurityDispositionStatus(dispositions)
}

func inspectionClassificationsForProvider(classifications []SecurityClassification, providers map[string]Provider, providerID string) ([]SecurityClassification, error) {
	var found []SecurityClassification
	seenScopes := map[string]struct{}{}
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providers); err != nil {
			return nil, fmt.Errorf("security classification[%d]: %w", i, err)
		}
		if classification.ProviderID != providerID {
			continue
		}
		scope := strings.TrimSpace(classification.SpecRefID)
		if _, ok := seenScopes[scope]; ok {
			return nil, fmt.Errorf("security classification %q/%q: duplicate provider/spec scope", providerID, scope)
		}
		seenScopes[scope] = struct{}{}
		found = append(found, classification)
	}
	sortSecurityClassifications(found)
	return found, nil
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
	if !validAuthEvidenceStatus(overlay.Status) {
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
	for i, set := range overlay.RootSecuritySets {
		if err := validateSecurityRequirementSetShape(set); err != nil {
			return fmt.Errorf("overlay %q root security sets[%d]: %w", overlay.ID, i, err)
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
		for j, set := range operation.SecuritySets {
			if err := validateSecurityRequirementSetShape(set); err != nil {
				return fmt.Errorf("overlay %q operation security[%d].security_sets[%d]: %w", overlay.ID, i, j, err)
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

func validateSecurityRequirementSetShape(set SecurityRequirementSet) error {
	if len(set.Requirements) == 0 {
		return fmt.Errorf("security requirement set must include at least one requirement")
	}
	seen := map[string]struct{}{}
	for i, requirement := range set.Requirements {
		if err := validateSecurityRequirementShape(requirement); err != nil {
			return fmt.Errorf("requirements[%d]: %w", i, err)
		}
		if _, exists := seen[requirement.Scheme]; exists {
			return fmt.Errorf("duplicate security requirement scheme %q", requirement.Scheme)
		}
		seen[requirement.Scheme] = struct{}{}
	}
	return nil
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

func effectiveRequirementSet(set SecurityRequirementSet, overlay SecurityOverlay) EffectiveSecurityRequirementSet {
	out := EffectiveSecurityRequirementSet{
		Provenance: SecurityProvenanceOverlay,
		OverlayID:  overlay.ID,
		SpecRefID:  overlay.SpecRefID,
		SourceRefs: append([]string(nil), overlay.SourceRefs...),
		SourceNote: overlay.SourceNote,
	}
	for _, requirement := range set.Requirements {
		out.Requirements = append(out.Requirements, effectiveRequirement(requirement, overlay))
	}
	return out
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
	sort.SliceStable(view.Classifications, func(i, j int) bool {
		return view.Classifications[i].SpecRefID < view.Classifications[j].SpecRefID
	})
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
	sortEffectiveSecurityRequirementSets(view.RootSecuritySets)
	sort.SliceStable(view.OperationSecurity, func(i, j int) bool {
		left := operationMatchSortKey(view.OperationSecurity[i].Match)
		right := operationMatchSortKey(view.OperationSecurity[j].Match)
		if left == right {
			return view.OperationSecurity[i].OverlayID < view.OperationSecurity[j].OverlayID
		}
		return left < right
	})
	for i := range view.OperationSecurity {
		sortEffectiveSecurityRequirementSets(view.OperationSecurity[i].SecuritySets)
	}
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

func sortEffectiveSecurityRequirementSets(sets []EffectiveSecurityRequirementSet) {
	for i := range sets {
		sort.SliceStable(sets[i].Requirements, func(j, k int) bool {
			return sets[i].Requirements[j].Requirement.Scheme < sets[i].Requirements[k].Requirement.Scheme
		})
	}
	sort.SliceStable(sets, func(i, j int) bool {
		left := effectiveSecurityRequirementSetSortKey(sets[i])
		right := effectiveSecurityRequirementSetSortKey(sets[j])
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

func effectiveSecurityRequirementSetSortKey(set EffectiveSecurityRequirementSet) string {
	parts := make([]string, 0, len(set.Requirements)+2)
	parts = append(parts, set.OverlayID, set.SpecRefID)
	for _, requirement := range set.Requirements {
		parts = append(parts, requirement.Requirement.Scheme)
	}
	return strings.Join(parts, "\x00")
}

func cloneSecurityInspectionView(view SecurityInspectionView) SecurityInspectionView {
	out := view
	out.Classifications = make([]SecurityInspectionClassification, len(view.Classifications))
	for i, classification := range view.Classifications {
		out.Classifications[i] = classification
		out.Classifications[i].SourceRefs = append([]string(nil), classification.SourceRefs...)
	}
	if view.Classification != nil {
		classification := *view.Classification
		classification.SourceRefs = append([]string(nil), classification.SourceRefs...)
		out.Classification = &classification
	}
	out.SpecReferences = append([]SpecReference(nil), view.SpecReferences...)
	out.SecuritySchemes = cloneEffectiveSecuritySchemes(view.SecuritySchemes)
	out.RootSecurity = cloneEffectiveSecurityRequirements(view.RootSecurity)
	out.RootSecuritySets = cloneEffectiveSecurityRequirementSets(view.RootSecuritySets)
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

func cloneEffectiveSecurityRequirementSets(in []EffectiveSecurityRequirementSet) []EffectiveSecurityRequirementSet {
	out := make([]EffectiveSecurityRequirementSet, len(in))
	for i, set := range in {
		out[i] = set
		out[i].Requirements = cloneEffectiveSecurityRequirements(set.Requirements)
		out[i].SourceRefs = append([]string(nil), set.SourceRefs...)
	}
	return out
}

func cloneEffectiveOperationSecurity(in []EffectiveOperationSecurity) []EffectiveOperationSecurity {
	out := make([]EffectiveOperationSecurity, len(in))
	for i, operation := range in {
		out[i] = operation
		out[i].Match = cloneOperationMatch(operation.Match)
		out[i].Security = cloneEffectiveSecurityRequirements(operation.Security)
		out[i].SecuritySets = cloneEffectiveSecurityRequirementSets(operation.SecuritySets)
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
