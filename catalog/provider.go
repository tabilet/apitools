package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SpecAvailability records whether an official spec source is known for a
// durable provider entry.
type SpecAvailability string

const (
	SpecAvailabilityKnown             SpecAvailability = "known"
	SpecAvailabilityNeedsVerification SpecAvailability = "needs-verification"
	SpecAvailabilityUnavailable       SpecAvailability = "unavailable"
	SpecAvailabilityUnknown           SpecAvailability = "unknown"
)

// ProviderReviewState records whether a provider entry is durable catalog
// metadata or still only a candidate.
type ProviderReviewState string

const (
	ProviderReviewedCatalogEntry ProviderReviewState = "reviewed-catalog-entry"
	ProviderCandidateOnly        ProviderReviewState = "candidate-only"
)

// SpecKind identifies the machine-readable or documentation format of a spec
// reference.
type SpecKind string

const (
	SpecKindOpenAPI         SpecKind = "openapi"
	SpecKindOpenAPIIndex    SpecKind = "openapi-index"
	SpecKindGoogleDiscovery SpecKind = "google-discovery"
	SpecKindHumanDocs       SpecKind = "human-docs"
)

// SourceAuthority describes who publishes a spec reference.
type SourceAuthority string

const (
	SourceAuthorityOfficialProvider SourceAuthority = "official-provider"
	SourceAuthorityOfficialGitHub   SourceAuthority = "official-github"
	SourceAuthorityOfficialDocs     SourceAuthority = "official-docs"
	SourceAuthorityPublicCatalog    SourceAuthority = "public-catalog"
)

// SpecReference records a source for provider API documentation or
// machine-readable specs. It is metadata only; callers must still validate
// referenced documents before importing or summarizing them.
type SpecReference struct {
	ID              string          `json:"id"`
	Kind            SpecKind        `json:"kind"`
	URL             string          `json:"url"`
	SourceAuthority SourceAuthority `json:"source_authority"`
	Version         string          `json:"version,omitempty"`
	VerifiedAt      string          `json:"verified_at,omitempty"`
	Revision        string          `json:"revision,omitempty"`
	LicenseNote     string          `json:"license_note,omitempty"`
	SourceNote      string          `json:"source_note,omitempty"`
}

// Provider is a durable built-in provider catalog entry.
type Provider struct {
	ID                              string              `json:"id"`
	DisplayName                     string              `json:"display_name"`
	Aliases                         []string            `json:"aliases,omitempty"`
	Category                        string              `json:"category,omitempty"`
	WorkflowRelevance               string              `json:"workflow_relevance,omitempty"`
	SourceHints                     []string            `json:"source_hints,omitempty"`
	ReviewState                     ProviderReviewState `json:"review_state"`
	CandidateID                     string              `json:"candidate_id,omitempty"`
	OfficialOpenAPIAvailability     SpecAvailability    `json:"official_openapi_availability"`
	OfficialMachineSpecAvailability SpecAvailability    `json:"official_machine_spec_availability"`
	UserOpenAPINeed                 UserOpenAPINeed     `json:"user_openapi_need"`
	SpecReferences                  []SpecReference     `json:"spec_references,omitempty"`
	Quirks                          []string            `json:"quirks,omitempty"`
}

// BuiltInProviders returns built-in provider catalog entries in deterministic
// order. Callers receive independent copies.
func BuiltInProviders() []Provider {
	out := cloneProviders(builtInProviders)
	sortProviders(out)
	return out
}

// FindBuiltInProvider returns a built-in provider by ID, display name, or
// alias. Matching is case-insensitive and whitespace-insensitive.
func FindBuiltInProvider(key string) (Provider, bool) {
	return BuiltInCatalog().FindProvider(key)
}

// ValidateProviders validates provider catalog records for deterministic
// loading and review.
func ValidateProviders(providers []Provider) error {
	ids := map[string]struct{}{}
	keys := map[string]string{}
	for i, provider := range providers {
		if err := validateProvider(provider); err != nil {
			return fmt.Errorf("provider[%d]: %w", i, err)
		}
		if _, exists := ids[provider.ID]; exists {
			return fmt.Errorf("provider %q: duplicate id", provider.ID)
		}
		ids[provider.ID] = struct{}{}
		for _, key := range provider.lookupKeys() {
			if previous, exists := keys[key]; exists {
				return fmt.Errorf("provider %q: lookup key %q conflicts with provider %q", provider.ID, key, previous)
			}
			keys[key] = provider.ID
		}
	}
	return nil
}

// ProviderIDs returns sorted provider IDs.
func ProviderIDs(providers []Provider) []string {
	out := make([]string, 0, len(providers))
	for _, provider := range providers {
		if strings.TrimSpace(provider.ID) != "" {
			out = append(out, provider.ID)
		}
	}
	sort.Strings(out)
	return out
}

// SpecReferencesByKind returns spec references matching kind in deterministic
// order.
func (p Provider) SpecReferencesByKind(kind SpecKind) []SpecReference {
	var out []SpecReference
	for _, ref := range p.SpecReferences {
		if ref.Kind == kind {
			out = append(out, ref)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func validateProvider(provider Provider) error {
	if strings.TrimSpace(provider.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(provider.ID) {
		return fmt.Errorf("invalid id %q", provider.ID)
	}
	if strings.TrimSpace(provider.DisplayName) == "" {
		return fmt.Errorf("provider %q: missing display name", provider.ID)
	}
	if !validProviderReviewState(provider.ReviewState) {
		return fmt.Errorf("provider %q: invalid review state %q", provider.ID, provider.ReviewState)
	}
	if provider.ReviewState == ProviderReviewedCatalogEntry && strings.TrimSpace(provider.CandidateID) == "" {
		return fmt.Errorf("provider %q: reviewed catalog entry requires candidate id", provider.ID)
	}
	if strings.TrimSpace(provider.CandidateID) != "" && !validID(provider.CandidateID) {
		return fmt.Errorf("provider %q: invalid candidate id %q", provider.ID, provider.CandidateID)
	}
	if !validSpecAvailability(provider.OfficialOpenAPIAvailability) {
		return fmt.Errorf("provider %q: invalid OpenAPI availability %q", provider.ID, provider.OfficialOpenAPIAvailability)
	}
	if !validSpecAvailability(provider.OfficialMachineSpecAvailability) {
		return fmt.Errorf("provider %q: invalid machine spec availability %q", provider.ID, provider.OfficialMachineSpecAvailability)
	}
	if !validUserOpenAPINeed(provider.UserOpenAPINeed) {
		return fmt.Errorf("provider %q: invalid user OpenAPI need %q", provider.ID, provider.UserOpenAPINeed)
	}
	seenRefs := map[string]struct{}{}
	for i, ref := range provider.SpecReferences {
		if err := validateSpecReference(provider.ID, ref); err != nil {
			return fmt.Errorf("provider %q spec[%d]: %w", provider.ID, i, err)
		}
		if _, exists := seenRefs[ref.ID]; exists {
			return fmt.Errorf("provider %q spec[%d]: duplicate spec reference id %q", provider.ID, i, ref.ID)
		}
		seenRefs[ref.ID] = struct{}{}
	}
	if provider.OfficialOpenAPIAvailability == SpecAvailabilityKnown && !provider.hasSpecKind(SpecKindOpenAPI, SpecKindOpenAPIIndex) {
		return fmt.Errorf("provider %q: known OpenAPI availability requires an OpenAPI spec reference", provider.ID)
	}
	if provider.OfficialMachineSpecAvailability == SpecAvailabilityKnown && !provider.hasSpecKind(SpecKindGoogleDiscovery) {
		return fmt.Errorf("provider %q: known machine spec availability requires a machine spec reference", provider.ID)
	}
	return nil
}

func validateSpecReference(providerID string, ref SpecReference) error {
	if strings.TrimSpace(ref.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(ref.ID) {
		return fmt.Errorf("invalid id %q", ref.ID)
	}
	if !validSpecKind(ref.Kind) {
		return fmt.Errorf("invalid kind %q", ref.Kind)
	}
	if strings.TrimSpace(ref.URL) == "" {
		return fmt.Errorf("missing url")
	}
	if !strings.HasPrefix(strings.TrimSpace(ref.URL), "https://") {
		return fmt.Errorf("url for %q must be https", providerID)
	}
	if !validSourceAuthority(ref.SourceAuthority) {
		return fmt.Errorf("invalid source authority %q", ref.SourceAuthority)
	}
	if strings.TrimSpace(ref.SourceNote) == "" {
		return fmt.Errorf("missing source note")
	}
	return nil
}

func (p Provider) matches(normalized string) bool {
	for _, key := range p.lookupKeys() {
		if key == normalized {
			return true
		}
	}
	return false
}

func (p Provider) lookupKeys() []string {
	values := append([]string{p.ID, p.DisplayName}, p.Aliases...)
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

func (p Provider) hasSpecKind(kinds ...SpecKind) bool {
	for _, ref := range p.SpecReferences {
		for _, kind := range kinds {
			if ref.Kind == kind {
				return true
			}
		}
	}
	return false
}

func validProviderReviewState(value ProviderReviewState) bool {
	switch value {
	case ProviderReviewedCatalogEntry, ProviderCandidateOnly:
		return true
	default:
		return false
	}
}

func validSpecAvailability(value SpecAvailability) bool {
	switch value {
	case SpecAvailabilityKnown, SpecAvailabilityNeedsVerification, SpecAvailabilityUnavailable, SpecAvailabilityUnknown:
		return true
	default:
		return false
	}
}

func validSpecKind(value SpecKind) bool {
	switch value {
	case SpecKindOpenAPI, SpecKindOpenAPIIndex, SpecKindGoogleDiscovery, SpecKindHumanDocs:
		return true
	default:
		return false
	}
}

func validSourceAuthority(value SourceAuthority) bool {
	switch value {
	case SourceAuthorityOfficialProvider, SourceAuthorityOfficialGitHub, SourceAuthorityOfficialDocs, SourceAuthorityPublicCatalog:
		return true
	default:
		return false
	}
}

func sortProviders(providers []Provider) {
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
}

func cloneProviders(in []Provider) []Provider {
	out := make([]Provider, len(in))
	for i, provider := range in {
		out[i] = cloneProvider(provider)
	}
	return out
}

func cloneProvider(provider Provider) Provider {
	provider.Aliases = append([]string(nil), provider.Aliases...)
	provider.SourceHints = append([]string(nil), provider.SourceHints...)
	provider.SpecReferences = append([]SpecReference(nil), provider.SpecReferences...)
	provider.Quirks = append([]string(nil), provider.Quirks...)
	return provider
}
