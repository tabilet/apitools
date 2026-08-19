package apitools

import (
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
)

// CatalogStatsReport is an offline aggregate over catalog providers, local
// artifact registrations, and refresh-review evidence.
type CatalogStatsReport struct {
	ProviderCount    int                          `json:"provider_count"`
	Protocols        []CatalogStatsProtocol       `json:"protocols"`
	ArtifactRegistry CatalogStatsArtifactRegistry `json:"artifact_registry"`
	Refresh          CatalogStatsRefreshEvidence  `json:"refresh"`
}

// CatalogStatsProtocol records a deterministic protocol bucket.
type CatalogStatsProtocol struct {
	Protocol    catalog.SpecProtocol `json:"protocol"`
	DisplayName string               `json:"display_name"`
	Count       int                  `json:"count"`
	ProviderIDs []string             `json:"provider_ids,omitempty"`
}

// CatalogStatsRefreshEvidence summarizes offline refresh-review results.
type CatalogStatsRefreshEvidence struct {
	RefreshableSpecCount       int                      `json:"refreshable_spec_count"`
	RegisteredArtifactCount    int                      `json:"registered_artifact_count"`
	ExistingArtifactCount      int                      `json:"existing_artifact_count"`
	MissingRegistrationCount   int                      `json:"missing_registration_count"`
	MissingRegisteredFileCount int                      `json:"missing_registered_file_count"`
	InvalidArtifactCount       int                      `json:"invalid_artifact_count"`
	StaleVerificationCount     int                      `json:"stale_verification_count"`
	TotalBytes                 int64                    `json:"total_bytes"`
	ValidationStatuses         []CatalogStatsValidation `json:"validation_statuses,omitempty"`
	Protocols                  []CatalogStatsProtocol   `json:"protocols,omitempty"`
}

// CatalogStatsArtifactRegistry summarizes local artifact registrations.
type CatalogStatsArtifactRegistry struct {
	ArtifactCount int                        `json:"artifact_count"`
	Kinds         []CatalogStatsArtifactKind `json:"kinds,omitempty"`
}

// CatalogStatsArtifactKind records one artifact-kind bucket.
type CatalogStatsArtifactKind struct {
	Kind        string   `json:"kind"`
	Count       int      `json:"count"`
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
}

// CatalogStatsValidation records one refresh validation-status bucket.
type CatalogStatsValidation struct {
	Status     string   `json:"status"`
	Count      int      `json:"count"`
	SpecRefIDs []string `json:"spec_ref_ids,omitempty"`
}

// BuildCatalogStatsReport applies catalog statistics policy outside the CLI.
// It performs no network requests or filesystem reads.
func BuildCatalogStatsReport(providers []catalog.Provider, artifacts []catalog.CatalogSpecArtifact, refreshReport CatalogSpecRefreshReviewReport) CatalogStatsReport {
	return CatalogStatsReport{
		ProviderCount:    len(providers),
		Protocols:        catalogProviderProtocolStats(providers),
		ArtifactRegistry: catalogArtifactRegistryStats(artifacts),
		Refresh:          catalogRefreshEvidenceStats(refreshReport),
	}
}

func catalogProviderProtocolStats(providers []catalog.Provider) []CatalogStatsProtocol {
	providerIDsByProtocol := map[catalog.SpecProtocol][]string{}
	for _, provider := range providers {
		protocol := primaryCatalogProviderProtocol(provider).Protocol
		providerIDsByProtocol[protocol] = append(providerIDsByProtocol[protocol], provider.ID)
	}
	out := make([]CatalogStatsProtocol, 0, len(catalogProtocolOrder()))
	for _, protocol := range catalogProtocolOrder() {
		providerIDs := sortedCatalogStatsStrings(providerIDsByProtocol[protocol])
		out = append(out, CatalogStatsProtocol{
			Protocol:    protocol,
			DisplayName: catalogProtocolDisplayName(protocol),
			Count:       len(providerIDs),
			ProviderIDs: providerIDs,
		})
	}
	return out
}

func primaryCatalogProviderProtocol(provider catalog.Provider) catalog.SpecProtocolClassification {
	for _, ref := range provider.SpecReferences {
		if ref.Kind != catalog.SpecKindHumanDocs {
			return ref.ProtocolClassification()
		}
	}
	for _, ref := range provider.SpecReferences {
		if ref.Kind == catalog.SpecKindHumanDocs {
			return ref.ProtocolClassification()
		}
	}
	return catalog.SpecProtocolClassification{Protocol: catalog.SpecProtocolUnknown}
}

func catalogRefreshEvidenceStats(report CatalogSpecRefreshReviewReport) CatalogStatsRefreshEvidence {
	statusIDs := map[string][]string{}
	providerIDsByProtocol := map[catalog.SpecProtocol][]string{}
	stats := CatalogStatsRefreshEvidence{RefreshableSpecCount: len(report.Results)}
	for _, result := range report.Results {
		key := result.ProviderID + "/" + result.SpecRefID
		if strings.TrimSpace(result.RegisteredArtifactPath) != "" {
			stats.RegisteredArtifactCount++
		}
		if result.Exists {
			stats.ExistingArtifactCount++
		}
		if result.Status == CatalogRefreshMissingRegistration {
			stats.MissingRegistrationCount++
		}
		if result.Status == CatalogRefreshMissingFile {
			stats.MissingRegisteredFileCount++
		}
		if result.Status == CatalogRefreshInvalid || result.RawValidationStatus == CatalogRefreshInvalid {
			stats.InvalidArtifactCount++
		}
		if result.VerificationStale {
			stats.StaleVerificationCount++
		}
		stats.TotalBytes += result.Bytes
		status := result.RawValidationStatus
		if status == "" {
			status = result.Status
		}
		statusIDs[status] = append(statusIDs[status], key)
		providerIDsByProtocol[result.Protocol] = append(providerIDsByProtocol[result.Protocol], key)
	}
	for _, status := range catalogRefreshValidationOrder(statusIDs) {
		ids := sortedCatalogStatsStrings(statusIDs[status])
		stats.ValidationStatuses = append(stats.ValidationStatuses, CatalogStatsValidation{
			Status:     status,
			Count:      len(ids),
			SpecRefIDs: ids,
		})
	}
	for _, protocol := range catalogProtocolOrder() {
		ids := sortedCatalogStatsStrings(providerIDsByProtocol[protocol])
		if len(ids) == 0 {
			continue
		}
		stats.Protocols = append(stats.Protocols, CatalogStatsProtocol{
			Protocol:    protocol,
			DisplayName: catalogProtocolDisplayName(protocol),
			Count:       len(ids),
			ProviderIDs: ids,
		})
	}
	return stats
}

func catalogArtifactRegistryStats(artifacts []catalog.CatalogSpecArtifact) CatalogStatsArtifactRegistry {
	idsByKind := map[string][]string{}
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		if kind == "" {
			kind = "unknown"
		}
		idsByKind[kind] = append(idsByKind[kind], strings.TrimSpace(artifact.ProviderID)+"/"+strings.TrimSpace(artifact.SpecRefID))
	}
	kinds := make([]string, 0, len(idsByKind))
	for kind := range idsByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	stats := CatalogStatsArtifactRegistry{ArtifactCount: len(artifacts)}
	for _, kind := range kinds {
		ids := sortedCatalogStatsStrings(idsByKind[kind])
		stats.Kinds = append(stats.Kinds, CatalogStatsArtifactKind{
			Kind:        kind,
			Count:       len(ids),
			ArtifactIDs: ids,
		})
	}
	return stats
}

func catalogRefreshValidationOrder(statusIDs map[string][]string) []string {
	preferred := []string{
		CatalogRefreshValidOpenAPI,
		CatalogRefreshValidSwagger,
		CatalogRefreshParseableOpenAPIInvalid,
		CatalogRefreshParseableSwaggerInvalid,
		CatalogRefreshValidStructured,
		CatalogRefreshSkippedValidation,
		CatalogRefreshMissingRegistration,
		CatalogRefreshMissingFile,
		CatalogRefreshInvalid,
	}
	seen := map[string]struct{}{}
	var out []string
	for _, status := range preferred {
		if _, ok := statusIDs[status]; ok {
			out = append(out, status)
			seen[status] = struct{}{}
		}
	}
	var extra []string
	for status := range statusIDs {
		if _, ok := seen[status]; !ok {
			extra = append(extra, status)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

func catalogProtocolOrder() []catalog.SpecProtocol {
	return []catalog.SpecProtocol{
		catalog.SpecProtocolOpenAPI,
		catalog.SpecProtocolSwagger,
		catalog.SpecProtocolSmithy,
		catalog.SpecProtocolGoogleDiscovery,
		catalog.SpecProtocolAsyncAPI,
		catalog.SpecProtocolOpenRPC,
		catalog.SpecProtocolGraphQL,
		catalog.SpecProtocolGRPCProtobuf,
		catalog.SpecProtocolOData,
		catalog.SpecProtocolDropboxStone,
		catalog.SpecProtocolOpenAPIIndex,
		catalog.SpecProtocolHumanDocs,
		catalog.SpecProtocolUnknown,
	}
}

func catalogProtocolDisplayName(protocol catalog.SpecProtocol) string {
	switch protocol {
	case catalog.SpecProtocolOpenAPI:
		return "OpenAPI"
	case catalog.SpecProtocolSwagger:
		return "Swagger"
	case catalog.SpecProtocolSmithy:
		return "Smithy"
	case catalog.SpecProtocolGoogleDiscovery:
		return "Google Discovery"
	case catalog.SpecProtocolAsyncAPI:
		return "AsyncAPI"
	case catalog.SpecProtocolOpenRPC:
		return "OpenRPC"
	case catalog.SpecProtocolGraphQL:
		return "GraphQL"
	case catalog.SpecProtocolGRPCProtobuf:
		return "gRPC/protobuf"
	case catalog.SpecProtocolOData:
		return "OData"
	case catalog.SpecProtocolDropboxStone:
		return "Dropbox Stone"
	case catalog.SpecProtocolOpenAPIIndex:
		return "OpenAPI index"
	case catalog.SpecProtocolHumanDocs:
		return "Human docs"
	default:
		return "Unknown"
	}
}

func sortedCatalogStatsStrings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
