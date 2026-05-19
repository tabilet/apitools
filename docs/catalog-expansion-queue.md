# Catalog Expansion Queue

Future catalog expansion should remain source-first and batch-frozen before
implementation begins.

## Candidate Inputs

- Current OpenUdon workflow needs.
- n8n node presence as a priority signal only, not runtime compatibility
  evidence.
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
