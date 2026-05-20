# Non-OpenAPI Protocol Notes

`apitools` records several official server-side API description model families
that are not strict OpenAPI import inputs. These notes define the current
stance for each family.

## Current Classification

| Protocol | Current role | Import stance |
|---|---|---|
| Smithy JSON | Official AWS service model review artifact and native protocol metadata source. | Parsed explicitly through `awssmithy.Parse` / `ParseMap`; deprecated OpenAPI-shaped conversion remains compatibility metadata only. |
| Google Discovery | Official Google REST API description artifact. | Best first candidate for a converter because it already describes REST resources, methods, parameters, request bodies, and schemas. |
| Dropbox Stone | Official Dropbox API model source. | Review-only until Stone schema parsing and route lowering are implemented. |
| OpenAPI index | Provider-owned index of OpenAPI documents. | Review-only index; individual child OpenAPI documents still need explicit selection and validation. |
| Human docs | Official documentation source for docs-derived advisory overlays. | Not importable by itself; endpoint overlays are advisory and source-backed, not provider truth. |

## Converter Priority

1. **Google Discovery to OpenAPI-shaped metadata.**
   Discovery documents are structured JSON, already include method IDs and REST
   paths, and have broad coverage for Google Workspace APIs. A converter should
   produce reviewable OpenAPI-shaped metadata without credentials, API calls, or
   automatic catalog promotion.

2. **Smithy JSON native parsing.**
   Smithy models describe AWS service operations and protocol traits. The
   native parser preserves AWS protocol details such as HTTP bindings, greedy
   labels, prefix headers, query-param maps, static Query/JSON protocol fields,
   and service/signing names for downstream protocol-aware generators.
   Deprecated OpenAPI-shaped lowering remains derived compatibility metadata,
   not a runtime contract.

3. **Dropbox Stone parsing.**
   Stone can provide useful route and schema metadata for Dropbox, but it needs
   a dedicated parser or pinned upstream tooling review before durable lowering.

4. **GraphQL-aware classification.**
   Linear and Monday.com should not get REST-shaped endpoint overlays for now.
   Useful coverage should come from a GraphQL protocol classification and
   schema/introspection workflow rather than a single `POST /graphql` wrapper
   that hides operation semantics.

## Boundary

Converters and parsers must treat source documents as untrusted metadata. They
may derive operation inventories, protocol metadata, and auth summaries, but
they must not execute provider operations, resolve credentials, sign requests,
choose accounts or regions, or promote converted output as official provider
OpenAPI.
