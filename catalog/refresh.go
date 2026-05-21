package catalog

import (
	"sort"
	"strings"
)

// CatalogSpecArtifact records a locally registered catalog artifact path that
// can be joined with refreshable spec references for maintainer review.
type CatalogSpecArtifact struct {
	ProviderID  string            `json:"provider_id"`
	SpecRefID   string            `json:"spec_ref_id"`
	ArtifactID  string            `json:"artifact_id,omitempty"`
	Kind        string            `json:"kind,omitempty"`
	Path        string            `json:"path,omitempty"`
	SourceURL   string            `json:"source_url,omitempty"`
	OverlayPath string            `json:"overlay_path,omitempty"`
	BuilderPath string            `json:"builder_path,omitempty"`
	SHA256      string            `json:"sha256,omitempty"`
	Bytes       int64             `json:"bytes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RefreshableSpecReference is a deterministic, metadata-only row describing a
// built-in catalog spec reference that can be refreshed by an opt-in workflow.
type RefreshableSpecReference struct {
	ProviderID             string          `json:"provider_id"`
	ProviderName           string          `json:"provider_name"`
	SpecRefID              string          `json:"spec_ref_id"`
	Kind                   SpecKind        `json:"kind"`
	Protocol               SpecProtocol    `json:"protocol"`
	ProtocolVersion        string          `json:"protocol_version,omitempty"`
	URL                    string          `json:"url"`
	SourceAuthority        SourceAuthority `json:"source_authority"`
	VerifiedAt             string          `json:"verified_at,omitempty"`
	RegisteredArtifactPath string          `json:"registered_artifact_path,omitempty"`
}

// ProtocolClassification returns the protocol/model family for a refreshable
// row. Built-in rows carry this explicitly; caller-constructed rows fall back
// to their source kind.
func (ref RefreshableSpecReference) ProtocolClassification() SpecProtocolClassification {
	if ref.Protocol != "" {
		return SpecProtocolClassification{Protocol: ref.Protocol, Version: ref.ProtocolVersion}
	}
	return specProtocolClassificationForKind(ref.Kind)
}

// BuiltInRefreshableSpecReferences returns refreshable built-in provider spec
// references without network access. Human documentation references are omitted
// because refresh tooling only downloads machine-readable review artifacts.
func BuiltInRefreshableSpecReferences(artifacts []CatalogSpecArtifact) []RefreshableSpecReference {
	return RefreshableSpecReferences(BuiltInProviders(), artifacts)
}

// RefreshableSpecReferences returns refreshable spec references for providers
// in deterministic provider/spec order.
func RefreshableSpecReferences(providers []Provider, artifacts []CatalogSpecArtifact) []RefreshableSpecReference {
	artifactPathByKey := map[string]string{}
	for _, artifact := range artifacts {
		providerID := strings.TrimSpace(artifact.ProviderID)
		specRefID := strings.TrimSpace(artifact.SpecRefID)
		if providerID == "" || specRefID == "" {
			continue
		}
		if path := strings.TrimSpace(artifact.Path); path != "" {
			artifactPathByKey[providerID+"/"+specRefID] = path
		}
	}
	var rows []RefreshableSpecReference
	for _, provider := range cloneProviders(providers) {
		for _, ref := range provider.SpecReferences {
			if !refreshableSpecKind(ref.Kind) {
				continue
			}
			protocol := ref.ProtocolClassification()
			rows = append(rows, RefreshableSpecReference{
				ProviderID:             provider.ID,
				ProviderName:           provider.DisplayName,
				SpecRefID:              ref.ID,
				Kind:                   ref.Kind,
				Protocol:               protocol.Protocol,
				ProtocolVersion:        protocol.Version,
				URL:                    ref.URL,
				SourceAuthority:        ref.SourceAuthority,
				VerifiedAt:             ref.VerifiedAt,
				RegisteredArtifactPath: artifactPathByKey[provider.ID+"/"+ref.ID],
			})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ProviderID == rows[j].ProviderID {
			return rows[i].SpecRefID < rows[j].SpecRefID
		}
		return rows[i].ProviderID < rows[j].ProviderID
	})
	return rows
}

func refreshableSpecKind(kind SpecKind) bool {
	switch kind {
	case SpecKindOpenAPI, SpecKindOpenAPIIndex, SpecKindDropboxStone, SpecKindGoogleDiscovery, SpecKindSmithyJSON:
		return true
	default:
		return false
	}
}
