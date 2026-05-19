# Non-OpenAPI Protocol Notes

`apitools` records several official server-side API description model families
that are not strict OpenAPI import inputs. These notes define the current
stance for each family.

## Current Classification

| Protocol | Current role | Import stance |
|---|---|---|
| Smithy JSON | Official AWS service model review artifact. | Review-only until an explicit Smithy-to-OpenAPI or Smithy-native inventory path exists. |
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

2. **Smithy JSON inventory or OpenAPI-shaped lowering.**
   Smithy models describe AWS service operations and protocol traits, but AWS
   SigV4 and endpoint rules must remain advisory metadata. A converter should
   not sign requests, choose accounts, or infer runtime regions.

3. **Dropbox Stone parsing.**
   Stone can provide useful route and schema metadata for Dropbox, but it needs
   a dedicated parser or pinned upstream tooling review before durable lowering.

4. **GraphQL-aware classification.**
   Linear and Monday.com should not get REST-shaped endpoint overlays for now.
   Useful coverage should come from a GraphQL protocol classification and
   schema/introspection workflow rather than a single `POST /graphql` wrapper
   that hides operation semantics.

## Boundary

Converters must treat source documents as untrusted metadata. They may derive
operation inventories and auth summaries, but they must not execute provider
operations, resolve credentials, sign requests, choose accounts, or promote
converted output as official provider OpenAPI.
