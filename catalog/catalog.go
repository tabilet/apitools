package catalog

// Catalog is a deterministic metadata-only view of provider catalog entries.
type Catalog struct {
	Providers        []Provider        `json:"providers,omitempty"`
	SecurityOverlays []SecurityOverlay `json:"security_overlays,omitempty"`
}

// BuiltInCatalog returns the built-in provider catalog. Callers receive
// independent copies of all nested slices.
func BuiltInCatalog() Catalog {
	return Catalog{
		Providers:        BuiltInProviders(),
		SecurityOverlays: BuiltInSecurityOverlays(),
	}
}

// ListProviders returns providers in deterministic order.
func (c Catalog) ListProviders() []Provider {
	out := cloneProviders(c.Providers)
	sortProviders(out)
	return out
}

// ListSecurityOverlays returns security overlays in deterministic order.
func (c Catalog) ListSecurityOverlays() []SecurityOverlay {
	out := cloneSecurityOverlays(c.SecurityOverlays)
	sortSecurityOverlays(out)
	return out
}

// FindProvider returns a provider by ID, display name, or alias.
func (c Catalog) FindProvider(key string) (Provider, bool) {
	normalized := normalizeKey(key)
	if normalized == "" {
		return Provider{}, false
	}
	for _, provider := range c.ListProviders() {
		if provider.matches(normalized) {
			return provider, true
		}
	}
	return Provider{}, false
}

// Validate validates catalog provider records.
func (c Catalog) Validate() error {
	if err := ValidateProviders(c.Providers); err != nil {
		return err
	}
	return ValidateSecurityOverlays(c.SecurityOverlays, c.Providers)
}
