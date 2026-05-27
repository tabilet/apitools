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
	SpecProtocolAsyncAPI        SpecProtocol = "asyncapi"
	SpecProtocolOpenRPC         SpecProtocol = "openrpc"
	SpecProtocolGraphQL         SpecProtocol = "graphql"
	SpecProtocolGRPCProtobuf    SpecProtocol = "grpc-protobuf"
	SpecProtocolOData           SpecProtocol = "odata"
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
// protocols that can be represented directly in UWS source descriptions. Empty means the
// protocol is not a first-class UWS source type.
func (c SpecProtocolClassification) UWSSourceType() string {
	switch c.Protocol {
	case SpecProtocolOpenAPI, SpecProtocolSwagger:
		return "openapi"
	case SpecProtocolGoogleDiscovery:
		return "google-discovery"
	case SpecProtocolSmithy:
		return "aws-smithy"
	case SpecProtocolAsyncAPI:
		return "asyncapi"
	case SpecProtocolOpenRPC:
		return "openrpc"
	case SpecProtocolGraphQL:
		return "graphql"
	case SpecProtocolGRPCProtobuf:
		return "grpc-protobuf"
	case SpecProtocolOData:
		return "odata"
	default:
		return ""
	}
}

// SourceAlignedArtifactDir returns the preferred local artifact directory for
// first-class UWS source types. Empty means the protocol is not represented as a
// source-aligned catalog artifact.
func (c SpecProtocolClassification) SourceAlignedArtifactDir() string {
	switch c.UWSSourceType() {
	case "openapi":
		return "openapi"
	case "google-discovery":
		return "google-discovery"
	case "aws-smithy":
		return "aws-smithy"
	case "asyncapi":
		return "asyncapi"
	case "openrpc":
		return "openrpc"
	case "graphql":
		return "graphql"
	case "grpc-protobuf":
		return "grpc-protobuf"
	case "odata":
		return "odata"
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
	case SpecKindAsyncAPI:
		return SpecProtocolClassification{Protocol: SpecProtocolAsyncAPI}
	case SpecKindOpenRPC:
		return SpecProtocolClassification{Protocol: SpecProtocolOpenRPC}
	case SpecKindGraphQL:
		return SpecProtocolClassification{Protocol: SpecProtocolGraphQL}
	case SpecKindGRPCProtobuf:
		return SpecProtocolClassification{Protocol: SpecProtocolGRPCProtobuf}
	case SpecKindOData:
		return SpecProtocolClassification{Protocol: SpecProtocolOData}
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
	case SpecKindAsyncAPI:
		return SpecProtocolClassification{Protocol: SpecProtocolAsyncAPI}
	case SpecKindOpenRPC:
		return SpecProtocolClassification{Protocol: SpecProtocolOpenRPC}
	case SpecKindGraphQL:
		return SpecProtocolClassification{Protocol: SpecProtocolGraphQL}
	case SpecKindGRPCProtobuf:
		return SpecProtocolClassification{Protocol: SpecProtocolGRPCProtobuf}
	case SpecKindOData:
		return SpecProtocolClassification{Protocol: SpecProtocolOData}
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
