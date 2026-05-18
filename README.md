# apitools

`github.com/OpenUdon/apitools` is a Go library and CLI for OpenAPI document
tooling: discovery, URL-safe download, validation, local file scanning,
importing, caching, operation inventories, operation summaries, auth/security
summaries, and deterministic operation ranking.

The module is intentionally narrow. It handles OpenAPI and Swagger documents as
untrusted data. It does not own workflow semantics, review handoff contracts,
runtime policy, account selection, credential resolution, request signing, or
execution. Downstream products keep those responsibilities in their own
repositories.

## Scope

Keep these responsibilities in `apitools`:

- Discover OpenAPI documents from local project files, URLs, APIs.guru, and
  public-apis probes.
- Safely download OpenAPI or Swagger documents over HTTP(S), with unsafe host
  rejection, redirect limits, request timeouts, and response-size limits.
- Validate OpenAPI 3.0, OpenAPI 3.1, and Swagger 2.0 roots well enough for
  import and authoring workflows.
- Import documents into a local `openapi/` directory with deterministic file
  names.
- Build operation inventories, prompt-safe document summaries, auth/security
  summaries, and request-field summaries.
- Rank operations deterministically from text and structured hints.
- Optionally cache catalog search results and downloaded document bytes.

Keep these responsibilities downstream:

- UWS workflow semantics and public workflow schema live in `../uws`.

`apitools` may describe what an OpenAPI document requires. It must not decide
which production account to use, fetch secrets, sign live requests, or execute
operations.

## CLI

```bash
go run ./cmd/apitools --help
go run ./cmd/apitools search --query slack
go run ./cmd/apitools search --query slack --json
go run ./cmd/apitools search --query slack --cache ~/.cache/apitools/cache.sqlite
go run ./cmd/apitools import --url https://example.com/openapi.yaml --dir ./openapi --name example
go run ./cmd/apitools catalog list
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog overlay-view github
go run ./cmd/apitools catalog security-report
go run ./cmd/apitools catalog check
```

Search uses APIs.guru first and can fall back to public-apis by probing common
OpenAPI and Swagger paths. Imported documents are treated as untrusted data.
This package never executes API operations.

`search` flags:

```bash
go run ./cmd/apitools search \
  --query "support ticket" \
  --limit 10 \
  --source auto \
  --public-probe 25 \
  --probe-timeout 5s \
  --probe-budget 30s \
  --cache ~/.cache/apitools/cache.sqlite \
  --cache-mode read-write \
  --json
```

`import` flags:

```bash
go run ./cmd/apitools import \
  --url https://example.com/openapi.yaml \
  --dir ./openapi \
  --name example \
  --cache ~/.cache/apitools/cache.sqlite \
  --cache-mode read-write \
  --json
```

Cache modes are `read-write`, `refresh`, `offline`, and `bypass`. The
`--offline` flag is shorthand for `--cache-mode offline`.

Catalog commands expose built-in provider metadata and security-overlay status
without fetching provider documents, executing operations, resolving
credentials, or claiming runtime compatibility:

```bash
go run ./cmd/apitools catalog list
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog inspect slack \
  --openapi ./openapi/slack.yaml \
  --security-overlay ./openapi/slack-security.json
go run ./cmd/apitools catalog overlay-view github --json
go run ./cmd/apitools catalog security-report --json
go run ./cmd/apitools catalog check --as-of 2026-05-18 --json
```

Catalog resolution is intentionally conservative. Explicit user OpenAPI inputs
and user security overlays take precedence over project-local documents, and
project-local documents take precedence over built-in spec references and
built-in advisory security overlays. Built-in catalog metadata is a discovery
baseline only; it does not override a team's local API contract.

Catalog curation follows a fixed per-service workflow: try official
OpenAPI/Swagger/Discovery sources first, review auth/security completeness, add
security-overlay metadata when needed, and generate docs-derived advisory
overlays only when no official OpenAPI document exists. The detailed workflow
lives in [docs/catalog-curation.md](docs/catalog-curation.md). Downloaded specs
and local SQLite caches stay ignored under `catalog-openapi-cache/`; generated
advisory overlays and service-specific overlay builders are tracked catalog
assets. For catalog curation, `cache.sqlite` records file paths for saved
documents and overlay artifacts rather than duplicating already-saved document
bodies; the local manifest registration program lives under
`catalog-openapi-cache/artifact-registry/`.

Overlay inspection views are metadata-only. `catalog overlay-view` reports how
built-in security overlays would supplement catalog classifications, preserving
provenance for schemes and security requirements and surfacing review conflicts
such as duplicate scheme names, missing referenced schemes, overlay-only
additions, and unresolved operation matches. It does not write overlay-applied
OpenAPI files.

Catalog quality checks are offline by default. `catalog check` validates
built-in providers, candidates, source notes, verification dates, security
classifications, and overlay references without probing URLs or downloading
provider documents. Error-level findings return a nonzero exit code; warning-only
reports remain exit code 0 for inspection.

## Go Usage

```go
import apitools "github.com/OpenUdon/apitools"

ctx := context.Background()
client := &apitools.Client{}

report, err := client.Search(ctx, apitools.SearchOptions{
	Query:  "slack",
	Source: apitools.SourceAuto,
	Limit:  10,
})
_, _ = report, err
```

Local project directories can be scanned without network access:

```go
results, err := apitools.LocalFiles(ctx, apitools.LocalOptions{
	Dir:     "./openapi",
	BaseDir: ".",
	Query:   "slack messages",
	// MaxBytes: 4 * 1024 * 1024, // optional; defaults to apitools.DefaultMaxBytes
})
_, _ = results, err
```

Prompt-safe OpenAPI operation context is available through inventories and
document summaries:

```go
inventory, err := apitools.BuildOperationInventory(ctx, apitools.InventoryOptions{
	Documents: []apitools.InventoryDocument{{Path: "openapi/support.yaml"}},
	Query:     "create support ticket",
	// MaxBytes: 4 * 1024 * 1024, // optional for path-backed documents
})
docs, err := apitools.BuildAuthoringAPIDocuments(ctx, apitools.AuthoringAPIDocumentOptions{
	Documents: []apitools.InventoryDocument{{Path: "openapi/support.yaml"}},
	Query:     "create support ticket",
})
_, _ = inventory, docs
```

Operation ranking helpers keep API selection deterministic while leaving
runtime policy downstream:

```go
selection := apitools.SelectOperationByHints(apitools.OperationSelectionHints{
	Provider: "aws",
	Action:   "CreateQueue",
	Purpose:  "create",
}, inventory.Operations)
_ = selection
```

Auth helpers summarize credential and configuration requirements without
resolving secrets or signing requests:

```go
requirements := apitools.AuthRequirementsForOperations("stripe", docs[0].Operations)
_ = requirements
```

Provider catalog helpers live in `github.com/OpenUdon/apitools/catalog`:

```go
import "github.com/OpenUdon/apitools/catalog"

resolved, err := catalog.ResolveProvider(catalog.ResolveProviderOptions{
	ProviderKey: "slack",
	UserOpenAPI: "./openapi/slack.yaml",
})
_, _ = resolved, err

securityReport, err := catalog.BuiltInSecurityReport()
_, _ = securityReport, err

view, err := catalog.BuiltInSecurityInspectionView("github")
_, _ = view, err

quality := catalog.BuiltInCatalogQualityReport(catalog.CatalogQualityOptions{})
_ = quality
```

Caching is optional through `github.com/OpenUdon/apitools/sqlitecache` or the
CLI `--cache` flag. Cache modes include `read-write`, `refresh`, `offline`, and
`bypass`.

## Safety Boundary

`apitools` may discover, validate, import, index, and summarize OpenAPI
documents.

It must not:

- Execute API operations or workflows.
- Select production accounts or runtime environments.
- Resolve credentials, acquire tokens, or sign requests.
- Store secrets, workflow execution data, approval state, or runtime state.
- Treat discovered documents as trusted input.

Remote fetches are limited to HTTP(S). Unsafe hosts are rejected by default,
including localhost, private, link-local, multicast, and unspecified addresses.
Callers that intentionally need local fixtures can opt into that behavior with
`Client.AllowUnsafeHosts`.

Local file reads are also fail-closed. `LocalFiles`, `BuildOperationInventory`,
and `LoadOperationIndex` reject symlinked scan roots, symlinked document paths,
symlinked parent components, directories, special files, and files larger than
the resolved byte limit before parsing. Path-backed local reads use bounded I/O;
`LocalOptions.MaxBytes` and `InventoryOptions.MaxBytes` can lower or raise the
limit, and `0` uses `DefaultMaxBytes` (`20 MiB`), matching remote downloads.
In-memory `InventoryDocument.Content` is unchanged. Regular `.json`, `.yaml`,
and `.yml` files that parse but are not OpenAPI or Swagger are still ignored by
`LocalFiles`.

## Packages

- `github.com/OpenUdon/apitools`: core client, discovery, import, validation,
  inventories, authoring summaries, auth summaries, and operation ranking.
- `github.com/OpenUdon/apitools/catalog`: metadata-only candidate inventory,
  durable provider catalog entries, official spec references, security
  overlays, auth/security reports, overlay inspection views, and provider
  resolution helpers for catalog curation. Catalog metadata is not provider or
  n8n runtime compatibility.
- `github.com/OpenUdon/apitools/sqlitecache`: optional SQLite cache
  implementation for the core `Cache` interface.
- `github.com/OpenUdon/apitools/openapidisco`: compatibility wrapper around
  local OpenAPI discovery and primary-candidate selection.

## Development

```bash
go test ./...
go vet ./...
git diff --check
go run ./cmd/apitools search --help
go run ./cmd/apitools import --help
go run ./cmd/apitools catalog check
go run ./cmd/apitools catalog list
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog security-report
```

When changing exported APIs, run dependent checks in sibling consumers when
available:

```bash
(cd ../openudon && go test ./...)
(cd ../udon && go test ./...)
```
