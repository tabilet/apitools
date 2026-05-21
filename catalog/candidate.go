package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// CandidateEvidenceSource identifies where a candidate inventory signal came
// from.
type CandidateEvidenceSource string

const (
	EvidenceLocalOpenAPIFixture CandidateEvidenceSource = "local-openapi-fixture"
	EvidencePublicCatalog       CandidateEvidenceSource = "public-catalog"
	EvidenceOfficialDocs        CandidateEvidenceSource = "official-docs"
	EvidenceUserReport          CandidateEvidenceSource = "user-report"
)

// CandidateEvidenceUse describes how evidence may be used during inventory
// review.
type CandidateEvidenceUse string

const (
	EvidenceUseSeedFixture CandidateEvidenceUse = "seed-fixture"
	EvidenceUsePriority    CandidateEvidenceUse = "priority-only"
	EvidenceUseReview      CandidateEvidenceUse = "review"
)

// SpecStatus records whether a candidate has an official machine-readable API
// spec source. Candidate-stage statuses are deliberately conservative.
type SpecStatus string

const (
	SpecStatusKnown             SpecStatus = "known"
	SpecStatusNeedsVerification SpecStatus = "needs-verification"
	SpecStatusUnavailable       SpecStatus = "unavailable"
	SpecStatusUnknown           SpecStatus = "unknown"
)

// UserOpenAPINeed records whether a later catalog entry is expected to need a
// user-provided OpenAPI document.
type UserOpenAPINeed string

const (
	UserOpenAPINeedLikely      UserOpenAPINeed = "likely"
	UserOpenAPINeedPossible    UserOpenAPINeed = "possible"
	UserOpenAPINeedNotExpected UserOpenAPINeed = "not-expected"
	UserOpenAPINeedUnknown     UserOpenAPINeed = "unknown"
)

// AuthSecurityReviewState records candidate-stage auth/security review status.
type AuthSecurityReviewState string

const (
	AuthSecurityNotReviewed            AuthSecurityReviewState = "not-reviewed"
	AuthSecurityLikelyIncomplete       AuthSecurityReviewState = "likely-incomplete"
	AuthSecurityLikelyComplete         AuthSecurityReviewState = "likely-complete"
	AuthSecurityIntentionallyAnonymous AuthSecurityReviewState = "intentionally-anonymous"
	AuthSecurityUnknown                AuthSecurityReviewState = "unknown"
)

// CandidateEvidence is one provenance note for a candidate service.
type CandidateEvidence struct {
	Source CandidateEvidenceSource `json:"source"`
	Use    CandidateEvidenceUse    `json:"use"`
	Ref    string                  `json:"ref"`
	Note   string                  `json:"note,omitempty"`
}

// Candidate is a review input for future catalog entries. It intentionally
// stops short of declaring a durable provider catalog contract.
type Candidate struct {
	ID                        string                  `json:"id"`
	DisplayName               string                  `json:"display_name"`
	Aliases                   []string                `json:"aliases,omitempty"`
	Category                  string                  `json:"category,omitempty"`
	WorkflowRelevance         string                  `json:"workflow_relevance,omitempty"`
	Evidence                  []CandidateEvidence     `json:"evidence,omitempty"`
	LocalOpenAPIFixture       string                  `json:"local_openapi_fixture,omitempty"`
	OfficialOpenAPIStatus     SpecStatus              `json:"official_openapi_status"`
	OfficialMachineSpecStatus SpecStatus              `json:"official_machine_spec_status"`
	OfficialMachineSpecKind   string                  `json:"official_machine_spec_kind,omitempty"`
	UserOpenAPINeed           UserOpenAPINeed         `json:"user_openapi_need"`
	AuthSecurityReview        AuthSecurityReviewState `json:"auth_security_review"`
}

// BuiltInCandidates returns the built-in candidate service inventory in a
// deterministic order. Callers receive independent copies.
func BuiltInCandidates() []Candidate {
	out := cloneCandidates(builtInCandidates)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// FindBuiltInCandidate returns a built-in candidate by ID, display name, or
// alias. Matching is case-insensitive and whitespace-insensitive.
func FindBuiltInCandidate(key string) (Candidate, bool) {
	normalized := normalizeKey(key)
	if normalized == "" {
		return Candidate{}, false
	}
	for _, candidate := range BuiltInCandidates() {
		if candidate.matches(normalized) {
			return candidate, true
		}
	}
	return Candidate{}, false
}

// ValidateCandidates validates candidate inventory records for deterministic
// loading and review.
func ValidateCandidates(candidates []Candidate) error {
	ids := map[string]struct{}{}
	keys := map[string]string{}
	for i, candidate := range candidates {
		if err := validateCandidate(candidate); err != nil {
			return fmt.Errorf("candidate[%d]: %w", i, err)
		}
		if _, exists := ids[candidate.ID]; exists {
			return fmt.Errorf("candidate %q: duplicate id", candidate.ID)
		}
		ids[candidate.ID] = struct{}{}
		for _, key := range candidate.lookupKeys() {
			if previous, exists := keys[key]; exists {
				return fmt.Errorf("candidate %q: lookup key %q conflicts with candidate %q", candidate.ID, key, previous)
			}
			keys[key] = candidate.ID
		}
	}
	return nil
}

// CandidateIDs returns sorted candidate IDs.
func CandidateIDs(candidates []Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.ID) != "" {
			out = append(out, candidate.ID)
		}
	}
	sort.Strings(out)
	return out
}

// HasEvidence reports whether a candidate has evidence from source.
func (c Candidate) HasEvidence(source CandidateEvidenceSource) bool {
	for _, evidence := range c.Evidence {
		if evidence.Source == source {
			return true
		}
	}
	return false
}

func validateCandidate(candidate Candidate) error {
	if strings.TrimSpace(candidate.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(candidate.ID) {
		return fmt.Errorf("invalid id %q", candidate.ID)
	}
	if strings.TrimSpace(candidate.DisplayName) == "" {
		return fmt.Errorf("candidate %q: missing display name", candidate.ID)
	}
	if !validSpecStatus(candidate.OfficialOpenAPIStatus) {
		return fmt.Errorf("candidate %q: invalid official OpenAPI status %q", candidate.ID, candidate.OfficialOpenAPIStatus)
	}
	if !validSpecStatus(candidate.OfficialMachineSpecStatus) {
		return fmt.Errorf("candidate %q: invalid official machine spec status %q", candidate.ID, candidate.OfficialMachineSpecStatus)
	}
	if !validUserOpenAPINeed(candidate.UserOpenAPINeed) {
		return fmt.Errorf("candidate %q: invalid user OpenAPI need %q", candidate.ID, candidate.UserOpenAPINeed)
	}
	if !validAuthSecurityReviewState(candidate.AuthSecurityReview) {
		return fmt.Errorf("candidate %q: invalid auth/security review state %q", candidate.ID, candidate.AuthSecurityReview)
	}
	if fixture := strings.TrimSpace(candidate.LocalOpenAPIFixture); fixture != "" && !candidate.hasEvidenceRef(EvidenceLocalOpenAPIFixture, fixture) {
		return fmt.Errorf("candidate %q: local fixture path requires matching local fixture evidence", candidate.ID)
	}
	if strings.TrimSpace(candidate.OfficialMachineSpecKind) != "" && candidate.OfficialMachineSpecStatus == SpecStatusUnknown {
		return fmt.Errorf("candidate %q: machine spec kind requires a non-unknown machine spec status", candidate.ID)
	}
	seenEvidence := map[string]struct{}{}
	for i, evidence := range candidate.Evidence {
		if err := validateEvidence(candidate.ID, evidence); err != nil {
			return fmt.Errorf("candidate %q evidence[%d]: %w", candidate.ID, i, err)
		}
		key := string(evidence.Source) + "\x00" + evidence.Ref
		if _, exists := seenEvidence[key]; exists {
			return fmt.Errorf("candidate %q evidence[%d]: duplicate source/ref", candidate.ID, i)
		}
		seenEvidence[key] = struct{}{}
	}
	return nil
}

func validateEvidence(candidateID string, evidence CandidateEvidence) error {
	if !validEvidenceSource(evidence.Source) {
		return fmt.Errorf("invalid source %q", evidence.Source)
	}
	if !validEvidenceUse(evidence.Use) {
		return fmt.Errorf("invalid use %q", evidence.Use)
	}
	if strings.TrimSpace(evidence.Ref) == "" {
		return fmt.Errorf("missing ref")
	}
	if strings.TrimSpace(evidence.Note) == "" {
		return fmt.Errorf("missing note")
	}
	return nil
}

func (c Candidate) matches(normalized string) bool {
	for _, key := range c.lookupKeys() {
		if key == normalized {
			return true
		}
	}
	return false
}

func (c Candidate) lookupKeys() []string {
	values := append([]string{c.ID, c.DisplayName}, c.Aliases...)
	keys := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key := normalizeKey(value)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func (c Candidate) hasEvidenceRef(source CandidateEvidenceSource, ref string) bool {
	for _, evidence := range c.Evidence {
		if evidence.Source == source && strings.TrimSpace(evidence.Ref) == ref {
			return true
		}
	}
	return false
}

func normalizeKey(value string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	return strings.Join(fields, "-")
}

func validID(id string) bool {
	if id != strings.ToLower(id) {
		return false
	}
	for _, r := range id {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' {
			continue
		}
		return false
	}
	return !strings.HasPrefix(id, "-") && !strings.HasSuffix(id, "-") && !strings.Contains(id, "--")
}

func validEvidenceSource(value CandidateEvidenceSource) bool {
	switch value {
	case EvidenceLocalOpenAPIFixture, EvidencePublicCatalog, EvidenceOfficialDocs, EvidenceUserReport:
		return true
	default:
		return false
	}
}

func validEvidenceUse(value CandidateEvidenceUse) bool {
	switch value {
	case EvidenceUseSeedFixture, EvidenceUsePriority, EvidenceUseReview:
		return true
	default:
		return false
	}
}

func validSpecStatus(value SpecStatus) bool {
	switch value {
	case SpecStatusKnown, SpecStatusNeedsVerification, SpecStatusUnavailable, SpecStatusUnknown:
		return true
	default:
		return false
	}
}

func validUserOpenAPINeed(value UserOpenAPINeed) bool {
	switch value {
	case UserOpenAPINeedLikely, UserOpenAPINeedPossible, UserOpenAPINeedNotExpected, UserOpenAPINeedUnknown:
		return true
	default:
		return false
	}
}

func validAuthSecurityReviewState(value AuthSecurityReviewState) bool {
	switch value {
	case AuthSecurityNotReviewed, AuthSecurityLikelyIncomplete, AuthSecurityLikelyComplete, AuthSecurityIntentionallyAnonymous, AuthSecurityUnknown:
		return true
	default:
		return false
	}
}

func cloneCandidates(in []Candidate) []Candidate {
	out := make([]Candidate, len(in))
	for i, candidate := range in {
		out[i] = cloneCandidate(candidate)
	}
	return out
}

func cloneCandidate(candidate Candidate) Candidate {
	candidate.Aliases = append([]string(nil), candidate.Aliases...)
	candidate.Evidence = append([]CandidateEvidence(nil), candidate.Evidence...)
	return candidate
}
