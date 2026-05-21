# Non-OpenAPI Protocol Notes

`apitools` records several official server-side API description model families
that are not strict OpenAPI import inputs. These notes define the current
stance for each family.

## Current Classification

| Protocol | Current role | Import stance |
|---|---|---|
| Smithy JSON | Official AWS service model review artifact and native protocol metadata source. | Parsed explicitly through standalone `github.com/OpenUdon/awssmithy.Parse` / `ParseMap`; OpenAPI-shaped conversion has been removed to avoid losing protocol semantics. |
| Google Discovery | Official Google REST API description artifact and native protocol metadata source. | Parsed explicitly through standalone `github.com/OpenUdon/googlediscovery.Parse` / `ParseMap`; OpenAPI-shaped conversion has been removed to avoid treating Discovery as an OpenAPI runtime contract. |
| Dropbox Stone | Official Dropbox API model source and advisory overlay provenance. | Review-only source metadata; no native Stone parser or route lowering is currently planned. |
| OpenAPI index | Provider-owned index of OpenAPI documents. | Review-only index; individual child OpenAPI documents still need explicit selection and validation. |
| Human docs | Official documentation source for docs-derived advisory overlays. | Not importable by itself; endpoint overlays are advisory and source-backed, not provider truth. |

## Parser Priority And Source Decisions

1. **Google Discovery native parsing.**
   Discovery documents are structured JSON, already include method IDs and REST
   paths, and have broad coverage for Google Workspace APIs. The parser
   preserves native service, method, schema, auth-scope, and media-upload
   metadata for downstream protocol-aware generators without producing OpenAPI,
   resolving credentials, making API calls, or automatically promoting catalog
   artifacts.

2. **Smithy JSON native parsing.**
   Smithy models describe AWS service operations and protocol traits. The
   native parser preserves AWS protocol details such as HTTP bindings, greedy
   labels, prefix headers, query-param maps, static Query/JSON protocol fields,
   and service/signing names for downstream protocol-aware generators without
   producing OpenAPI-shaped metadata.

3. **Dropbox Stone no-parser decision.**
   M47 retained Dropbox Stone as official Dropbox machine-readable metadata and
   advisory-overlay provenance, but did not open native parser work. Stone is
   provider-specific, the current catalog value is served by the reviewed
   advisory overlay, and n8n-comparison gaps are better handled as targeted
   overlay expansion if needed.

4. **GraphQL-aware classification.**
   Linear and Monday.com should not get REST-shaped endpoint overlays for now.
   Useful coverage should come from a GraphQL protocol classification and
   schema/introspection workflow rather than a single `POST /graphql` wrapper
   that hides operation semantics.

## Boundary

Parsers and compatibility adapters must treat source documents as untrusted
metadata. They may derive operation inventories, protocol metadata, and auth
summaries, but they must not execute provider operations, resolve credentials,
sign requests, choose accounts or regions, or promote derived output as
official provider OpenAPI.
