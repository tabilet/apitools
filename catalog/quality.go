package catalog

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const DefaultStaleVerificationDays = 365

// CatalogQualitySeverity is the quality finding severity used by offline
// catalog checks.
type CatalogQualitySeverity string

const (
	CatalogQualityError   CatalogQualitySeverity = "error"
	CatalogQualityWarning CatalogQualitySeverity = "warning"
)

// CatalogQualityOptions controls deterministic offline catalog checks.
type CatalogQualityOptions struct {
	Catalog                   Catalog
	Candidates                []Candidate
	SecurityClassifications   []SecurityClassification
	AsOf                      time.Time
	StaleVerificationDays     int
	KnownOperationsByProvider map[string][]OperationMatch
}

// CatalogQualityReport records deterministic offline quality findings.
type CatalogQualityReport struct {
	Findings []CatalogQualityFinding `json:"findings,omitempty"`
}

// CatalogQualityFinding records one catalog metadata quality issue.
type CatalogQualityFinding struct {
	Severity    CatalogQualitySeverity `json:"severity"`
	Code        string                 `json:"code"`
	ProviderID  string                 `json:"provider_id,omitempty"`
	CandidateID string                 `json:"candidate_id,omitempty"`
	OverlayID   string                 `json:"overlay_id,omitempty"`
	SpecRefID   string                 `json:"spec_ref_id,omitempty"`
	Field       string                 `json:"field,omitempty"`
	Message     string                 `json:"message"`
}

// BuiltInCatalogQualityReport checks built-in providers, candidates,
// classifications, and overlays together without network access.
func BuiltInCatalogQualityReport(options CatalogQualityOptions) CatalogQualityReport {
	options.Catalog = BuiltInCatalog()
	options.Candidates = BuiltInCandidates()
	options.SecurityClassifications = BuiltInSecurityClassifications()
	return BuildCatalogQualityReport(options)
}

// BuildCatalogQualityReport runs deterministic offline catalog checks. It does
// not fetch URLs, download documents, execute API operations, or inspect
// credentials.
func BuildCatalogQualityReport(options CatalogQualityOptions) CatalogQualityReport {
	options = normalizeQualityOptions(options)
	providers := cloneProviders(options.Catalog.Providers)
	overlays := cloneSecurityOverlays(options.Catalog.SecurityOverlays)
	candidates := cloneCandidates(options.Candidates)
	classifications := cloneSecurityClassifications(options.SecurityClassifications)

	var findings []CatalogQualityFinding
	add := func(finding CatalogQualityFinding) {
		if finding.Severity == "" {
			finding.Severity = CatalogQualityError
		}
		findings = append(findings, finding)
	}

	if err := ValidateProviders(providers); err != nil {
		add(CatalogQualityFinding{
			Severity: CatalogQualityError,
			Code:     "invalid-providers",
			Field:    "providers",
			Message:  err.Error(),
		})
	}
	if options.Candidates != nil {
		if err := ValidateCandidates(candidates); err != nil {
			add(CatalogQualityFinding{
				Severity: CatalogQualityError,
				Code:     "invalid-candidates",
				Field:    "candidates",
				Message:  err.Error(),
			})
		}
	}
	if err := ValidateSecurityOverlays(overlays, providers); err != nil {
		add(CatalogQualityFinding{
			Severity: CatalogQualityError,
			Code:     "invalid-security-overlays",
			Field:    "security_overlays",
			Message:  err.Error(),
		})
	}

	securityReport, err := BuildSecurityReport(providers, overlays, classifications)
	if err != nil {
		add(CatalogQualityFinding{
			Severity: CatalogQualityError,
			Code:     "invalid-security-classifications",
			Field:    "security_classifications",
			Message:  err.Error(),
		})
	}
	securityByProvider := map[string]ProviderSecurityReport{}
	for _, row := range securityReport.Providers {
		securityByProvider[row.ProviderID] = row
	}

	for _, provider := range sortedProviders(providers) {
		checkProviderQuality(provider, options, securityByProvider, add)
	}
	checkOperationOverlayQuality(overlays, options.KnownOperationsByProvider, add)
	if options.Candidates != nil {
		checkProviderCandidateLookupConflicts(providers, candidates, add)
	}

	sortQualityFindings(findings)
	return CatalogQualityReport{Findings: findings}
}

// HasErrors reports whether the quality report contains error-level findings.
func (r CatalogQualityReport) HasErrors() bool {
	return r.ErrorCount() > 0
}

// ErrorCount returns the number of error-level findings.
func (r CatalogQualityReport) ErrorCount() int {
	return r.CountSeverity(CatalogQualityError)
}

// WarningCount returns the number of warning-level findings.
func (r CatalogQualityReport) WarningCount() int {
	return r.CountSeverity(CatalogQualityWarning)
}

// CountSeverity returns the number of findings with a severity.
func (r CatalogQualityReport) CountSeverity(severity CatalogQualitySeverity) int {
	var count int
	for _, finding := range r.Findings {
		if finding.Severity == severity {
			count++
		}
	}
	return count
}

func normalizeQualityOptions(options CatalogQualityOptions) CatalogQualityOptions {
	if options.AsOf.IsZero() {
		options.AsOf = time.Now().UTC()
	}
	if options.StaleVerificationDays <= 0 {
		options.StaleVerificationDays = DefaultStaleVerificationDays
	}
	return options
}

func checkProviderQuality(provider Provider, options CatalogQualityOptions, securityByProvider map[string]ProviderSecurityReport, add func(CatalogQualityFinding)) {
	if len(nonEmptyStrings(provider.SourceHints)) == 0 {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityWarning,
			Code:       "missing-provider-source-hints",
			ProviderID: provider.ID,
			Field:      "source_hints",
			Message:    "provider is missing source hint provenance",
		})
	}
	if len(provider.SpecReferences) == 0 {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "missing-spec-reference",
			ProviderID: provider.ID,
			Field:      "spec_references",
			Message:    "provider has no spec or documentation reference",
		})
	}
	for _, ref := range provider.SpecReferences {
		checkSpecReferenceQuality(provider.ID, ref, options, add)
	}
	checkUserOpenAPINeedQuality(provider, add)
	report, ok := securityByProvider[provider.ID]
	if !ok || report.Status == "" || report.Status == AuthStatusUnknown {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "missing-security-status",
			ProviderID: provider.ID,
			Field:      "security_status",
			Message:    "provider has no source-backed security classification or overlay status",
		})
	}
	if ok && report.Status == AuthStatusConflict {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "conflicting-security-status",
			ProviderID: provider.ID,
			Field:      "security_status",
			Message:    "provider has conflicting security overlays in the same provider/spec scope",
		})
	}
}

func checkSpecReferenceQuality(providerID string, ref SpecReference, options CatalogQualityOptions, add func(CatalogQualityFinding)) {
	if strings.TrimSpace(ref.SourceNote) == "" {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "missing-spec-source-note",
			ProviderID: providerID,
			SpecRefID:  ref.ID,
			Field:      "spec_references.source_note",
			Message:    "spec reference is missing source note provenance",
		})
	}
	if strings.TrimSpace(ref.VerifiedAt) == "" {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityWarning,
			Code:       "missing-verification-date",
			ProviderID: providerID,
			SpecRefID:  ref.ID,
			Field:      "spec_references.verified_at",
			Message:    "spec reference is missing verification date",
		})
		return
	}
	verifiedAt, err := time.Parse("2006-01-02", strings.TrimSpace(ref.VerifiedAt))
	if err != nil {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "invalid-verification-date",
			ProviderID: providerID,
			SpecRefID:  ref.ID,
			Field:      "spec_references.verified_at",
			Message:    fmt.Sprintf("spec reference verification date %q must use YYYY-MM-DD", ref.VerifiedAt),
		})
		return
	}
	staleAfter := time.Duration(options.StaleVerificationDays) * 24 * time.Hour
	if options.AsOf.Sub(verifiedAt) > staleAfter {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityWarning,
			Code:       "stale-verification-date",
			ProviderID: providerID,
			SpecRefID:  ref.ID,
			Field:      "spec_references.verified_at",
			Message:    fmt.Sprintf("spec reference was last verified on %s, more than %d days before %s", ref.VerifiedAt, options.StaleVerificationDays, options.AsOf.Format("2006-01-02")),
		})
	}
}

func checkUserOpenAPINeedQuality(provider Provider, add func(CatalogQualityFinding)) {
	hasOpenAPI := provider.hasSpecKind(SpecKindOpenAPI, SpecKindOpenAPIIndex)
	hasMachine := provider.hasMachineSpecKind()
	if !hasOpenAPI && !hasMachine && provider.UserOpenAPINeed == UserOpenAPINeedNotExpected {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityError,
			Code:       "inconsistent-user-openapi-need",
			ProviderID: provider.ID,
			Field:      "user_openapi_need",
			Message:    "provider has no known machine-readable spec but user OpenAPI need is not-expected",
		})
	}
	if provider.OfficialOpenAPIAvailability == SpecAvailabilityKnown && provider.UserOpenAPINeed == UserOpenAPINeedLikely {
		add(CatalogQualityFinding{
			Severity:   CatalogQualityWarning,
			Code:       "unexpected-user-openapi-need",
			ProviderID: provider.ID,
			Field:      "user_openapi_need",
			Message:    "provider has known official OpenAPI but user OpenAPI need is likely",
		})
	}
}

func checkOperationOverlayQuality(overlays []SecurityOverlay, known map[string][]OperationMatch, add func(CatalogQualityFinding)) {
	if len(known) == 0 {
		return
	}
	for _, overlay := range overlays {
		operations, ok := known[overlay.ProviderID]
		if !ok {
			continue
		}
		for _, operation := range overlay.OperationSecurity {
			if operationMatchResolved(operation.Match, operations) {
				continue
			}
			add(CatalogQualityFinding{
				Severity:   CatalogQualityError,
				Code:       "unresolved-overlay-operation",
				ProviderID: overlay.ProviderID,
				OverlayID:  overlay.ID,
				SpecRefID:  overlay.SpecRefID,
				Field:      "security_overlays.operation_security.match",
				Message:    "overlay operation security target was not resolved against known operations",
			})
		}
	}
}

func checkProviderCandidateLookupConflicts(providers []Provider, candidates []Candidate, add func(CatalogQualityFinding)) {
	lookupProvider := map[string]string{}
	for _, provider := range providers {
		for _, key := range provider.lookupKeys() {
			lookupProvider[key] = provider.ID
		}
	}
	for _, candidate := range candidates {
		for _, key := range candidate.lookupKeys() {
			providerID, ok := lookupProvider[key]
			if !ok || providerID == candidate.ID {
				continue
			}
			add(CatalogQualityFinding{
				Severity:    CatalogQualityWarning,
				Code:        "provider-candidate-lookup-conflict",
				ProviderID:  providerID,
				CandidateID: candidate.ID,
				Field:       "aliases",
				Message:     fmt.Sprintf("candidate lookup key %q overlaps provider %q", key, providerID),
			})
		}
	}
}

func sortedProviders(providers []Provider) []Provider {
	out := cloneProviders(providers)
	sortProviders(out)
	return out
}

func nonEmptyStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func sortQualityFindings(findings []CatalogQualityFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]
		for _, less := range []int{
			strings.Compare(string(left.Severity), string(right.Severity)),
			strings.Compare(left.ProviderID, right.ProviderID),
			strings.Compare(left.CandidateID, right.CandidateID),
			strings.Compare(left.OverlayID, right.OverlayID),
			strings.Compare(left.SpecRefID, right.SpecRefID),
			strings.Compare(left.Code, right.Code),
			strings.Compare(left.Field, right.Field),
			strings.Compare(left.Message, right.Message),
		} {
			if less < 0 {
				return true
			}
			if less > 0 {
				return false
			}
		}
		return false
	})
}
