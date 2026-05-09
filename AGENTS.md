# AGENTS.md

## Purpose

`apitools` is an OpenAPI tooling module and CLI. It discovers, downloads,
validates, imports, caches, scans, indexes, summarizes, and ranks OpenAPI or
Swagger documents from public catalogs and local files.

## Boundaries

- Keep provider discovery, URL safety, OpenAPI/Swagger validation, local cache,
  operation inventories, prompt-safe OpenAPI summaries, auth/security summaries,
  operation ranking, and CLI behavior here.
- Keep product workflow behavior and execution behavior downstream.
- Put UWS workflow semantics in `../uws`.
- Put private UWS/OpenAPI lowering and runtime execution in `../udon`.
- Put Ramen project templates, examples, review evidence, trusted-runner gates,
  and Symphony policy in `../ramen`.
- Put OpenW8M IaC intent, graph, planning, state, and executor-facing artifacts
  in `../openw8m`.

Rule of thumb:

- If it helps find, validate, import, cache, summarize, or rank OpenAPI
  documents, it belongs here.
- If it owns product workflow behavior or executes anything, it belongs
  downstream.

## Commands

```bash
go test ./...
go vet ./...
git diff --check
go run ./cmd/apitools search --help
go run ./cmd/apitools import --help
```

When changing public APIs, run dependent checks in local sibling consumers when
available:

```bash
(cd ../ramen && go test ./...)
(cd ../udon && go test ./...)
```

## Go Conventions

- Primary language is Go.
- Keep `cmd/apitools` thin; reusable behavior belongs in the root package or a
  focused subpackage such as `sqlitecache`.
- Keep the root package dependency-light. Optional storage integrations should
  live in subpackages.
- Preserve exported API compatibility unless the change is intentionally
  breaking and documented.
- Prefer deterministic tests with `httptest` over live network tests.

## Safety

- Treat all discovered OpenAPI documents as untrusted.
- Never execute operations from a discovered document.
- Enforce HTTP/HTTPS only for remote fetches.
- Reject localhost, private, link-local, multicast, and unspecified addresses by
  default.
- Keep redirect limits, response-size limits, and request timeouts in place.
- Do not cache secrets, tokens, workflow execution data, or sensitive values.
