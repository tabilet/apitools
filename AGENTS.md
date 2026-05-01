# AGENTS.md

## Purpose

`apitools` is an open-source Go module and CLI-backed library for discovering,
validating, caching, and importing OpenAPI or Swagger documents from public
catalogs. It also provides the public OpenAPI-backed authoring substrate for
AI-assisted workflows that turn natural-language briefs and prompt-safe
operation context into draft `project.md` and `intent.hcl` artifacts.

The package is intentionally generic. It is shared by private consumers such as
Ramen and public consumers such as OpenUdon, but it must not contain Ramen
workflow semantics, OpenUdon IaC semantics, udon runtime behavior,
product-specific policy, or private infrastructure assumptions.

## Boundaries

- Keep provider discovery, URL safety, OpenAPI/Swagger validation, local cache,
  operation inventories, ranking, prompt-safe summaries, and CLI behavior in
  this repository.
- Keep generic AI-assisted authoring primitives here when they are
  domain-neutral: common fields, methods, interfaces, transcripts, diagnostics,
  slots, assumptions, symbolic bindings, readiness issues, question plans, and
  draft artifact flow for `project.md` and `intent.hcl`.
- Put Ramen-specific `project.md`, workflow `intent.hcl`, `workflow.hcl`, UWS
  generation, Symphony review packages, trusted-runner gates, evals, example
  artifacts, and private udon integration in Ramen.
- Put concrete OpenUdon IaC intent models, Terraform generation, graph/profile
  planning, state/drift/handoff bundles, and w8m-facing public IaC artifacts in
  OpenUdon.
- Put UWS/OpenAPI execution, lowering, concrete execution, credential
  resolution, runtime binding, and trusted runner behavior in udon or the
  calling runtime.
- Put public UWS semantics in `../uws`.
- Do not add product-specific LLM synthesis, API execution, credential
  injection, or production side effects here.

Rule of thumb:

- If it helps find or safely import OpenAPI documents, it belongs here.
- If it helps author prompt-safe, OpenAPI-backed draft artifacts without
  product-specific semantics, it belongs here.
- If it executes an API or workflow, it does not belong here.
- If it depends on a private repo or product workflow, it does not belong here.

Ramen and OpenUdon depend on or embed `apitools` concepts directly. Ramen
does not inherit from OpenUdon, and OpenUdon does not inherit from Ramen.
Downstreams may inherit common behavior at runtime by embedding concrete
`apitools` review/deferred-leaf helpers or composing shared functions, but
that inheritance must stay domain-neutral: the shared object can carry
artifacts, symbolic binding names, diagnostics, readiness issues, review
scaffolding, and safe filesystem behavior, while the downstream leaf keeps its
own rendering, validation, approval, bundling, and execution policy.

## Architecture

OpenAPI owns methods, paths, schemas, servers, and security declarations.
`apitools` owns discovery, validation, import, inventories, prompt-safe
metadata, and domain-neutral authoring loops over that OpenAPI context.

Authoring follows this shared flow:

```text
brief + OpenAPI docs -> prompt-safe operation context -> project.md / intent.hcl draft -> caller-specific leaf renderer
```

Generated `project.md` and `intent.hcl` outputs are drafts. Callers are
responsible for validating, reviewing, and rendering them into their own leaf
artifacts.

### Binding happens at execution time

`apitools` may name symbolic bindings and describe the contract a caller
must satisfy, but it does not resolve credentials, choose concrete accounts, or
execute operations. Specialized engines bind runtime implementations and leaf
adapters only when they validate and execute their own artifacts.

This is an inheritance/composition rule as much as a safety rule. Shared
authoring objects should be embeddable so Ramen, OpenUdon, and other callers can
reuse common artifact, review, readiness, binding-audit, and safe-write logic at
runtime without inheriting each other's product semantics. Runtime inheritance
must flow from `apitools` into caller-owned leaves; it must not create a
dependency from `apitools` back into Ramen, OpenUdon, udon, w8m, private
credential resolvers, or trusted runners.

## Commands

```bash
go test ./...
go vet ./...
git diff --check
go run ./cmd/openapisearch search --query slack
go run ./cmd/openapisearch search --query slack --cache /tmp/apitools.sqlite
go run ./cmd/openapisearch import --url https://example.com/openapi.yaml --dir /tmp/openapi
```

When changing public APIs, also run dependent checks in local sibling consumers
when available:

```bash
(cd ../ramen && go test ./...)
(cd ../udon && go test ./...)
```

## Go Conventions

- Primary language is Go.
- Keep `cmd/openapisearch` thin; reusable behavior belongs in the root package
  or a focused subpackage such as `sqlitecache`.
- Keep the root package dependency-light. Optional storage integrations should
  live in subpackages.
- Preserve exported API compatibility unless the change is intentionally
  breaking and documented.
- Prefer deterministic tests with `httptest` over live network tests.

## Safety

- Treat all discovered OpenAPI documents as untrusted.
- Never execute operations from a discovered document.
- Enforce HTTP/HTTPS only for remote fetches.
- Reject localhost, private, link-local, multicast, and unspecified addresses
  by default.
- Keep redirect limits, response-size limits, and request timeouts in place.
- Do not cache secrets, credentials, tokens, or workflow execution data.
- Do not add HTML scraping or LLM-synthesized specs without explicit review and
  provenance labeling.
