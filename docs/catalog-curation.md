# Catalog Curation Workflow

This workflow applies to every service or workflow node reviewed for the core
OpenAPI catalog. Catalog data is advisory metadata only: it may describe API
shape and auth requirements, but it must not execute provider operations,
resolve credentials, sign requests, or choose runtime accounts.

## Per-Service Process

For each service, use the same four-step review loop.

1. Try to obtain an official machine-readable API document.

   Prefer provider-owned sources in this order:

   - official OpenAPI or Swagger document;
   - official OpenAPI index that points to provider-owned specs;
   - official non-OpenAPI machine-readable document, such as Google Discovery;
   - official human API documentation when no machine-readable source is found.

   Downloaded official artifacts are local review inputs, not vendored catalog
   data. Store them under ignored paths in `catalog-openapi-cache/`:

   - `catalog-openapi-cache/openapi/` for OpenAPI or Swagger documents;
   - `catalog-openapi-cache/google-discovery/` for Google Discovery documents;
   - `catalog-openapi-cache/cache.sqlite` for optional local cache metadata.

   When the original document is already saved on disk, register the cache row
   with `content_path` instead of duplicating the document bytes in SQLite.
   `content_path` values should normally be relative to
   `catalog-openapi-cache/`, such as `openapi/slack-web-openapi-v2.json`.
   Use `catalog-openapi-cache/artifact-registry/register_catalog_artifacts.go`
   to refresh the local path manifest for the current curated batch.

2. Review auth/security completeness.

   Inspect official docs and the machine-readable document, when one exists.
   If the upstream spec has missing, stale, ambiguous, or incomplete security
   metadata, keep the upstream document unchanged and add catalog security
   overlay metadata with source references. Security overlays must contain only
   credential shape metadata, never credential values.

3. Build a docs-derived advisory overlay when no official OpenAPI exists.

   For human-docs-only services, create a minimal OpenAPI-shaped advisory
   overlay containing both reviewed endpoint coverage and auth/security
   metadata. Mark it explicitly as docs-derived and not official provider
   OpenAPI, for example with an `x-apitools-overlay` object containing:

   - `provider_id`;
   - `overlay_id`;
   - `official_openapi: false`;
   - `derived_from_docs: true`;
   - `source_refs`;
   - `source_note`.

   Store generated overlays as tracked catalog assets under
   `catalog-openapi-cache/advisory-overlays/<provider>-...-overlay.json`.
   Register the overlay path in `cache.sqlite` as a catalog artifact so the
   cache can act as a local manifest of reviewed docs-derived assets.

4. Save the service-specific overlay builder.

   Overlay builders may be specific to one service because documentation shape,
   endpoint coverage, and security conventions differ by provider. Save each
   tracked builder individually under
   `catalog-openapi-cache/overlay-builders/build_<provider>_overlay.go`.

   Required builder conventions:

   - include `//go:build ignore` so `go test ./...` does not treat saved
     builders as package files;
   - use deterministic output and stable JSON indentation;
   - write only into `catalog-openapi-cache/advisory-overlays/`;
   - keep source URLs and source notes in the generated overlay;
   - use no secrets and perform no provider API calls.

## Tracked Changes

The public `apitools` repository should track durable metadata, source-backed
advisory assets, and code:

- `catalog/` provider, spec-reference, and security-overlay metadata;
- `catalog-openapi-cache/advisory-overlays/` docs-derived advisory overlays;
- `catalog-openapi-cache/overlay-builders/` service-specific builders;
- `catalog-openapi-cache/artifact-registry/` local manifest registration
  programs;
- tests that validate catalog entries and overlays;
- README or docs updates for public behavior.

The private harness repository should track memory-bank milestone/status
updates when the catalog process changes.

The public repository should not track downloaded provider specs, Google
Discovery documents, or SQLite caches. Those review artifacts live under
ignored paths in `catalog-openapi-cache/`. The SQLite cache should point at
those files with paths instead of copying their document bodies into BLOB
columns when a file-backed artifact exists.

## Verification

For each reviewed batch, run:

```bash
go test ./...
go vet ./...
git diff --check
go run ./cmd/apitools catalog list
go run ./cmd/apitools catalog security-report
```

For each generated overlay, also run a structural check such as:

```bash
jq '{openapi, title: .info.title, provider: ."x-apitools-overlay".provider_id, paths: (.paths | keys)}' \
  catalog-openapi-cache/advisory-overlays/<provider>-...-overlay.json
```
