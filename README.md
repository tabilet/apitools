# apitools

`github.com/OpenUdon/apitools` is a Go library and CLI for OpenAPI document
tooling: discovery, URL-safe download, validation, local file scanning,
importing, caching, operation inventories, operation summaries, auth/security
summaries, and deterministic operation ranking.

It does not own product workflow behavior or execution. Downstream products keep
non-OpenAPI behavior in their own repositories.

## CLI

```bash
go run ./cmd/apitools --help
go run ./cmd/apitools search --query slack
go run ./cmd/apitools search --query slack --json
go run ./cmd/apitools search --query slack --cache ~/.cache/apitools/cache.sqlite
go run ./cmd/apitools import --url https://example.com/openapi.yaml --dir ./openapi --name example
```

Search uses APIs.guru first and can fall back to public-apis by probing common
OpenAPI and Swagger paths. Imported documents are treated as untrusted data.
This package never executes API operations.

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
})
_, _ = results, err
```

Prompt-safe OpenAPI operation context is available through inventories and
document summaries:

```go
inventory, err := apitools.BuildOperationInventory(ctx, apitools.InventoryOptions{
	Documents: []apitools.InventoryDocument{{Path: "openapi/support.yaml"}},
	Query:     "create support ticket",
})
docs, err := apitools.BuildAuthoringAPIDocuments(ctx, apitools.AuthoringAPIDocumentOptions{
	Documents: []apitools.InventoryDocument{{Path: "openapi/support.yaml"}},
	Query:     "create support ticket",
})
_, _ = inventory, docs
```

Caching is optional through `github.com/OpenUdon/apitools/sqlitecache` or the
CLI `--cache` flag. Cache modes include `read-write`, `refresh`, `offline`, and
`bypass`.

## Safety Boundary

`apitools` may discover, validate, import, index, and summarize OpenAPI
documents. It must not select production accounts or execute side-effectful
workflows.

## Development

```bash
go test ./...
go vet ./...
git diff --check
go run ./cmd/apitools search --help
go run ./cmd/apitools import --help
```
