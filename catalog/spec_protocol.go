package catalog

import "strings"

// SpecProtocol identifies the API description protocol or model family
// represented by a spec reference. It is separate from SpecKind so callers can
// distinguish source container kinds from server-side API model families.
type SpecProtocol string

const (
	SpecProtocolOpenAPI         SpecProtocol = "openapi"
	SpecProtocolSwagger         SpecProtocol = "swagger"
	SpecProtocolSmithy          SpecProtocol = "smithy"
	SpecProtocolGoogleDiscovery SpecProtocol = "google-discovery"
	SpecProtocolDropboxStone    SpecProtocol = "dropbox-stone"
	SpecProtocolOpenAPIIndex    SpecProtocol = "openapi-index"
	SpecProtocolHumanDocs       SpecProtocol = "human-docs"
	SpecProtocolUnknown         SpecProtocol = "unknown"
)

// SpecProtocolClassification records the protocol family and, when known, the
// protocol version for a catalog spec reference.
type SpecProtocolClassification struct {
	Protocol SpecProtocol `json:"protocol"`
	Version  string       `json:"version,omitempty"`
}

// UWSSourceType returns the first-class UWS sourceDescription.type for
// protocols that can be represented directly in UWS 1.2. Empty means the
// protocol is not a first-class UWS source type.
func (c SpecProtocolClassification) UWSSourceType() string {
	switch c.Protocol {
	case SpecProtocolOpenAPI, SpecProtocolSwagger:
		return "openapi"
	case SpecProtocolGoogleDiscovery:
		return "google-discovery"
	case SpecProtocolSmithy:
		return "aws-smithy"
	default:
		return ""
	}
}

// ProtocolClassification returns the protocol/model family represented by a
// spec reference without fetching or parsing the remote document.
func (ref SpecReference) ProtocolClassification() SpecProtocolClassification {
	switch ref.Kind {
	case SpecKindOpenAPI:
		if specReferenceLooksSwagger(ref) {
			return SpecProtocolClassification{Protocol: SpecProtocolSwagger, Version: "2.0"}
		}
		return SpecProtocolClassification{Protocol: SpecProtocolOpenAPI, Version: inferredOpenAPIVersion(ref)}
	case SpecKindOpenAPIIndex:
		return SpecProtocolClassification{Protocol: SpecProtocolOpenAPIIndex}
	case SpecKindSmithyJSON:
		return SpecProtocolClassification{Protocol: SpecProtocolSmithy}
	case SpecKindGoogleDiscovery:
		return SpecProtocolClassification{Protocol: SpecProtocolGoogleDiscovery}
	case SpecKindDropboxStone:
		return SpecProtocolClassification{Protocol: SpecProtocolDropboxStone}
	case SpecKindHumanDocs:
		return SpecProtocolClassification{Protocol: SpecProtocolHumanDocs}
	default:
		return SpecProtocolClassification{Protocol: SpecProtocolUnknown}
	}
}

func specProtocolClassificationForKind(kind SpecKind) SpecProtocolClassification {
	switch kind {
	case SpecKindOpenAPI:
		return SpecProtocolClassification{Protocol: SpecProtocolOpenAPI}
	case SpecKindOpenAPIIndex:
		return SpecProtocolClassification{Protocol: SpecProtocolOpenAPIIndex}
	case SpecKindSmithyJSON:
		return SpecProtocolClassification{Protocol: SpecProtocolSmithy}
	case SpecKindGoogleDiscovery:
		return SpecProtocolClassification{Protocol: SpecProtocolGoogleDiscovery}
	case SpecKindDropboxStone:
		return SpecProtocolClassification{Protocol: SpecProtocolDropboxStone}
	case SpecKindHumanDocs:
		return SpecProtocolClassification{Protocol: SpecProtocolHumanDocs}
	default:
		return SpecProtocolClassification{Protocol: SpecProtocolUnknown}
	}
}

func specReferenceLooksSwagger(ref SpecReference) bool {
	id := strings.ToLower(ref.ID)
	version := strings.ToLower(strings.TrimSpace(ref.Version))
	note := strings.ToLower(ref.SourceNote)
	return strings.Contains(id, "swagger") ||
		strings.Contains(version, "swagger") ||
		strings.Contains(version, "openapi 2.0") ||
		strings.Contains(note, "swagger") ||
		strings.Contains(note, "openapi 2.0")
}

func inferredOpenAPIVersion(ref SpecReference) string {
	version := strings.ToLower(strings.TrimSpace(ref.Version))
	switch {
	case strings.HasPrefix(version, "3.1"):
		return "3.1"
	case strings.HasPrefix(version, "3.0"):
		return "3.0"
	}
	note := strings.ToLower(ref.SourceNote)
	switch {
	case strings.Contains(note, "openapi 3.1") || strings.Contains(note, "openapi v3.1") || strings.Contains(note, "oas 3.1"):
		return "3.1"
	case strings.Contains(note, "openapi 3.0") || strings.Contains(note, "openapi v3.0") || strings.Contains(note, "oas 3.0"):
		return "3.0"
	case strings.Contains(note, "openapi v3") || strings.Contains(note, "openapi 3") || strings.Contains(note, "oas v3"):
		return "3"
	default:
		return ""
	}
}
