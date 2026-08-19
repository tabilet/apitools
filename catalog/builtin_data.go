package catalog

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
)

//go:embed data/catalog.json
var embeddedCatalogJSON []byte

type generatedCatalogIndexes struct {
	candidates                map[string]int
	providers                 map[string]int
	classificationsByProvider map[string][]int
	overlaysByProvider        map[string][]int
}

var loadedBuiltInCatalog = mustLoadBuiltInCatalog(embeddedCatalogJSON, generatedCatalogIndexes{
	candidates:                generatedCandidateIndexByID,
	providers:                 generatedProviderIndexByID,
	classificationsByProvider: generatedSecurityClassificationIndexesByProvider,
	overlaysByProvider:        generatedSecurityOverlayIndexesByProvider,
})

var builtInCandidates = loadedBuiltInCatalog.Candidates
var builtInProviders = loadedBuiltInCatalog.Providers
var builtInSecurityClassifications = loadedBuiltInCatalog.SecurityClassifications
var builtInSecurityOverlays = loadedBuiltInCatalog.SecurityOverlays

var builtInCandidateLookupIndex = mustCandidateLookupIndex(builtInCandidates)
var builtInProviderLookupIndex = mustProviderLookupIndex(builtInProviders)

// CatalogManifest records the generated golden identity and record counts for
// the reviewed built-in catalog source bundle.
type CatalogManifest struct {
	Version                     string `json:"version"`
	CatalogVersion              string `json:"catalog_version"`
	CatalogSHA256               string `json:"catalog_sha256"`
	CatalogBytes                int    `json:"catalog_bytes"`
	CandidateCount              int    `json:"candidate_count"`
	ProviderCount               int    `json:"provider_count"`
	SpecReferenceCount          int    `json:"spec_reference_count"`
	SecurityClassificationCount int    `json:"security_classification_count"`
	SecurityOverlayCount        int    `json:"security_overlay_count"`
}

// BuiltInCatalogManifest returns the generated golden identity for the
// reviewed built-in catalog.
func BuiltInCatalogManifest() CatalogManifest {
	return CatalogManifest{
		Version:                     generatedCatalogManifestVersion,
		CatalogVersion:              CatalogBundleVersion,
		CatalogSHA256:               generatedCatalogSHA256,
		CatalogBytes:                generatedCatalogBytes,
		CandidateCount:              generatedCandidateCount,
		ProviderCount:               generatedProviderCount,
		SpecReferenceCount:          generatedSpecReferenceCount,
		SecurityClassificationCount: generatedSecurityClassificationCount,
		SecurityOverlayCount:        generatedSecurityOverlayCount,
	}
}

func mustLoadBuiltInCatalog(content []byte, indexes generatedCatalogIndexes) CatalogBundle {
	if generatedCatalogManifestVersion != "apitools.catalog-manifest.v1" {
		panic(fmt.Sprintf("unsupported generated catalog manifest version %q", generatedCatalogManifestVersion))
	}
	if len(content) != generatedCatalogBytes {
		panic(fmt.Sprintf("embedded catalog has %d bytes, want %d", len(content), generatedCatalogBytes))
	}
	digest := sha256.Sum256(content)
	if got := hex.EncodeToString(digest[:]); got != generatedCatalogSHA256 {
		panic(fmt.Sprintf("embedded catalog SHA256 is %s, want %s", got, generatedCatalogSHA256))
	}
	bundle, err := ParseCatalogBundle(content)
	if err != nil {
		panic(err)
	}
	if len(bundle.Candidates) != generatedCandidateCount || len(bundle.Providers) != generatedProviderCount || len(bundle.SecurityClassifications) != generatedSecurityClassificationCount || len(bundle.SecurityOverlays) != generatedSecurityOverlayCount {
		panic("embedded catalog counts do not match generated manifest")
	}
	specCount := 0
	for _, provider := range bundle.Providers {
		specCount += len(provider.SpecReferences)
	}
	if specCount != generatedSpecReferenceCount {
		panic(fmt.Sprintf("embedded catalog has %d spec references, want %d", specCount, generatedSpecReferenceCount))
	}
	if err := validateGeneratedIDIndex("candidate", indexes.candidates, len(bundle.Candidates), func(index int) string { return bundle.Candidates[index].ID }); err != nil {
		panic(err)
	}
	if err := validateGeneratedIDIndex("provider", indexes.providers, len(bundle.Providers), func(index int) string { return bundle.Providers[index].ID }); err != nil {
		panic(err)
	}
	if err := validateGeneratedGroupIndex("security classification", indexes.classificationsByProvider, len(bundle.SecurityClassifications), func(index int) string { return bundle.SecurityClassifications[index].ProviderID }); err != nil {
		panic(err)
	}
	if err := validateGeneratedGroupIndex("security overlay", indexes.overlaysByProvider, len(bundle.SecurityOverlays), func(index int) string { return bundle.SecurityOverlays[index].ProviderID }); err != nil {
		panic(err)
	}
	return bundle
}

func validateGeneratedIDIndex(label string, index map[string]int, count int, idAt func(int) string) error {
	if len(index) != count {
		return fmt.Errorf("generated %s index has %d entries, want %d", label, len(index), count)
	}
	seen := make([]bool, count)
	for id, position := range index {
		if position < 0 || position >= count || seen[position] || idAt(position) != id {
			return fmt.Errorf("generated %s index entry %q=%d is invalid", label, id, position)
		}
		seen[position] = true
	}
	return nil
}

func validateGeneratedGroupIndex(label string, groups map[string][]int, count int, providerAt func(int) string) error {
	seen := make([]bool, count)
	for providerID, indexes := range groups {
		for _, index := range indexes {
			if index < 0 || index >= count || seen[index] || providerAt(index) != providerID {
				return fmt.Errorf("generated %s index entry %q=%d is invalid", label, providerID, index)
			}
			seen[index] = true
		}
	}
	for index, ok := range seen {
		if !ok {
			return fmt.Errorf("generated %s index is missing record %d", label, index)
		}
	}
	return nil
}

func mustCandidateLookupIndex(candidates []Candidate) map[string]int {
	index := map[string]int{}
	for position, candidate := range candidates {
		for _, key := range candidate.lookupKeys() {
			if _, exists := index[key]; exists {
				panic(fmt.Sprintf("duplicate built-in candidate lookup key %q", key))
			}
			index[key] = position
		}
	}
	return index
}

func mustProviderLookupIndex(providers []Provider) map[string]int {
	index := map[string]int{}
	for position, provider := range providers {
		for _, key := range provider.lookupKeys() {
			if _, exists := index[key]; exists {
				panic(fmt.Sprintf("duplicate built-in provider lookup key %q", key))
			}
			index[key] = position
		}
	}
	return index
}
