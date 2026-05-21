# apitools

`github.com/OpenUdon/apitools` is a Go library and CLI for OpenAPI document
tooling and provider API-source metadata: discovery, URL-safe download,
validation, local file scanning, importing, caching, operation inventories,
operation summaries, auth/security summaries, deterministic operation ranking,
catalog protocol classification, and advisory endpoint overlays.

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
go run ./cmd/apitools catalog advisory slack
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog overlay-view github
go run ./cmd/apitools catalog security-audit
go run ./cmd/apitools catalog security-report
go run ./cmd/apitools catalog check
go run ./cmd/apitools catalog stats
go run ./cmd/apitools catalog refresh-report
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
go run ./cmd/apitools catalog advisory slack \
  --cache catalog-openapi-cache/cache.sqlite
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog inspect slack \
  --openapi ./openapi/slack.yaml \
  --security-overlay ./openapi/slack-security.json
go run ./cmd/apitools catalog overlay-view github --json
go run ./cmd/apitools catalog security-audit --json
go run ./cmd/apitools catalog security-report --json
go run ./cmd/apitools catalog check --as-of 2026-05-18 --json
go run ./cmd/apitools catalog specs --cache catalog-openapi-cache/cache.sqlite
go run ./cmd/apitools catalog stats \
  --cache-dir catalog-openapi-cache \
  --cache catalog-openapi-cache/cache.sqlite \
  --as-of 2026-05-19
go run ./cmd/apitools catalog refresh-report \
  --cache-dir catalog-openapi-cache \
  --cache catalog-openapi-cache/cache.sqlite \
  --as-of 2026-05-19
go run ./cmd/apitools catalog refresh \
  --provider slack \
  --cache-dir catalog-openapi-cache \
  --cache catalog-openapi-cache/cache.sqlite
```

Catalog resolution is intentionally conservative. Explicit user OpenAPI inputs
and user security overlays take precedence over project-local documents, and
project-local documents take precedence over built-in spec references and
built-in advisory security overlays. Built-in catalog metadata is a discovery
baseline only; it does not override a team's local API contract.

Provider advisory output is a read-only summary for operators and downstream
authoring integrations. `catalog advisory [provider]` combines provider
metadata, spec references, user OpenAPI need, auth/security status, overlay IDs,
source notes, manual follow-ups, and optional registered artifact paths from an
existing `cache.sqlite`. It does not create a cache when the file is missing,
fetch remote documents, apply overlays, execute API operations, or resolve
credentials.

Catalog curation follows a fixed per-service workflow: try official
OpenAPI/Swagger/Discovery sources first, review auth/security completeness, add
security-overlay metadata when needed, and, when no official OpenAPI document
exists but official API docs expose usable endpoint instructions, generate a
service-specific docs-derived endpoint overlay asset in addition to any
auth/security overlay metadata. The detailed workflow lives in
[docs/catalog-curation.md](docs/catalog-curation.md). Non-OpenAPI source-family
and protocol-connector boundaries live in
[docs/non-openapi-protocols.md](docs/non-openapi-protocols.md). Downloaded specs
and local SQLite caches stay ignored under `catalog-openapi-cache/`; generated
advisory overlays and service-specific overlay builders are tracked catalog
assets. For catalog curation, `cache.sqlite` records file paths for saved
documents and overlay artifacts rather than duplicating already-saved document
bodies; the local manifest registration program lives under
`catalog-openapi-cache/artifact-registry/`.

Control-plane OpenAPI sources remain metadata-only. Catalog rows such as Docker
Engine may describe local daemon APIs, but `apitools` must not open local
sockets, contact daemon endpoints, encode registry auth headers, or execute
image, container, network, volume, registry, or cluster operations.
Kubernetes is treated the same way: catalog metadata may describe exported
cluster Discovery and OpenAPI artifacts from `/api`, `/apis`, `/openapi/v2`,
and `/openapi/v3`, but `apitools` must not read kubeconfig, service-account
tokens, certificates, or cluster environment variables, and must not contact API
servers.

Enterprise application metadata is also source-first and tenant-safe. NetSuite,
SAP S/4HANA, SAP SuccessFactors, Oracle Fusion Cloud Applications, and Workday
are cataloged from official provider documentation as high-value source
families, but account-specific OpenAPI metadata catalogs, Fusion `/describe`
responses, SAP OData/SOAP artifacts, and Workday WWS/REST metadata must be
user-provided or exported by downstream tooling. `apitools` does not sign in to
enterprise tenants, resolve OAuth clients, fetch tokens, call metadata
endpoints, infer enabled modules, or lower OData/WSDL/SOAP into OpenAPI in this
catalog step.

OpenAPI-first provider rows follow the same metadata-only rule even when a
durable provider-owned spec is available. Adyen, DocuSign, Auth0, Confluence
Cloud, BigCommerce, Cisco Meraki, and Confluent Cloud are cataloged from
official OpenAPI/Swagger documents or provider repositories; Acumatica is
cataloged as tenant-generated Swagger/OpenAPI metadata that must be supplied by
the operator. `apitools` does not create API keys, request OAuth consent, submit
payments, send envelopes, administer networks, inspect stores, choose tenants,
or call streaming/control-plane endpoints.

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

Catalog security audits are offline. `catalog security-audit` classifies each
durable provider by effective auth/security disposition and, when local cache
artifacts are registered, inspects OpenAPI/Swagger `securitySchemes` or
`securityDefinitions` plus root and operation-level `security` requirements.
It reports missing, incomplete, or internally inconsistent artifact security
metadata without fetching provider documents or applying credentials.

Catalog stats are offline. `catalog stats` summarizes primary provider
protocol classifications, local catalog artifact registry counts by kind, and
refresh artifact validation buckets without probing URLs or executing provider
operations.

Catalog spec refresh is opt-in and selected. `catalog specs` lists known
machine-readable built-in spec references without network access, optionally
joining registered local artifact paths from `cache.sqlite`. `catalog
refresh-report` reviews built-in refreshable spec references against existing
SQLite registrations and saved files under `catalog-openapi-cache/` without
creating a cache, downloading documents, or promoting metadata. It reports
missing registrations, missing files, SHA-256 and byte evidence, validation
status, stale verification dates, and deterministic manual follow-ups. It reads
saved artifacts with a bounded local file limit and rejects symlinked artifact
paths. `catalog refresh`
downloads only the selected provider/spec reference using the same safe HTTP(S)
download limits as imports, saves the artifact under ignored
`catalog-openapi-cache/openapi/` or `catalog-openapi-cache/google-discovery/`,
and registers file paths in SQLite instead of storing duplicate document blobs.
Refresh reports are review inputs only: they do not edit provider metadata,
verified dates, tracked advisory overlays, or security classifications.

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

advisory, err := catalog.BuiltInProviderAdvisoryReport(catalog.ProviderAdvisoryOptions{
	ProviderKey: "slack",
})
_, _ = advisory, err

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
  resolution helpers for catalog curation. Catalog metadata is not provider
  runtime compatibility.
- `github.com/OpenUdon/apitools/sqlitecache`: optional SQLite cache
  implementation for the core `Cache` interface.
- `github.com/OpenUdon/apitools/openapidisco`: compatibility wrapper around
  local OpenAPI discovery and primary-candidate selection.
- `github.com/OpenUdon/apitools/googlediscovery`: deprecated compatibility
  wrapper around the standalone `github.com/OpenUdon/googlediscovery` native
  Google Discovery parser.
- `github.com/OpenUdon/apitools/awssmithy`: deprecated compatibility wrapper
  around the standalone `github.com/OpenUdon/awssmithy` native AWS Smithy JSON
  parser.

## Development

```bash
go test ./...
go vet ./...
GOWORK=off go test ./...
GOWORK=off go vet ./...
git diff --check
go run ./cmd/apitools search --help
go run ./cmd/apitools import --help
go run ./cmd/apitools catalog check
go run ./cmd/apitools catalog list
go run ./cmd/apitools catalog advisory slack
go run ./cmd/apitools catalog inspect slack
go run ./cmd/apitools catalog security-audit
go run ./cmd/apitools catalog security-report
go run ./cmd/apitools catalog stats
```

When changing exported APIs, run dependent checks in sibling consumers when
available:

```bash
(cd ../openudon && go test ./...)
(cd ../udon && go test ./...)
```
