# apitools

`github.com/tabilet/apitools` is two related things:

1. A CLI and Go library for finding, validating, caching, importing, and
   indexing OpenAPI documents.
2. An upstream intent and iCoT authoring library for tools that draft
   OpenAPI-backed artifacts while providing their own runtime, review, approval,
   and execution behavior.

[![GoDoc](https://godoc.org/github.com/tabilet/apitools?status.svg)](https://godoc.org/github.com/tabilet/apitools)

The module path is now:

```go
github.com/tabilet/apitools
```

## CLI

Install:

```bash
go install github.com/tabilet/apitools/cmd/apitools@latest
```

Run from a checkout:

```bash
go run ./cmd/apitools --help
```

Common commands:

```bash
go run ./cmd/apitools search --query slack
go run ./cmd/apitools search --query slack --json
go run ./cmd/apitools search --query slack --cache ~/.cache/apitools/cache.sqlite
go run ./cmd/apitools import --url https://example.com/openapi.yaml --dir ./openapi --name example
```

Search uses APIs.guru first and can fall back to public-apis by probing common
OpenAPI and Swagger paths. Imported documents are treated as untrusted data:
this package downloads, validates, indexes, and writes OpenAPI documents. It
does not execute APIs or workflows.

## Search Library

```go
import apitools "github.com/tabilet/apitools"

ctx := context.Background()
client := &apitools.Client{}

report, err := client.Search(ctx, apitools.SearchOptions{
	Query:  "slack",
	Source: apitools.SourceAuto,
	Limit:  10,
})
_, _ = report, err
```

Local project directories can be searched without network access:

```go
import apitools "github.com/tabilet/apitools"

results, err := apitools.LocalFiles(ctx, apitools.LocalOptions{
	Dir:     "./openapi",
	BaseDir: ".",
	Query:   "slack messages",
})
_, _ = results, err
```

Caching is optional through `github.com/tabilet/apitools/sqlitecache` or
the CLI `--cache` flag. Cache modes include `read-write`, `refresh`, `offline`,
and `bypass`.

## Intent And ICOT Library

The authoring side provides shared structs and control flow for downstream tools
that want common OpenAPI/UWS drafting behavior but must own their own runtime.
It includes:

- operation inventories, summaries, and deterministic operation selection
- prompt-safe OpenAPI context
- artifact sets, diagnostics, readiness issues, slots, assumptions, and
  symbolic bindings
- credential-value scans and binding audits
- chat JSON fallback helpers
- prompt sessions, transcripts, replay helpers, and progressive iCoT loop
  control
- review-only leaf adapters, public review handoff manifests, and artifact
  writing helpers

Downstream packages implement runtime-specific interfaces such as chat clients,
parsers, renderers, validators, refiners, interactive extractors, approval
gates, binders, and executors.

```go
core := apitools.NewAuthoringCore()

opctx, artifacts, err := apitools.DraftFromOpenAPI(
	ctx,
	core,
	apitools.Brief{
		Text:        "Create one support ticket from runtime inputs.",
		ProjectName: "Support Ticket Draft",
	},
	[]apitools.OpenAPIDoc{{Path: "openapi/support.yaml"}},
	[]string{"createTicket"},
)
_, _, _ = opctx, artifacts, err

leaf := apitools.NewLeafAdapter(artifacts, apitools.LeafOptions{
	Name:   "Support Ticket Draft",
	Source: "example",
})
review := leaf.ReviewMarkdown()
handoff := leaf.ReviewHandoff(apitools.ReviewHandoffOptions{
	ExecutionPolicy: apitools.DefaultReviewExecutionPolicy(true),
})
_, _ = review, handoff
```

For runtime implementers, see
[docs/runtime-integration.md](docs/runtime-integration.md). For the authoring
model and binding terminology, see [docs/authoring.md](docs/authoring.md).

## Safety Boundary

`apitools` is upstream shared infrastructure. It may discover, index, draft,
validate, scan, summarize, and produce review evidence. It must not resolve
concrete credentials, select production accounts, bypass caller review, or
execute side-effectful workflows.

Runtime packages such as Ramen and OpenUdon import and compose the shared
structs and helpers, then supply product-specific validation, review, approval,
persistence, binding, and execution.

## Development

```bash
go test ./...
go vet ./...
git diff --check
```
