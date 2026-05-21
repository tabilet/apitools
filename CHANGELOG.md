# Changelog

## v0.1.0 - 2026-05-21

Initial tagged release of `github.com/OpenUdon/apitools`.

- OpenAPI/Swagger discovery, safe download, validation, import, local scanning,
  operation inventories, prompt-safe summaries, auth summaries, and operation
  ranking.
- Metadata-only provider catalog with protocol classification, advisory
  security overlays, catalog quality gates, provider advisory reports, refresh
  review reports, stats, and security audits.
- Native source-family support through standalone Google Discovery and AWS
  Smithy parser modules, with deprecated compatibility wrappers retained in
  this module.
- Source-first catalog coverage for OpenAPI/Swagger, Google Discovery, AWS
  Smithy, Dropbox Stone, OpenAPI index, tenant-exported metadata, and reviewed
  docs-derived advisory overlays.
- Safety boundary: no API operation execution, credential resolution, token
  fetching, request signing, account/region selection, or provider runtime
  behavior.
