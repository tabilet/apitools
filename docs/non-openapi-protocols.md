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
| n8n protocol connector nodes | n8n node roots that indicate common protocol families rather than provider-owned API-description sources. | Priority signal only; not catalog provider rows, OpenAPI overlays, native source parsers, or runtime compatibility evidence. |

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

5. **n8n protocol connector boundary.**
   The n8n tree includes generic protocol connectors and local execution
   utilities that are valuable workflow signals but not provider API-source
   evidence. Do not promote those nodes into provider catalog rows, docs-derived
   OpenAPI overlays, or native parser work just because the node directory
   exists.

## n8n Protocol Connector Nodes

The following n8n node roots are currently classified as protocol or generic
connector signals, not first-class provider OpenAPI/Smithy/Discovery catalog
targets:

| Family | n8n node roots | Current stance |
|---|---|---|
| Database and key-value protocols | `CrateDb`, `MongoDb`, `MySql`, `Oracle`, `Postgres`, `QuestDb`, `Redis`, `TimescaleDb` | Do not map protocol clients to provider API entries or docs-derived OpenAPI overlays. Service-specific admin APIs, such as a hosted database control-plane API, need their own source-backed milestone. |
| Message and broker protocols | `Amqp`, `Kafka`, `Mqtt`, `RabbitMQ` | Do not model broker operations as OpenAPI provider metadata. Future support should start from a stable machine-readable protocol or event contract source, not from n8n runtime node behavior. |
| Mail, file, and directory protocols | `EmailReadImap`, `EmailSend`, `Ftp`, `Ldap` | Keep outside the provider catalog unless a future source-family milestone defines metadata-only protocol documents and boundary checks. |
| Generic execution and HTTP utility nodes | `Code`, `ExecuteCommand`, `GraphQL`, `HttpRequest`, `Webhook` | Treat as workflow construction tools, not provider sources. GraphQL remains a possible future protocol family when backed by schema artifacts, not by a generic n8n node. |

This boundary is intentionally separate from provider-specific API curation.
For example, the `MongoDb` node is not evidence for the MongoDB Atlas Admin
API, and the `RabbitMQ` node is not evidence for the RabbitMQ Management HTTP
API. Those APIs can still be reviewed in a future catalog milestone if they
have provider-owned OpenAPI, official human docs suitable for a reviewed
advisory overlay, or another explicit source model.

Future native protocol-family work should meet all of these criteria before it
enters `apitools`:

- A stable, reusable metadata artifact exists, such as GraphQL SDL, AsyncAPI,
  database schema dumps, or another reviewed protocol schema format.
- The implementation can parse local or downloaded metadata without opening
  sockets to live services, introspecting running databases, or reading runtime
  credentials.
- The output is clearly labeled as native protocol metadata or an advisory
  lowering, not provider-owned OpenAPI truth.
- Downstream runtime behavior, credential binding, host selection, queue/topic
  selection, database selection, and account policy remain outside `apitools`.

## Boundary

Parsers and compatibility adapters must treat source documents as untrusted
metadata. They may derive operation inventories, protocol metadata, and auth
summaries, but they must not execute provider operations, resolve credentials,
sign requests, choose accounts or regions, or promote derived output as
official provider OpenAPI.
