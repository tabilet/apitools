# Catalog Expansion Queue

Future catalog expansion should remain source-first and batch-frozen before
implementation begins.

## Candidate Inputs

- Current OpenUdon workflow needs.
- Provider-owned OpenAPI, Swagger, OpenAPI index, Smithy, Discovery, Stone, or
  other machine-readable sources.
- Official human docs only when they expose enough endpoint instructions for a
  reviewed advisory overlay.

## Next Batch Shape

Prefer a small batch of 8-12 services with at least one of these properties:

- official OpenAPI/Swagger can be downloaded and validated or classified as
  parseable-invalid review evidence;
- official non-OpenAPI machine specs are available and fit the protocol
  classifications already tracked by the catalog;
- official human docs expose a small REST-shaped subset suitable for an
  advisory endpoint overlay.

Avoid broad platform surfaces until there is a clear source and validation
strategy. GraphQL-first services should wait for GraphQL protocol support
instead of being forced into REST-shaped overlays.

Control-plane API sources need an explicit local-safety boundary even when they
publish normal OpenAPI or Swagger documents. For example, Docker Engine is a
local daemon API: catalog metadata may describe its document, but `apitools`
must not open Docker sockets, contact daemon TCP endpoints, encode registry
auth headers, or perform image/container operations.

Cluster-published API metadata needs the same boundary. Kubernetes clusters
publish Discovery and OpenAPI artifacts at `/api`, `/apis`, `/openapi/v2`, and
`/openapi/v3`, but the exact API groups, versions, aggregated APIs, CRDs, and
security posture are cluster-specific. Catalog rows may point to the official
source model and expect user-exported local artifacts; they must not contact API
servers, read kubeconfig or service-account credentials, perform discovery
calls, watch resources, or claim runtime compatibility with any cluster.

Enterprise application metadata needs account and tenant boundaries even when
official docs describe OpenAPI-shaped metadata. NetSuite metadata catalogs,
Oracle Fusion `/describe` responses, SAP Business Accelerator Hub OData/SOAP
surfaces, and Workday WWS/REST metadata are useful source families, but the
portable catalog row should stop at official source refs, auth posture, and
user-provided metadata guidance unless a future milestone adds a local parser
for exported artifacts.

## Source-First Batch Queue

Freeze future batches only from provider-owned or protocol-owned evidence.
The next 8-12 service batch should be selected from sources that can be
reviewed directly: downloadable OpenAPI/Swagger, Smithy or Discovery models,
official OpenAPI indexes, provider-owned source repositories, or official
human docs that support a small advisory endpoint overlay.

Runtime connector catalogs, workflow product node lists, screenshots,
community examples, and third-party integration code may suggest areas of user
interest, but they must not be copied into `apitools` or used as provider
truth. They are not enough to add provider rows, endpoint overlays, auth
metadata, or native source parsers.

## Required Work Per Service

- Add or update candidate metadata.
- Add durable provider metadata with source refs, availability, user OpenAPI
  need, quirks, and verification date.
- Register downloadable official review artifacts when refresh succeeds or is
  parseable-invalid.
- Add auth/security classification or advisory security overlay metadata.
- Add a docs-derived endpoint overlay only when official docs support a useful
  reviewed subset and the overlay would not misrepresent the API model.
- Run catalog stats, refresh-report, catalog quality, Go tests, vet, and diff
  checks before committing the batch.
