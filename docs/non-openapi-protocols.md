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
| Kubernetes Discovery/OpenAPI | Cluster-published API metadata for enabled groups, versions, resources, aggregated APIs, and CRDs. | User-exported local metadata only for now; no live cluster discovery, kubeconfig reading, credential resolution, or native parser in M54. |
| OData | Enterprise application source family used by SAP S/4HANA and SAP SuccessFactors surfaces. | User-exported or provider metadata only for now; no OData parser, lowering, or generic REST-shaped overlay in M55. |
| WSDL/SOAP | Enterprise application source family used by Workday WWS and some SAP surfaces. | Review-only source metadata for now; no WSDL/SOAP parser, credential resolution, tenant calls, or OpenAPI lowering in M55. |
| Tenant describe and metadata catalogs | Account-specific metadata catalogs such as NetSuite SuiteTalk REST OpenAPI 3.0 metadata, Oracle Fusion `/describe` responses, and Acumatica endpoint Swagger/OpenAPI. | User-provided exported metadata only for now; no live tenant metadata calls, ERP login, or native describe parser in M55/M56. |
| OpenAPI index | Provider-owned index of OpenAPI documents. | Review-only index; individual child OpenAPI documents still need explicit selection and validation. |
| Human docs | Official documentation source for docs-derived advisory overlays. | Not importable by itself; endpoint overlays are advisory and source-backed, not provider truth. |
| Generic protocol connectors | Workflow/runtime connectors for databases, brokers, mail, files, directories, generic HTTP, webhooks, or local execution. | Not provider catalog evidence by themselves; future native source work must start from provider-owned or protocol-owned metadata artifacts. |

## Parser Priority And Source Decisions

Catalog refresh and materialization keep first-class API source families in
source-aligned directories: OpenAPI/Swagger under `openapi/`, Google Discovery
under `google-discovery/`, and AWS Smithy JSON under `aws-smithy/`. Other
review-only families remain metadata-only and materialize under `artifacts/`
when copied for provenance.

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

6. **Kubernetes cluster metadata no-parser decision.**
   M54 records Kubernetes as a cluster-published source family rather than a
   static provider-wide spec. `/api` and `/apis` Discovery outputs and
   `/openapi/v2` or `/openapi/v3` OpenAPI outputs must be supplied as exported
   local metadata by the operator or downstream tooling. No parser is added in
   M54 because OpenAPI import already handles user-provided OpenAPI documents,
   and live discovery would cross the `apitools` boundary.

7. **Enterprise application metadata no-parser decision.**
   M55 records NetSuite, SAP S/4HANA, SAP SuccessFactors, Oracle Fusion Cloud
   Applications, and Workday as official source families without adding parser
   code. NetSuite and Oracle Fusion can expose OpenAPI-shaped metadata only
   through account or tenant endpoints; SAP and Workday include OData and
   WSDL/SOAP semantics that should not be hidden inside generic endpoint
   overlays. Future parser work should start from exported local metadata and
   preserve native protocol fields.

8. **Tenant-generated OpenAPI metadata.**
   M56 records Acumatica as an OpenAPI-adjacent source family but not as a
   built-in downloadable provider spec. Useful Acumatica Swagger/OpenAPI is
   generated per ERP instance, endpoint, tenant, version, and customization, so
   operators or downstream tooling must supply exported local metadata before
   import. `apitools` must not log in to ERP tenants or call endpoint metadata
   URLs.

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
