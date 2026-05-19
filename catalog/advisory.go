package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// ProviderAdvisoryOptions controls metadata-only provider advisory reports.
type ProviderAdvisoryOptions struct {
	Catalog                 Catalog
	SecurityClassifications []SecurityClassification
	ProviderKey             string
	Artifacts               []CatalogSpecArtifact
}

// ProviderAdvisoryReport is a deterministic metadata-only report for one or
// more catalog providers.
type ProviderAdvisoryReport struct {
	Providers []ProviderAdvisory `json:"providers,omitempty"`
}

// ProviderAdvisory summarizes catalog guidance for a provider without making
// that metadata authoritative.
type ProviderAdvisory struct {
	ProviderID              string                       `json:"provider_id"`
	DisplayName             string                       `json:"display_name"`
	Aliases                 []string                     `json:"aliases,omitempty"`
	Category                string                       `json:"category,omitempty"`
	WorkflowRelevance       string                       `json:"workflow_relevance,omitempty"`
	OpenAPIAvailability     SpecAvailability             `json:"openapi_availability"`
	MachineSpecAvailability SpecAvailability             `json:"machine_spec_availability"`
	UserOpenAPINeed         UserOpenAPINeed              `json:"user_openapi_need"`
	AuthStatus              AuthCompletenessStatus       `json:"auth_status"`
	SpecReferences          []AdvisorySpecReference      `json:"spec_references,omitempty"`
	SecurityOverlayIDs      []string                     `json:"security_overlay_ids,omitempty"`
	SecuritySpecRefIDs      []string                     `json:"security_spec_ref_ids,omitempty"`
	SecuritySourceNotes     []string                     `json:"security_source_notes,omitempty"`
	Quirks                  []string                     `json:"quirks,omitempty"`
	ManualFollowUps         []string                     `json:"manual_follow_ups,omitempty"`
	RegisteredArtifactPaths []AdvisoryRegisteredArtifact `json:"registered_artifact_paths,omitempty"`
	EndpointOverlays        []AdvisoryEndpointOverlay    `json:"endpoint_overlays,omitempty"`
	ResolvedOpenAPI         ResolvedReference            `json:"resolved_openapi"`
	ResolvedSecurity        ResolvedReference            `json:"resolved_security"`
}

// AdvisorySpecReference records one provider spec or documentation reference,
// including an optional local artifact path joined from a cache registry.
type AdvisorySpecReference struct {
	ID                     string          `json:"id"`
	Kind                   SpecKind        `json:"kind"`
	Protocol               SpecProtocol    `json:"protocol"`
	ProtocolVersion        string          `json:"protocol_version,omitempty"`
	URL                    string          `json:"url"`
	SourceAuthority        SourceAuthority `json:"source_authority"`
	Version                string          `json:"version,omitempty"`
	VerifiedAt             string          `json:"verified_at,omitempty"`
	Revision               string          `json:"revision,omitempty"`
	LicenseNote            string          `json:"license_note,omitempty"`
	SourceNote             string          `json:"source_note,omitempty"`
	RegisteredArtifactPath string          `json:"registered_artifact_path,omitempty"`
}

// AdvisoryRegisteredArtifact records a cache-registered artifact path that was
// joined into advisory output.
type AdvisoryRegisteredArtifact struct {
	SpecRefID string `json:"spec_ref_id"`
	Kind      string `json:"kind,omitempty"`
	Path      string `json:"path"`
}

// AdvisoryEndpointOverlay records a docs-derived endpoint overlay registered
// in the local catalog artifact cache.
type AdvisoryEndpointOverlay struct {
	ArtifactID  string `json:"artifact_id"`
	Kind        string `json:"kind,omitempty"`
	Path        string `json:"path"`
	OverlayPath string `json:"overlay_path,omitempty"`
	BuilderPath string `json:"builder_path,omitempty"`
}

// BuiltInProviderAdvisoryReport builds advisory output for built-in catalog
// metadata without network access or cache writes.
func BuiltInProviderAdvisoryReport(options ProviderAdvisoryOptions) (ProviderAdvisoryReport, error) {
	options.Catalog = BuiltInCatalog()
	if options.SecurityClassifications == nil {
		options.SecurityClassifications = BuiltInSecurityClassifications()
	}
	return BuildProviderAdvisoryReport(options)
}

// BuildProviderAdvisoryReport builds advisory output from catalog metadata,
// security classifications, overlays, and optional cache artifact path rows. It
// does not fetch documents, execute operations, resolve credentials, or apply
// overlays to upstream specs.
func BuildProviderAdvisoryReport(options ProviderAdvisoryOptions) (ProviderAdvisoryReport, error) {
	catalog := options.Catalog
	classifications := options.SecurityClassifications
	if catalog.isZero() {
		catalog = BuiltInCatalog()
		if classifications == nil {
			classifications = BuiltInSecurityClassifications()
		}
	}
	if err := catalog.Validate(); err != nil {
		return ProviderAdvisoryReport{}, err
	}
	securityReport, err := BuildSecurityReport(catalog.Providers, catalog.SecurityOverlays, classifications)
	if err != nil {
		return ProviderAdvisoryReport{}, err
	}

	var providers []Provider
	if key := strings.TrimSpace(options.ProviderKey); key != "" {
		provider, ok := catalog.FindProvider(key)
		if !ok {
			return ProviderAdvisoryReport{}, fmt.Errorf("unknown provider %q", options.ProviderKey)
		}
		providers = []Provider{provider}
	} else {
		providers = catalog.ListProviders()
	}

	artifactPathByKey := advisoryArtifactPathMap(options.Artifacts)
	endpointOverlaysByProvider := advisoryEndpointOverlaysByProvider(options.Artifacts)
	overlaysByProvider := map[string][]SecurityOverlay{}
	for _, overlay := range catalog.ListSecurityOverlays() {
		overlaysByProvider[overlay.ProviderID] = append(overlaysByProvider[overlay.ProviderID], overlay)
	}

	out := make([]ProviderAdvisory, 0, len(providers))
	for _, provider := range providers {
		report, _ := securityReport.FindProvider(provider.ID)
		resolved, err := ResolveProvider(ResolveProviderOptions{
			Catalog:                 catalog,
			SecurityClassifications: classifications,
			ProviderKey:             provider.ID,
		})
		if err != nil {
			return ProviderAdvisoryReport{}, err
		}
		advisory := ProviderAdvisory{
			ProviderID:              provider.ID,
			DisplayName:             provider.DisplayName,
			Aliases:                 append([]string(nil), provider.Aliases...),
			Category:                provider.Category,
			WorkflowRelevance:       provider.WorkflowRelevance,
			OpenAPIAvailability:     provider.OfficialOpenAPIAvailability,
			MachineSpecAvailability: provider.OfficialMachineSpecAvailability,
			UserOpenAPINeed:         provider.UserOpenAPINeed,
			AuthStatus:              report.Status,
			SecurityOverlayIDs:      append([]string(nil), report.OverlayIDs...),
			SecuritySpecRefIDs:      append([]string(nil), report.SpecRefIDs...),
			SecuritySourceNotes:     append([]string(nil), report.SourceNotes...),
			Quirks:                  append([]string(nil), provider.Quirks...),
			EndpointOverlays:        cloneAdvisoryEndpointOverlays(endpointOverlaysByProvider[provider.ID]),
			ResolvedOpenAPI:         resolved.OpenAPI,
			ResolvedSecurity:        resolved.Security,
		}
		for _, ref := range provider.SpecReferences {
			artifactPath := artifactPathByKey[provider.ID+"/"+ref.ID]
			protocol := ref.ProtocolClassification()
			advisory.SpecReferences = append(advisory.SpecReferences, AdvisorySpecReference{
				ID:                     ref.ID,
				Kind:                   ref.Kind,
				Protocol:               protocol.Protocol,
				ProtocolVersion:        protocol.Version,
				URL:                    ref.URL,
				SourceAuthority:        ref.SourceAuthority,
				Version:                ref.Version,
				VerifiedAt:             ref.VerifiedAt,
				Revision:               ref.Revision,
				LicenseNote:            ref.LicenseNote,
				SourceNote:             ref.SourceNote,
				RegisteredArtifactPath: artifactPath.path,
			})
			if artifactPath.path != "" {
				advisory.RegisteredArtifactPaths = append(advisory.RegisteredArtifactPaths, AdvisoryRegisteredArtifact{
					SpecRefID: ref.ID,
					Kind:      artifactPath.kind,
					Path:      artifactPath.path,
				})
			}
		}
		advisory.ManualFollowUps = advisoryManualFollowUps(provider, report, overlaysByProvider[provider.ID], advisory.SpecReferences, advisory.EndpointOverlays)
		out = append(out, advisory)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ProviderID < out[j].ProviderID
	})
	return ProviderAdvisoryReport{Providers: out}, nil
}

type advisoryArtifactPath struct {
	kind string
	path string
}

func advisoryArtifactPathMap(artifacts []CatalogSpecArtifact) map[string]advisoryArtifactPath {
	out := map[string]advisoryArtifactPath{}
	for _, artifact := range artifacts {
		providerID := strings.TrimSpace(artifact.ProviderID)
		specRefID := strings.TrimSpace(artifact.SpecRefID)
		path := strings.TrimSpace(artifact.Path)
		if providerID == "" || specRefID == "" || path == "" {
			continue
		}
		out[providerID+"/"+specRefID] = advisoryArtifactPath{
			kind: strings.TrimSpace(artifact.Kind),
			path: path,
		}
	}
	return out
}

func advisoryEndpointOverlaysByProvider(artifacts []CatalogSpecArtifact) map[string][]AdvisoryEndpointOverlay {
	out := map[string][]AdvisoryEndpointOverlay{}
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) != "advisory-overlay" {
			continue
		}
		providerID := strings.TrimSpace(artifact.ProviderID)
		path := strings.TrimSpace(artifact.Path)
		if providerID == "" || path == "" {
			continue
		}
		artifactID := strings.TrimSpace(artifact.ArtifactID)
		if artifactID == "" {
			artifactID = strings.TrimSpace(artifact.SpecRefID)
		}
		if artifactID == "" {
			continue
		}
		overlayPath := strings.TrimSpace(artifact.OverlayPath)
		if overlayPath == "" {
			overlayPath = path
		}
		out[providerID] = append(out[providerID], AdvisoryEndpointOverlay{
			ArtifactID:  artifactID,
			Kind:        strings.TrimSpace(artifact.Kind),
			Path:        path,
			OverlayPath: overlayPath,
			BuilderPath: strings.TrimSpace(artifact.BuilderPath),
		})
	}
	for providerID := range out {
		sort.SliceStable(out[providerID], func(i, j int) bool {
			if out[providerID][i].ArtifactID == out[providerID][j].ArtifactID {
				return out[providerID][i].Path < out[providerID][j].Path
			}
			return out[providerID][i].ArtifactID < out[providerID][j].ArtifactID
		})
	}
	return out
}

func cloneAdvisoryEndpointOverlays(overlays []AdvisoryEndpointOverlay) []AdvisoryEndpointOverlay {
	return append([]AdvisoryEndpointOverlay(nil), overlays...)
}

func advisoryManualFollowUps(provider Provider, report ProviderSecurityReport, overlays []SecurityOverlay, refs []AdvisorySpecReference, endpointOverlays []AdvisoryEndpointOverlay) []string {
	var followUps []string
	if provider.UserOpenAPINeed == UserOpenAPINeedLikely {
		if len(endpointOverlays) > 0 {
			followUps = append(followUps, "Review registered docs-derived endpoint overlay metadata for the covered advisory subset; broader workflows may still need a user-provided or generated OpenAPI document.")
		} else {
			followUps = append(followUps, "OpenAPI-only workflows likely need a user-provided or generated OpenAPI document before import.")
		}
	}
	if len(report.OverlayIDs) > 0 {
		followUps = append(followUps, "Review advisory security overlay metadata before using it with an imported OpenAPI document.")
	}
	for _, ref := range refs {
		if !refreshableSpecKind(ref.Kind) || ref.RegisteredArtifactPath != "" {
			continue
		}
		followUps = append(followUps, fmt.Sprintf("Optional: run catalog refresh for %s to register a local review artifact path.", ref.ID))
	}
	for _, overlay := range overlays {
		if overlay.Status == AuthStatusPresentIncomplete {
			followUps = append(followUps, fmt.Sprintf("Review present-incomplete security overlay %s before downstream credential handoff.", overlay.ID))
		}
	}
	return sortedUniqueStrings(followUps)
}
