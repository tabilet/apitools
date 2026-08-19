package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const CatalogBundleVersion = "apitools.catalog.v1"

// CatalogBundle is the reviewed, metadata-only source representation used to
// generate the built-in catalog indexes. It never contains credentials or
// executable provider behavior.
type CatalogBundle struct {
	Version                 string                   `json:"version"`
	Candidates              []Candidate              `json:"candidates"`
	Providers               []Provider               `json:"providers"`
	SecurityClassifications []SecurityClassification `json:"security_classifications"`
	SecurityOverlays        []SecurityOverlay        `json:"security_overlays"`
}

// ParseCatalogBundle strictly decodes and validates one reviewed catalog JSON
// bundle. Unknown or trailing content fails closed.
func ParseCatalogBundle(content []byte) (CatalogBundle, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var bundle CatalogBundle
	if err := decoder.Decode(&bundle); err != nil {
		return CatalogBundle{}, fmt.Errorf("decode catalog bundle: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return CatalogBundle{}, fmt.Errorf("decode catalog bundle: trailing JSON value")
		}
		return CatalogBundle{}, fmt.Errorf("decode catalog bundle trailing content: %w", err)
	}
	if err := ValidateCatalogBundle(bundle); err != nil {
		return CatalogBundle{}, err
	}
	return bundle, nil
}

// ValidateCatalogBundle checks all record and cross-record invariants required
// before catalog data can be generated into the built-in indexes.
func ValidateCatalogBundle(bundle CatalogBundle) error {
	if bundle.Version != CatalogBundleVersion {
		return fmt.Errorf("catalog bundle version %q, want %q", bundle.Version, CatalogBundleVersion)
	}
	if len(bundle.Candidates) == 0 || len(bundle.Providers) == 0 {
		return fmt.Errorf("catalog bundle requires candidates and providers")
	}
	if err := ValidateCandidates(bundle.Candidates); err != nil {
		return err
	}
	if err := ValidateProviders(bundle.Providers); err != nil {
		return err
	}
	if err := ValidateSecurityClassifications(bundle.SecurityClassifications, bundle.Providers); err != nil {
		return err
	}
	if err := ValidateSecurityOverlays(bundle.SecurityOverlays, bundle.Providers); err != nil {
		return err
	}
	if err := validateCatalogBundleOrder(bundle); err != nil {
		return err
	}
	candidateIDs := make(map[string]struct{}, len(bundle.Candidates))
	for _, candidate := range bundle.Candidates {
		candidateIDs[candidate.ID] = struct{}{}
	}
	for _, provider := range bundle.Providers {
		if _, ok := candidateIDs[provider.CandidateID]; !ok {
			return fmt.Errorf("provider %q references unknown candidate %q", provider.ID, provider.CandidateID)
		}
	}
	report, err := BuildSecurityReport(bundle.Providers, bundle.SecurityOverlays, bundle.SecurityClassifications)
	if err != nil {
		return err
	}
	for _, provider := range report.Providers {
		if provider.Status == AuthStatusConflict {
			return fmt.Errorf("provider %q has conflicting security dispositions", provider.ProviderID)
		}
	}
	return nil
}

// ValidateSecurityClassifications validates source-backed disposition records
// and permits at most one record for each provider/spec scope.
func ValidateSecurityClassifications(classifications []SecurityClassification, providers []Provider) error {
	providerByID := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	seen := map[string]struct{}{}
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providerByID); err != nil {
			return fmt.Errorf("security classification[%d]: %w", i, err)
		}
		key := classification.ProviderID + "\x00" + strings.TrimSpace(classification.SpecRefID)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("security classification[%d]: duplicate provider/spec scope %q/%q", i, classification.ProviderID, classification.SpecRefID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateCatalogBundleOrder(bundle CatalogBundle) error {
	if !sort.SliceIsSorted(bundle.Candidates, func(i, j int) bool { return bundle.Candidates[i].ID < bundle.Candidates[j].ID }) {
		return fmt.Errorf("catalog candidates must be sorted by id")
	}
	if !sort.SliceIsSorted(bundle.Providers, func(i, j int) bool { return bundle.Providers[i].ID < bundle.Providers[j].ID }) {
		return fmt.Errorf("catalog providers must be sorted by id")
	}
	if !sort.SliceIsSorted(bundle.SecurityClassifications, func(i, j int) bool {
		left, right := bundle.SecurityClassifications[i], bundle.SecurityClassifications[j]
		if left.ProviderID == right.ProviderID {
			return left.SpecRefID < right.SpecRefID
		}
		return left.ProviderID < right.ProviderID
	}) {
		return fmt.Errorf("catalog security classifications must be sorted by provider and spec ref")
	}
	if !sort.SliceIsSorted(bundle.SecurityOverlays, func(i, j int) bool {
		left, right := bundle.SecurityOverlays[i], bundle.SecurityOverlays[j]
		if left.ProviderID == right.ProviderID {
			return left.ID < right.ID
		}
		return left.ProviderID < right.ProviderID
	}) {
		return fmt.Errorf("catalog security overlays must be sorted by provider and id")
	}
	return nil
}
