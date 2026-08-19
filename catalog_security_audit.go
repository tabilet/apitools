package apitools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
)

const (
	SecurityAuditDispositionCompleteUpstream          = "complete-upstream"
	SecurityAuditDispositionCompleteViaOverlay        = "complete-via-overlay"
	SecurityAuditDispositionPresentIncompleteReviewed = "present-incomplete-reviewed"
	SecurityAuditDispositionIntentionallyAnonymous    = "intentionally-anonymous"
	SecurityAuditDispositionQueuedSourceReReview      = "queued-source-re-review"
)

const (
	SecurityAuditArtifactNoLocalArtifact             = "no-local-artifact"
	SecurityAuditArtifactMissingFile                 = "missing-file"
	SecurityAuditArtifactNotOpenAPI                  = "not-openapi-or-swagger"
	SecurityAuditArtifactHasSecurityMetadata         = "has-security-metadata"
	SecurityAuditArtifactMissingSecurityMetadata     = "missing-security-metadata"
	SecurityAuditArtifactMissingSecuritySchemes      = "missing-security-schemes"
	SecurityAuditArtifactMissingSecurityRequirements = "missing-security-requirements"
	SecurityAuditArtifactUndeclaredSecuritySchemes   = "undeclared-security-schemes"
	SecurityAuditArtifactUnparseable                 = "unparseable"
)

// CatalogSecurityAuditOptions controls offline audit of built-in provider
// security metadata. The audit never fetches remote documents, resolves
// credentials, signs requests, or executes provider operations.
type CatalogSecurityAuditOptions struct {
	Catalog                 catalog.Catalog
	SecurityClassifications []catalog.SecurityClassification
	Artifacts               []catalog.CatalogSpecArtifact
	CacheDir                string
	MaxBytes                int64
}

// CatalogSecurityAuditReport records provider-level security dispositions and
// optional local OpenAPI/Swagger artifact security metadata.
type CatalogSecurityAuditReport struct {
	Summary   CatalogSecurityAuditSummary `json:"summary"`
	Providers []CatalogSecurityAuditRow   `json:"providers,omitempty"`
}

// CatalogSecurityAuditSummary records aggregate provider and artifact counts.
type CatalogSecurityAuditSummary struct {
	ProviderCount int                            `json:"provider_count"`
	Dispositions  []CatalogSecurityAuditCount    `json:"dispositions,omitempty"`
	AuthStatuses  []CatalogSecurityAuditCount    `json:"auth_statuses,omitempty"`
	Protocols     []CatalogSecurityAuditCount    `json:"protocols,omitempty"`
	Artifacts     []CatalogSecurityArtifactCount `json:"artifacts,omitempty"`
}

// CatalogSecurityAuditCount records one provider-count bucket.
type CatalogSecurityAuditCount struct {
	Name        string   `json:"name"`
	Count       int      `json:"count"`
	ProviderIDs []string `json:"provider_ids,omitempty"`
}

// CatalogSecurityArtifactCount records one local artifact security bucket.
type CatalogSecurityArtifactCount struct {
	Status     string   `json:"status"`
	Count      int      `json:"count"`
	SpecRefIDs []string `json:"spec_ref_ids,omitempty"`
}

// CatalogSecurityAuditRow records one provider's security audit disposition.
type CatalogSecurityAuditRow struct {
	ProviderID       string                            `json:"provider_id"`
	DisplayName      string                            `json:"display_name,omitempty"`
	Protocol         catalog.SpecProtocol              `json:"protocol"`
	AuthStatus       catalog.AuthCompletenessStatus    `json:"auth_status"`
	Disposition      string                            `json:"disposition"`
	SpecRefIDs       []string                          `json:"spec_ref_ids,omitempty"`
	OverlayIDs       []string                          `json:"overlay_ids,omitempty"`
	SourceNotes      []string                          `json:"source_notes,omitempty"`
	Reasons          []string                          `json:"reasons,omitempty"`
	ManualFollowUps  []string                          `json:"manual_follow_ups,omitempty"`
	ArtifactSecurity []CatalogSecurityArtifactAuditRow `json:"artifact_security,omitempty"`
}

// CatalogSecurityArtifactAuditRow records security metadata found in one local
// OpenAPI or Swagger artifact.
type CatalogSecurityArtifactAuditRow struct {
	ProviderID                    string   `json:"provider_id"`
	SpecRefID                     string   `json:"spec_ref_id"`
	Kind                          string   `json:"kind,omitempty"`
	Path                          string   `json:"path,omitempty"`
	Status                        string   `json:"status"`
	Exists                        bool     `json:"exists,omitempty"`
	SecuritySchemeCount           int      `json:"security_scheme_count,omitempty"`
	RootSecurityRequirementCount  int      `json:"root_security_requirement_count,omitempty"`
	OperationCount                int      `json:"operation_count,omitempty"`
	OperationSecurityCount        int      `json:"operation_security_count,omitempty"`
	UndeclaredSecuritySchemes     []string `json:"undeclared_security_schemes,omitempty"`
	ParseError                    string   `json:"parse_error,omitempty"`
	SecurityDefinitionsInspected  bool     `json:"security_definitions_inspected,omitempty"`
	SecurityRequirementsInspected bool     `json:"security_requirements_inspected,omitempty"`
	ManualFollowUps               []string `json:"manual_follow_ups,omitempty"`
}

// BuiltInCatalogSecurityAuditReport audits the built-in provider catalog.
func BuiltInCatalogSecurityAuditReport(opts CatalogSecurityAuditOptions) (CatalogSecurityAuditReport, error) {
	opts.Catalog = catalog.BuiltInCatalog()
	if opts.SecurityClassifications == nil {
		opts.SecurityClassifications = catalog.BuiltInSecurityClassifications()
	}
	return BuildCatalogSecurityAuditReport(opts)
}

// BuildCatalogSecurityAuditReport builds an offline provider security audit.
func BuildCatalogSecurityAuditReport(opts CatalogSecurityAuditOptions) (CatalogSecurityAuditReport, error) {
	opts = normalizeCatalogSecurityAuditOptions(opts)
	cat := opts.Catalog
	catalogWasZero := catalogSecurityAuditCatalogIsZero(cat)
	if catalogWasZero {
		cat = catalog.BuiltInCatalog()
	}
	classifications := opts.SecurityClassifications
	if classifications == nil && catalogWasZero {
		classifications = catalog.BuiltInSecurityClassifications()
	}
	if err := cat.Validate(); err != nil {
		return CatalogSecurityAuditReport{}, err
	}
	securityReport, err := catalog.BuildSecurityReport(cat.ListProviders(), cat.ListSecurityOverlays(), classifications)
	if err != nil {
		return CatalogSecurityAuditReport{}, err
	}
	securityByProvider := map[string]catalog.ProviderSecurityReport{}
	for _, row := range securityReport.Providers {
		securityByProvider[row.ProviderID] = row
	}
	artifactsByProvider := catalogSecurityArtifactsByProvider(opts.Artifacts)
	overlaysByProvider := catalogSecurityAuditOverlaysByProvider(cat.ListSecurityOverlays())

	var rows []CatalogSecurityAuditRow
	for _, provider := range cat.ListProviders() {
		security := securityByProvider[provider.ID]
		row := CatalogSecurityAuditRow{
			ProviderID:       provider.ID,
			DisplayName:      provider.DisplayName,
			Protocol:         primarySecurityAuditProtocol(provider),
			AuthStatus:       security.Status,
			SpecRefIDs:       append([]string(nil), security.SpecRefIDs...),
			OverlayIDs:       append([]string(nil), security.OverlayIDs...),
			SourceNotes:      append([]string(nil), security.SourceNotes...),
			ArtifactSecurity: auditProviderSecurityArtifacts(provider, artifactsByProvider[provider.ID], overlaysByProvider[provider.ID], opts),
		}
		row.Disposition, row.Reasons, row.ManualFollowUps = catalogSecurityDisposition(security, row.ArtifactSecurity)
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ProviderID < rows[j].ProviderID })
	return CatalogSecurityAuditReport{
		Summary:   buildCatalogSecurityAuditSummary(rows),
		Providers: rows,
	}, nil
}

func normalizeCatalogSecurityAuditOptions(opts CatalogSecurityAuditOptions) CatalogSecurityAuditOptions {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = CatalogRefreshReviewDefaultMaxBytes
	}
	return opts
}

func catalogSecurityAuditCatalogIsZero(cat catalog.Catalog) bool {
	return len(cat.Providers) == 0 && len(cat.SecurityOverlays) == 0
}

func primarySecurityAuditProtocol(provider catalog.Provider) catalog.SpecProtocol {
	for _, ref := range provider.SpecReferences {
		if ref.Kind != catalog.SpecKindHumanDocs {
			return ref.ProtocolClassification().Protocol
		}
	}
	for _, ref := range provider.SpecReferences {
		return ref.ProtocolClassification().Protocol
	}
	return catalog.SpecProtocolUnknown
}

func catalogSecurityArtifactsByProvider(artifacts []catalog.CatalogSpecArtifact) map[string][]catalog.CatalogSpecArtifact {
	out := map[string][]catalog.CatalogSpecArtifact{}
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.ProviderID) == "" {
			continue
		}
		out[artifact.ProviderID] = append(out[artifact.ProviderID], artifact)
	}
	for providerID := range out {
		sort.SliceStable(out[providerID], func(i, j int) bool {
			if out[providerID][i].SpecRefID == out[providerID][j].SpecRefID {
				return out[providerID][i].Path < out[providerID][j].Path
			}
			return out[providerID][i].SpecRefID < out[providerID][j].SpecRefID
		})
	}
	return out
}

func catalogSecurityAuditOverlaysByProvider(overlays []catalog.SecurityOverlay) map[string][]catalog.SecurityOverlay {
	out := map[string][]catalog.SecurityOverlay{}
	for _, overlay := range overlays {
		if strings.TrimSpace(overlay.ProviderID) == "" {
			continue
		}
		out[overlay.ProviderID] = append(out[overlay.ProviderID], overlay)
	}
	for providerID := range out {
		sort.SliceStable(out[providerID], func(i, j int) bool {
			return out[providerID][i].ID < out[providerID][j].ID
		})
	}
	return out
}

func auditProviderSecurityArtifacts(provider catalog.Provider, artifacts []catalog.CatalogSpecArtifact, overlays []catalog.SecurityOverlay, opts CatalogSecurityAuditOptions) []CatalogSecurityArtifactAuditRow {
	var rows []CatalogSecurityArtifactAuditRow
	for _, ref := range provider.SpecReferences {
		protocol := ref.ProtocolClassification().Protocol
		if protocol != catalog.SpecProtocolOpenAPI && protocol != catalog.SpecProtocolSwagger {
			continue
		}
		artifact, ok := findSecurityAuditArtifact(artifacts, ref.ID)
		if !ok {
			rows = append(rows, CatalogSecurityArtifactAuditRow{
				ProviderID: provider.ID,
				SpecRefID:  ref.ID,
				Kind:       string(ref.Kind),
				Status:     SecurityAuditArtifactNoLocalArtifact,
				ManualFollowUps: []string{
					"Register or refresh the official artifact before re-auditing upstream security metadata.",
				},
			})
			continue
		}
		row := auditOpenAPISecurityArtifact(provider.ID, ref.ID, artifact, opts)
		applyCoveredSecurityArtifactFollowUp(&row, ref, overlays)
		rows = append(rows, row)
	}
	return rows
}

func findSecurityAuditArtifact(artifacts []catalog.CatalogSpecArtifact, specRefID string) (catalog.CatalogSpecArtifact, bool) {
	for _, artifact := range artifacts {
		if artifact.SpecRefID == specRefID || artifact.ArtifactID == specRefID {
			return artifact, true
		}
	}
	return catalog.CatalogSpecArtifact{}, false
}

func auditOpenAPISecurityArtifact(providerID, specRefID string, artifact catalog.CatalogSpecArtifact, opts CatalogSecurityAuditOptions) CatalogSecurityArtifactAuditRow {
	row := CatalogSecurityArtifactAuditRow{
		ProviderID: providerID,
		SpecRefID:  specRefID,
		Kind:       strings.TrimSpace(artifact.Kind),
		Path:       strings.TrimSpace(artifact.Path),
	}
	if row.Path == "" || strings.TrimSpace(opts.CacheDir) == "" {
		row.Status = SecurityAuditArtifactNoLocalArtifact
		row.ManualFollowUps = append(row.ManualFollowUps, "Register a local artifact path before auditing upstream OpenAPI security metadata.")
		return row
	}
	savedPath, err := resolveCatalogRefreshReviewPath(opts.CacheDir, row.Path)
	if err != nil {
		row.Status = SecurityAuditArtifactUnparseable
		row.ParseError = err.Error()
		row.ManualFollowUps = append(row.ManualFollowUps, "Fix the registered artifact path before auditing upstream OpenAPI security metadata.")
		return row
	}
	content, _, exists, err := readCatalogRefreshReviewArtifact(opts.CacheDir, savedPath, opts.MaxBytes)
	row.Exists = exists
	if err != nil {
		if exists {
			row.Status = SecurityAuditArtifactUnparseable
			row.ParseError = err.Error()
			row.ManualFollowUps = append(row.ManualFollowUps, "Review or replace the local artifact before auditing upstream OpenAPI security metadata.")
			return row
		}
		row.Status = SecurityAuditArtifactMissingFile
		row.ParseError = err.Error()
		row.ManualFollowUps = append(row.ManualFollowUps, "Restore the registered artifact file before auditing upstream OpenAPI security metadata.")
		return row
	}
	root, err := parseInventoryDocument(content)
	if err != nil {
		row.Status = SecurityAuditArtifactUnparseable
		row.ParseError = err.Error()
		row.ManualFollowUps = append(row.ManualFollowUps, "Review or replace the artifact before auditing upstream OpenAPI security metadata.")
		return row
	}
	if strings.TrimSpace(stringValue(root["openapi"])) == "" && strings.TrimSpace(stringValue(root["swagger"])) == "" {
		row.Status = SecurityAuditArtifactNotOpenAPI
		return row
	}
	schemes := securitySchemes(root)
	row.SecuritySchemeCount = len(schemes)
	rootNames := securityRequirementNames(root["security"])
	row.RootSecurityRequirementCount = len(rootNames)
	row.SecurityDefinitionsInspected = true
	row.SecurityRequirementsInspected = true
	opCount, opSecCount, undeclared := operationSecurityAuditCounts(root, schemes)
	row.OperationCount = opCount
	row.OperationSecurityCount = opSecCount
	undeclared = append(undeclared, undeclaredSecurityRequirementNames(rootNames, schemes)...)
	row.UndeclaredSecuritySchemes = sortedUniqueStrings(undeclared)
	switch {
	case len(row.UndeclaredSecuritySchemes) > 0 && row.SecuritySchemeCount == 0:
		row.Status = SecurityAuditArtifactMissingSecuritySchemes
		row.ManualFollowUps = append(row.ManualFollowUps, "Add or verify catalog security overlay metadata for referenced but undeclared security schemes.")
	case len(row.UndeclaredSecuritySchemes) > 0:
		row.Status = SecurityAuditArtifactUndeclaredSecuritySchemes
		row.ManualFollowUps = append(row.ManualFollowUps, "Review upstream security requirements that reference undeclared schemes.")
	case row.SecuritySchemeCount == 0 && row.RootSecurityRequirementCount == 0 && row.OperationSecurityCount == 0:
		row.Status = SecurityAuditArtifactMissingSecurityMetadata
		row.ManualFollowUps = append(row.ManualFollowUps, "Review provider auth docs and add a source-backed security overlay if endpoints are protected.")
	case row.SecuritySchemeCount == 0:
		row.Status = SecurityAuditArtifactMissingSecuritySchemes
		row.ManualFollowUps = append(row.ManualFollowUps, "Review provider auth docs and add a source-backed security overlay if endpoints are protected.")
	case row.RootSecurityRequirementCount == 0 && row.OperationSecurityCount == 0:
		row.Status = SecurityAuditArtifactMissingSecurityRequirements
		row.ManualFollowUps = append(row.ManualFollowUps, "Review provider auth docs and add root or operation security overlay metadata if endpoints are protected.")
	default:
		row.Status = SecurityAuditArtifactHasSecurityMetadata
	}
	return row
}

func applyCoveredSecurityArtifactFollowUp(row *CatalogSecurityArtifactAuditRow, ref catalog.SpecReference, overlays []catalog.SecurityOverlay) {
	switch row.Status {
	case SecurityAuditArtifactMissingSecurityMetadata,
		SecurityAuditArtifactMissingSecuritySchemes,
		SecurityAuditArtifactMissingSecurityRequirements,
		SecurityAuditArtifactUndeclaredSecuritySchemes:
	default:
		return
	}
	overlayIDs := coveringSecurityOverlayIDs(ref, overlays)
	if len(overlayIDs) == 0 {
		return
	}
	row.ManualFollowUps = []string{
		fmt.Sprintf("Reviewed by catalog security overlay(s) %s; upstream artifact still reports %s for source review.", strings.Join(overlayIDs, ", "), row.Status),
	}
}

func coveringSecurityOverlayIDs(ref catalog.SpecReference, overlays []catalog.SecurityOverlay) []string {
	specRefID := strings.TrimSpace(ref.ID)
	specURL := strings.TrimSpace(ref.URL)
	var out []string
	for _, overlay := range overlays {
		if strings.TrimSpace(overlay.SpecRefID) == specRefID {
			out = append(out, overlay.ID)
			continue
		}
		if specURL == "" {
			continue
		}
		for _, sourceRef := range overlay.SourceRefs {
			if strings.TrimSpace(sourceRef) == specURL {
				out = append(out, overlay.ID)
				break
			}
		}
	}
	return sortedUniqueStrings(out)
}

func securityRequirementNames(value any) []string {
	var out []string
	for _, requirementValue := range sliceValue(value) {
		for _, name := range sortedMapKeys(mapValue(requirementValue)) {
			out = append(out, name)
		}
	}
	return sortedUniqueStrings(out)
}

func operationSecurityAuditCounts(root map[string]any, schemes map[string]SecuritySummary) (operationCount int, operationSecurityCount int, undeclared []string) {
	for _, pathItemValue := range mapValue(root["paths"]) {
		pathItem := mapValue(pathItemValue)
		for _, method := range operationMethods(pathItem) {
			operationCount++
			operation := mapValue(pathItem[method])
			value, ok := operation["security"]
			if !ok {
				continue
			}
			names := securityRequirementNames(value)
			if len(names) == 0 {
				continue
			}
			operationSecurityCount++
			undeclared = append(undeclared, undeclaredSecurityRequirementNames(names, schemes)...)
		}
	}
	return operationCount, operationSecurityCount, sortedUniqueStrings(undeclared)
}

func undeclaredSecurityRequirementNames(names []string, schemes map[string]SecuritySummary) []string {
	var out []string
	for _, name := range names {
		if _, ok := schemes[name]; !ok {
			out = append(out, name)
		}
	}
	return out
}

func catalogSecurityDisposition(security catalog.ProviderSecurityReport, artifacts []CatalogSecurityArtifactAuditRow) (string, []string, []string) {
	var reasons []string
	var followUps []string
	switch {
	case security.Status == catalog.AuthStatusComplete && len(security.OverlayIDs) == 0:
		reasons = append(reasons, "security metadata is complete from upstream source classification")
		return SecurityAuditDispositionCompleteUpstream, reasons, nil
	case security.Status == catalog.AuthStatusPresentIncomplete:
		reasons = append(reasons, "security metadata is reviewed but remains environment, tenant, account, or user specific")
		if len(security.OverlayIDs) > 0 {
			reasons = append(reasons, "source-backed catalog security overlay supplies partial reviewed metadata")
		}
		if hasArtifactSecurityFinding(artifacts) {
			followUps = append(followUps, "Review local OpenAPI/Swagger artifact security metadata before changing classification.")
		}
		return SecurityAuditDispositionPresentIncompleteReviewed, reasons, followUps
	case security.Status == catalog.AuthStatusIntentionallyAnonymous:
		reasons = append(reasons, "provider source is intentionally anonymous or public")
		return SecurityAuditDispositionIntentionallyAnonymous, reasons, nil
	case len(security.OverlayIDs) > 0:
		reasons = append(reasons, "source-backed catalog security overlay supplies reviewed metadata")
		return SecurityAuditDispositionCompleteViaOverlay, reasons, nil
	default:
		reasons = append(reasons, "provider needs source re-review before security metadata can be considered complete")
		followUps = append(followUps, "Inspect provider-owned auth docs and local artifacts; add classification, overlay, or explicit no-portable-overlay decision.")
		return SecurityAuditDispositionQueuedSourceReReview, reasons, followUps
	}
}

func hasArtifactSecurityFinding(artifacts []CatalogSecurityArtifactAuditRow) bool {
	for _, artifact := range artifacts {
		switch artifact.Status {
		case SecurityAuditArtifactMissingSecurityMetadata, SecurityAuditArtifactMissingSecuritySchemes, SecurityAuditArtifactMissingSecurityRequirements, SecurityAuditArtifactUndeclaredSecuritySchemes:
			return true
		}
	}
	return false
}

func buildCatalogSecurityAuditSummary(rows []CatalogSecurityAuditRow) CatalogSecurityAuditSummary {
	dispositions := map[string][]string{}
	authStatuses := map[string][]string{}
	protocols := map[string][]string{}
	artifactStatuses := map[string][]string{}
	for _, row := range rows {
		dispositions[row.Disposition] = append(dispositions[row.Disposition], row.ProviderID)
		authStatuses[string(row.AuthStatus)] = append(authStatuses[string(row.AuthStatus)], row.ProviderID)
		protocols[string(row.Protocol)] = append(protocols[string(row.Protocol)], row.ProviderID)
		for _, artifact := range row.ArtifactSecurity {
			key := artifact.ProviderID + "/" + artifact.SpecRefID
			artifactStatuses[artifact.Status] = append(artifactStatuses[artifact.Status], key)
		}
	}
	return CatalogSecurityAuditSummary{
		ProviderCount: len(rows),
		Dispositions:  auditCounts(dispositions),
		AuthStatuses:  auditCounts(authStatuses),
		Protocols:     auditCounts(protocols),
		Artifacts:     artifactAuditCounts(artifactStatuses),
	}
}

func auditCounts(values map[string][]string) []CatalogSecurityAuditCount {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]CatalogSecurityAuditCount, 0, len(keys))
	for _, key := range keys {
		ids := sortedUniqueStrings(values[key])
		out = append(out, CatalogSecurityAuditCount{Name: key, Count: len(ids), ProviderIDs: ids})
	}
	return out
}

func artifactAuditCounts(values map[string][]string) []CatalogSecurityArtifactCount {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]CatalogSecurityArtifactCount, 0, len(keys))
	for _, key := range keys {
		ids := sortedUniqueStrings(values[key])
		out = append(out, CatalogSecurityArtifactCount{Status: key, Count: len(ids), SpecRefIDs: ids})
	}
	return out
}
