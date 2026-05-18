package catalog

// Catalog is a deterministic metadata-only view of provider catalog entries.
type Catalog struct {
	Providers []Provider `json:"providers,omitempty"`
}

// BuiltInCatalog returns the built-in provider catalog. Callers receive
// independent copies of all nested slices.
func BuiltInCatalog() Catalog {
	return Catalog{Providers: BuiltInProviders()}
}

// ListProviders returns providers in deterministic order.
func (c Catalog) ListProviders() []Provider {
	out := cloneProviders(c.Providers)
	sortProviders(out)
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
	return ValidateProviders(c.Providers)
}
