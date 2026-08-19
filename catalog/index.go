package catalog

import "strings"

// CatalogIndex is an immutable, reusable lookup view over a validated catalog
// and its security evidence. All exported accessors return independent copies.
type CatalogIndex struct {
	providers          []Provider
	overlays           []SecurityOverlay
	providerByLookup   map[string]int
	specByProvider     map[string]map[string]int
	overlaysByProvider map[string][]int
	securityReport     SecurityReport
	securityByProvider map[string]int
}

var builtInCatalogIndex = mustCatalogIndex(BuiltInCatalog(), BuiltInSecurityClassifications())

// NewCatalogIndex validates and indexes catalog metadata once for repeated
// provider, spec, overlay, and security-report lookups.
func NewCatalogIndex(cat Catalog, classifications []SecurityClassification) (*CatalogIndex, error) {
	if err := cat.Validate(); err != nil {
		return nil, err
	}
	report, err := BuildSecurityReport(cat.Providers, cat.SecurityOverlays, classifications)
	if err != nil {
		return nil, err
	}
	providers := cat.ListProviders()
	overlays := cat.ListSecurityOverlays()
	index := &CatalogIndex{
		providers:          providers,
		overlays:           overlays,
		providerByLookup:   map[string]int{},
		specByProvider:     map[string]map[string]int{},
		overlaysByProvider: map[string][]int{},
		securityReport:     report,
		securityByProvider: map[string]int{},
	}
	for providerIndex, provider := range providers {
		for _, key := range provider.lookupKeys() {
			index.providerByLookup[key] = providerIndex
		}
		specs := make(map[string]int, len(provider.SpecReferences))
		for specIndex, ref := range provider.SpecReferences {
			specs[ref.ID] = specIndex
		}
		index.specByProvider[provider.ID] = specs
	}
	for overlayIndex, overlay := range overlays {
		index.overlaysByProvider[overlay.ProviderID] = append(index.overlaysByProvider[overlay.ProviderID], overlayIndex)
	}
	for reportIndex, provider := range report.Providers {
		index.securityByProvider[provider.ProviderID] = reportIndex
	}
	return index, nil
}

// BuiltInCatalogIndex returns the process-wide immutable index for reviewed
// built-in catalog metadata.
func BuiltInCatalogIndex() *CatalogIndex {
	return builtInCatalogIndex
}

func mustCatalogIndex(cat Catalog, classifications []SecurityClassification) *CatalogIndex {
	index, err := NewCatalogIndex(cat, classifications)
	if err != nil {
		panic(err)
	}
	return index
}

// ListProviders returns indexed providers in deterministic order.
func (i *CatalogIndex) ListProviders() []Provider {
	if i == nil {
		return nil
	}
	return cloneProviders(i.providers)
}

// FindProvider finds a provider by ID, display name, or alias in constant
// expected time.
func (i *CatalogIndex) FindProvider(key string) (Provider, bool) {
	if i == nil {
		return Provider{}, false
	}
	position, ok := i.providerByLookup[normalizeKey(key)]
	if !ok {
		return Provider{}, false
	}
	return cloneProvider(i.providers[position]), true
}

// FindSpecReference finds one exact spec reference for a provider ID. Provider
// aliases are intentionally not accepted for the first argument.
func (i *CatalogIndex) FindSpecReference(providerID, specRefID string) (SpecReference, bool) {
	if i == nil {
		return SpecReference{}, false
	}
	providerPosition, ok := i.providerByLookup[normalizeKey(providerID)]
	if !ok || i.providers[providerPosition].ID != strings.TrimSpace(providerID) {
		return SpecReference{}, false
	}
	position, ok := i.specByProvider[providerID][strings.TrimSpace(specRefID)]
	if !ok {
		return SpecReference{}, false
	}
	return i.providers[providerPosition].SpecReferences[position], true
}

// SecurityOverlaysForProvider returns indexed overlays for a provider ID.
func (i *CatalogIndex) SecurityOverlaysForProvider(providerID string) []SecurityOverlay {
	if i == nil {
		return nil
	}
	indexes := i.overlaysByProvider[strings.TrimSpace(providerID)]
	out := make([]SecurityOverlay, 0, len(indexes))
	for _, position := range indexes {
		out = append(out, cloneSecurityOverlay(i.overlays[position]))
	}
	return out
}

// SecurityForProvider returns the precomputed security report for a provider
// ID.
func (i *CatalogIndex) SecurityForProvider(providerID string) (ProviderSecurityReport, bool) {
	if i == nil {
		return ProviderSecurityReport{}, false
	}
	position, ok := i.securityByProvider[strings.TrimSpace(providerID)]
	if !ok {
		return ProviderSecurityReport{}, false
	}
	return cloneProviderSecurityReport(i.securityReport.Providers[position]), true
}

// SecurityReport returns the complete precomputed report as an independent
// copy.
func (i *CatalogIndex) SecurityReport() SecurityReport {
	if i == nil {
		return SecurityReport{}
	}
	return SecurityReport{Providers: SecurityReportRows(i.securityReport)}
}
