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
| Generic protocol connectors | Workflow/runtime connectors for databases, brokers, mail, files, directories, generic HTTP, webhooks, or local execution. | Not provider catalog evidence by themselves; future native source work must start from provider-owned or protocol-owned metadata artifacts. |

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
   provider-specific, and the current catalog value is served by the reviewed
   advisory overlay. Future expansion should be targeted source-backed overlay
   work if needed.

4. **GraphQL-aware classification.**
   Linear and Monday.com should not get REST-shaped endpoint overlays for now.
   Useful coverage should come from a GraphQL protocol classification and
   schema/introspection workflow rather than a single `POST /graphql` wrapper
   that hides operation semantics.

5. **Generic protocol connector boundary.**
   Runtime workflow connectors and local execution utilities are not
   provider-owned API-source evidence. Do not promote connector names into
   provider catalog rows, docs-derived OpenAPI overlays, or native parser work
   without an explicit provider-owned or protocol-owned source artifact.

## Generic Protocol Connector Families

The following connector families are not first-class provider
OpenAPI/Smithy/Discovery catalog targets by name alone:

| Family | Examples | Current stance |
|---|---|---|
| Database and key-value protocols | PostgreSQL, MySQL, Oracle, Redis, MongoDB wire clients | Do not map protocol clients to provider API entries or docs-derived OpenAPI overlays. Service-specific admin APIs, such as a hosted database control-plane API, need their own source-backed milestone. |
| Message and broker protocols | AMQP, Kafka, MQTT, RabbitMQ wire clients | Do not model broker operations as OpenAPI provider metadata. Future support should start from a stable machine-readable protocol or event contract source. |
| Mail, file, and directory protocols | IMAP, SMTP, FTP/SFTP, LDAP | Keep outside the provider catalog unless a future source-family milestone defines metadata-only protocol documents and boundary checks. |
| Generic execution and HTTP utility connectors | Local command execution, generic HTTP clients, webhooks, GraphQL wrappers | Treat as workflow construction tools, not provider sources. GraphQL remains a possible future protocol family when backed by schema artifacts, not by a generic connector. |

This boundary is intentionally separate from provider-specific API curation.
For example, a MongoDB wire-protocol connector is not evidence for the MongoDB
Atlas Admin API, and an AMQP/RabbitMQ connector is not evidence for the
RabbitMQ Management HTTP API. Those APIs can still be reviewed in a future
catalog milestone if they have provider-owned OpenAPI, official human docs
suitable for a reviewed advisory overlay, or another explicit source model.

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
