# Bring icot to openudon by sharing the elicitation engine in apitools

## Context

Today, `ramen` ships a working `icot` subcommand that interactively elicits user requirements via an LLM and writes `intent.hcl`. `openudon` builds its own `intent.hcl` differently — using only Context7 documentation snippets converted into `assumption` blocks, no LLM-driven elicitation.

The user wants openudon to gain the same elicitation experience. Investigation shows the two `intent.hcl` files describe **fundamentally different domains** and cannot share a schema:

- **ramen**: workflow-execution intent — Steps, Triggers, Inputs, Outputs (`udon/pkg/rollout/intent.go:17`).
- **openudon**: infrastructure-as-code intent — Resources, DataSources, Providers, Variables (`openudon/intent/v1alpha1/types.go:29`).

What **is** sharable: the LLM-driven elicitation engine itself — the conversation loop, slot classification, draft persistence, transcript handling, progressive-ICOT integration. That engine currently lives in `ramen/internal/icot/` with hard dependencies on `udon/pkg/runner` for LLM clients and on ramen-internal types for schema rendering.

Additionally, the LLM clients in `udon/pkg/rollout/llm.go` hardcode upstream base URLs (Google, OpenAI, Anthropic), so neither repo can route LLM traffic through the local `copilot-api` proxy we just adopted.

**Intended outcome (three phases):**

1. Move the LLM client factory to `apitools/llm` and add base-URL override env vars so `claude-c`-style Copilot routing works for `intent.hcl` building.
2. Extract the schema-agnostic elicitation engine from `ramen/internal/icot` to `apitools/icot`, behind a renderer interface; ramen becomes a thin consumer.
3. Add an `openudon icot` subcommand that consumes `apitools/icot` and renders into openudon's `intentv1.Intent` schema.

The phases are sequential because each later phase depends on the prior one's API. Each phase ships a green workspace before the next begins.

---

## Phase 1 — Move LLM factory to `apitools/llm`, add base-URL overrides

### Critical files

- **Source** (currently in udon):
  - `udon/pkg/rollout/llm.go` — concrete Anthropic / OpenAI / Gemini HTTP clients, hardcoded base URLs at lines 123, 366, 578
  - `udon/pkg/runner/llm.go` — `NewLLMClientFromEnvWithOptions` factory (called from `ramen/internal/icot/icot.go:1056`)
- **New home**: `apitools/llm/` (new subpackage)
  - `apitools/llm/anthropic.go`, `apitools/llm/openai.go`, `apitools/llm/gemini.go` — concrete clients
  - `apitools/llm/factory.go` — `NewClientFromEnvWithOptions` factory, returning `apitools.ChatClient`
- **Callers to update**:
  - `ramen/internal/icot/icot.go:1056`
  - `udon/cmd/udon/app.go:287` (writes intent.hcl) and any other internal udon callers
  - Any other consumers shown by `grep -rln "rollout.NewLLMClient\|runner.NewLLMClientFromEnv" /home/peter/Workspace/`

### Behavior changes

Each provider's `Chat` and `StructuredChat` methods read a base-URL override env var; if unset, fall back to the current hardcoded URL:

| Provider  | New env var          | Default (current hardcoded value)                        |
|-----------|----------------------|----------------------------------------------------------|
| Anthropic | `ANTHROPIC_BASE_URL` | `https://api.anthropic.com`                              |
| OpenAI    | `OPENAI_BASE_URL`    | `https://api.openai.com`                                 |
| Gemini    | `GEMINI_BASE_URL`    | `https://generativelanguage.googleapis.com`              |

(`ANTHROPIC_BASE_URL` is the standard env var the official Anthropic SDK already honors; matching it lets `claude-c` users get Copilot routing for free.) The path suffix (`/v1/messages`, `/v1/chat/completions`, etc.) stays in the client; only the host portion is overridable.

### Verification

```bash
# in each repo
go vet ./...
go test ./...

# end-to-end: point ramen icot at copilot-api proxy
ANTHROPIC_BASE_URL=http://localhost:4141 \
  ANTHROPIC_API_KEY=dummy \
  ramen-icot author --provider=anthropic --project examples/.../project.md
# expect: copilot-api journal shows /v1/messages POSTs; intent.hcl produced

# default still works
unset ANTHROPIC_BASE_URL
ANTHROPIC_API_KEY=$REAL ramen-icot author ...
# expect: traffic to api.anthropic.com
```

---

## Phase 2 — Light-touch helper extraction to `apitools/icot` + `apitools/openapidisco`

### Premise correction after surveying `ramen/internal/icot/elicitor` (2026-05-07)

Two earlier framings were wrong and have been discarded:

1. **"Extract a schema-agnostic engine behind a `Renderer` interface."** The conversation in `loop.go`/`session.go`/`progressive.go` directly constructs `rollout.Step`/`Input`/`Output` and embeds rollout-shaped JSON schemas in the LLM prompts. A post-elicitation Renderer cannot rescue this — the rollout shape is baked into the questions.
2. **"Move `extractor.go` / `classification.go` / generic-`progressive.go` to `apitools/icot/`."** Closer reading shows these files are also rollout-bound: `Extractor`'s interface signatures are `Session`-typed, the embedded `kickoffSchema`/`draftSchema` describe rollout `Step`/`Trigger`/`Input`/`Output`, `classification.go` walks `[]*rollout.Step` and `[]*rollout.Output`, and `progressive.go` references `rollout.*` 42 times across helpers like `mergeStepsByName`, `operationForStep`, `validateOpenAPIRequestMappings`, etc.

### Why `apitools` already implements the bound-runtime pattern

The uws repo uses a "bound-runtime" pattern (`uws1.Runtime` is a 3-method interface; `uws1.Orchestrator` does the structural work; concrete engines bind their own runtime). **`apitools` already applies this same pattern to icot:**

- `apitools.ProgressiveLoopHooks[S, D, A]` (`apitools/interactive.go:258`) is the engine interface, expressed as a struct of 15 callbacks (Normalize, ApplyOpeningAnswer, Autosave, RankDocuments, DeterministicPrefill, LooksLikeSession, MergeDraft, AfterDraft, DraftResultSummary, OnDraftError, CheckReadiness, Ready, PlanQuestion, ApplyAnswer, FinalConfirm, FinalResultSummary, SaveTranscript).
- `apitools.RunProgressiveICOT[S, D, A]` (`apitools/interactive.go:288`) is the shared structural loop: opening question → optional disambiguation → up-to-N draft+question iterations → final confirmation → transcript persistence.
- `apitools.InteractiveExtractor[S, D]`, `apitools.InteractiveDraftRequest[S, D]`, `apitools.InteractiveQuestion`, `apitools.ReadinessIssue`, `apitools.PromptTurn`, `apitools.PromptEvent`, `apitools.PromptSession`, `apitools.SavePromptTranscript`, `apitools.DecodeJSONBlock` are all already generic / domain-neutral.
- ramen's `runProgressive` (90 lines in `progressive.go:24-114`) is already a thin consumer that binds rollout-shaped functions to those generic hooks.

So no `SlotEngine` interface needs to be invented — the abstraction layer ships in `apitools` already and openudon's Phase 3 will instantiate `ProgressiveLoopHooks[intentv1.Session, IacDoc, IacArtifacts]` with IaC-shaped callbacks the same way ramen does for rollout.

### What actually moves in Phase 2

Genuinely small, because the heavy lifting is already in `apitools`:

| File / package | Lines | Action | Why |
|---|---|---|---|
| `ramen/internal/openapidisco/` | ~40 (incl. tests) | Move to `apitools/openapidisco/` | Pure thin wrapper over `apitools.Discoverer` etc.; promoting it removes the ramen detour |
| `apitools/icot/draft.go` (new) | ~85 | Generic `LoadDraft[T]`/`SaveDraft[T]`/`DraftPath`/`DeleteDraft` | Both engines will use the same `.icot/session.yaml` on-disk convention; today only ramen has it |

Everything else stays in ramen unchanged:

| File | Why it stays |
|---|---|
| `extractor.go` | `Extractor` interface returns `Session`; embedded JSON schemas describe rollout shape |
| `classification.go` | `MappingClassification` struct is generic but slot vocabulary (`steps.X.operation`, `intent.outputs.Y`) is rollout-shaped; the helpers walk `[]*rollout.Step`/`[]*rollout.Output` |
| `transcript.go` | 21-line convenience wrapper; both engines can call `apitools.SavePromptTranscript` directly |
| `progressive.go` | 42 `rollout.*` references across 50+ functions; the generic loop is `apitools.RunProgressiveICOT` |
| `loop.go`, `session.go`, `api.go`, `drift.go`, `replay.go` | rollout-bound throughout |

### Critical files

- New: `apitools/openapidisco/discovery.go` + `discovery_test.go` (moved from `ramen/internal/openapidisco/`)
- New: `apitools/icot/draft.go` + `draft_test.go` (generic `LoadDraft[T]`/`SaveDraft[T]`/`DraftPath`/`DeleteDraft` over an opaque session payload)
- Modified: `ramen/internal/icot/elicitor/draft.go` — keep as a 1-line shim that calls `apitools/icot.LoadDraft[Session]` / `SaveDraft[Session]` (so `loop.go`/`replay.go` don't need import sweeps)
- Modified: ramen import sites that referenced `internal/openapidisco`
- Deleted: `ramen/internal/openapidisco/` (replaced by `apitools/openapidisco/`)

### Verification

```bash
# in apitools — new packages compile and pass their own tests
go vet ./...
go test ./icot/... ./openapidisco/...

# in ramen — must still pass byte-identical for fixture-driven tests
go vet ./...
go test ./...

# byte-identity check for a stable example (capture /tmp/before.hcl from current main first)
ramen-icot author --project examples/.../project.md --output /tmp/after.hcl
diff /tmp/before.hcl /tmp/after.hcl
```

### Out of scope for Phase 2

- Designing or extracting a schema-agnostic conversation engine. `apitools.RunProgressiveICOT` + `apitools.ProgressiveLoopHooks[S, D, A]` already provide it; nothing to invent.
- Touching openudon. Phase 2 is a ramen → apitools relocation that keeps ramen behavior identical.

---

## Phase 3 — Add `openudon icot` subcommand

### How Phase 3 reuses the bound-runtime layer

openudon writes an IaC-shaped icot loop directly against `apitools.RunProgressiveICOT[Session, Doc, Artifacts]` — the same generic loop ramen already uses. The shared surface is exactly what the bound-runtime pattern promises: structural orchestration in `apitools` (opening question → optional disambiguation → up-to-N draft+question iterations → final confirm → transcript), domain logic in openudon's hooks (slot vocabulary, JSON schemas, sanitizers, HCL renderer).

This means openudon does NOT duplicate the conversation pacing, transcript handling, draft persistence, or progressive-question loop. It implements:
- An `intentv1.Session` type (holds `intentv1.Intent` + provenance/classifications)
- An `InteractiveExtractor[Session, IacDoc]` with IaC-shaped prompts and JSON schemas
- The 15 callback fields of `ProgressiveLoopHooks[Session, IacDoc, Artifacts]`, each domain-shaped
- A small `runIcot` glue function (~80 lines, mirroring ramen's `runProgressive`)

### Goal

Insert an LLM-elicitation step before openudon's existing `Draft` flow, producing a richer `intentv1.Intent` than the current Context7-only path:

```
openudon icot author --project project.md --provider gemini
openudon icot author --project project.md --provider anthropic
```

…and end up with `workflows/intent.hcl` populated with elicited Resources, Providers, Bindings, Assumptions.

### Critical files

- New: `openudon/cmd/openudon/icot.go` — subcommand wiring; flag style mirrors `intentcli.Draft` (`--provider`, `--model`, `--temperature`, plus `--advisory-*`)
- New: `openudon/internal/icot/loop.go` — small `runIcot(ctx, in, out, opts)` that mirrors ramen's `runProgressive`: builds the hooks struct, calls `apitools.RunProgressiveICOT`, returns artifacts.
- New: `openudon/internal/icot/session.go` — `Session` type holding `intentv1.Intent` + `Annotations` + `Classifications`.
- New: `openudon/internal/icot/extractor.go` — `InteractiveExtractor[Session, IacDoc]` impl with IaC-shaped kickoff/draft/refine prompts and JSON schemas.
- New: `openudon/internal/icot/hooks.go` — implementations of the 15 `ProgressiveLoopHooks` callbacks (CheckReadiness, PlanQuestion, ApplyAnswer, MergeDraft, DeterministicPrefill, FinalConfirm, …) for the IaC slot vocabulary (`resources.X.type`, `providers.Y.config`, `bindings.Z.source`, …).
- New: `openudon/internal/icot/api.go` — OpenAPI prompt context for IaC operations using `apitools/openapidisco`.
- Reused as-is from apitools: `apitools.RunProgressiveICOT`, `apitools.ProgressiveLoopHooks`, `apitools.InteractiveExtractor`, `apitools.InteractiveDraftRequest`, `apitools.InteractiveQuestion`, `apitools.ReadinessIssue`, `apitools.PromptTurn`, `apitools.PromptEvent`, `apitools.SavePromptTranscript`, `apitools.DecodeJSONBlock`, `apitools/icot.LoadDraft`/`SaveDraft` (Phase 2), `apitools/openapidisco` (Phase 2), `apitools/llm.NewClientFromEnvWithOptions` (Phase 1).
- Modified: `openudon/internal/intentcli/build.go` (`writeIntentHCL`) — reused for persistence
- Modified: `openudon/internal/intentcli/options.go` — add `--provider/--model/--temperature`
- Modified: `openudon/internal/intentcli/commands.go` — wire the optional `--elicit` chain between `fetchAdvisoryContext` and `buildIntent`

### Reused vs new (summary)

| Layer | Source |
|---|---|
| LLM client / structured chat / base-URL routing | `apitools/llm` (Phase 1) |
| Generic conversation loop, hook contract, transcript, draft requests, JSON-block decode | `apitools` directly (already generic) |
| `.icot/session.yaml` load/save | `apitools/icot` (Phase 2) |
| OpenAPI discovery | `apitools/openapidisco` (Phase 2) |
| IaC slot vocabulary, JSON schemas, sanitizers, HCL renderer wiring | new in `openudon/internal/icot/` (Phase 3) |
| Intent persistence | existing `openudon/internal/intentcli.writeIntentHCL` |

### Verification

```bash
# in openudon
go vet ./...
go test ./...

# end-to-end with copilot-api routing (validates Phase 1 + Phase 2 + Phase 3 together)
ANTHROPIC_BASE_URL=http://localhost:4141 \
  ANTHROPIC_API_KEY=dummy \
  CONTEXT7_API_KEY=$CTX7 \
  openudon icot author --project examples/aws-bucket/project.md --provider anthropic

# expect: workflows/intent.hcl with non-empty resources[], providers[],
#         bindings[], plus assumptions[] from advisory context
intentv1 validate workflows/intent.hcl   # passes
```

---

## Out of scope

- **Schema unification.** Confirmed by the survey: the rollout schema is woven into the conversation loop, not a render-time concern.
- **Inventing a new icot abstraction layer.** The bound-runtime pattern is already implemented in `apitools.RunProgressiveICOT[S, D, A]` + `ProgressiveLoopHooks[S, D, A]`. There is no `SlotEngine` interface to design — both engines bind hooks to the existing generic loop.
- **A new third domain.** No new "apitools intent" type. `apitools/icot` exposes draft persistence helpers, not a schema.
- **Renaming `rollout.Intent`.** Stays in `udon/pkg/rollout` as ramen's domain type.
- **Rewriting openudon's existing `Draft` advisory path.** Phase 3 adds a parallel command; the existing `openudon build` flow is preserved.
- **Cross-repo `go.mod` pseudo-version bumps beyond what each phase requires.** Done at the end of each phase, not as a separate sweep.

---

## Phase ordering rationale

Phase 1 first: pure code move + small additive change, low risk; later phases naturally import `apitools/llm`; independently delivers Copilot-routing.

Phase 2 next: small relocations only — `openapidisco` package and a generic `LoadDraft[T]`/`SaveDraft[T]` helper. The substantive abstraction layer (the bound-runtime pattern adopted from `uws1.Runtime`) already ships in `apitools.RunProgressiveICOT[S, D, A]` + `ProgressiveLoopHooks[S, D, A]`, so there is no engine to extract. ramen behavior stays byte-identical (verified against fixtures).

Phase 3 last: openudon writes IaC-shaped `ProgressiveLoopHooks[Session, IacDoc, Artifacts]` and an IaC-shaped `InteractiveExtractor`, then calls `apitools.RunProgressiveICOT` directly — the same pattern ramen uses. Each engine binds its domain at execution time; the shared structural loop and persistence helpers come from `apitools`.

If a phase fails verification, stop and resolve before moving on. The workspace must be green between phases (`go vet ./...` and `go test ./...` clean across apitools, openudon, uws, udon, ramen, and the supporting modules they depend on).

---

## Phase 1 status (as of 2026-05-07)

Phase 1 is functionally complete:

- `apitools/llm/` ships `client.go` (provider clients with base-URL overrides and neutral structured-output naming) + `factory.go` (env-driven factory) + `client_test.go` and `coverage_test.go` (full provider coverage including base-URL routing through `httptest`).
- `udon/pkg/rollout/llm.go` is now a ~90-line shim re-exporting every previously-public symbol from `apitools/llm`. Existing `rollout.*` consumers in `udon` and `ramen` compile unchanged.
- `udon/pkg/runner/llm.go` already delegates to `rollout` constructors and so transitively rides on `apitools/llm`. An explicit re-route to `apitools/llm.NewClientFromEnvWithOptions` is deferred until callers are migrated to the new factory directly.
- Tracked in `udon/memory-bank/milestone.md` and `status.md` under M6.

---

## Phase 2 status (as of 2026-05-07)

Phase 2 is complete:

- `apitools/openapidisco/` is the canonical home for the discovery wrapper; `discovery.go` + `discovery_test.go` were promoted out of `ramen/internal/openapidisco/`.
- `apitools/icot/draft.go` ships generic `LoadDraft[T]` / `SaveDraft[T]` / `DraftPath` / `DeleteDraft` over an opaque session payload, with `draft_test.go` covering atomic writes (0o600 perm, sibling-tmp rename), `.icot/` directory pruning, and the invariant that draft cleanup never removes the caller's example/workspace directory.
- `ramen/internal/openapidisco/discovery.go` and `ramen/internal/icot/elicitor/draft.go` remain as type-alias / forwarding shims so existing ramen call sites compile unchanged. `ramen/internal/icot/elicitor/draft.go` keeps the `LooksLikeSession` gate after `apicot.LoadDraft` so fixture-driven behavior is byte-identical.
- Verified: `go vet ./... && go test ./...` clean in apitools and ramen.

## Phase 3 status (as of 2026-05-07)

Phase 3 is complete:

- `openudon/internal/icot/` ships an IaC-shaped binding to `apitools.RunProgressiveICOT[Session, APIDocument, Artifacts]`, split across `session.go` (types + `Normalize` + `LooksLikeSession` + merge), `hooks.go` (slot vocabulary + `CheckReadiness` / `Ready` / `PlanQuestion` / `ApplyAnswer`), `parse.go` (slot parsers), `extractor.go` (optional `apitools/llm` chat extractor), and `loop.go` (~110-line glue that mirrors ramen's `runProgressive`).
- IaC slot vocabulary: `intent.goal`, `intent.workspace`, `intent.openapi_refs`, `intent.providers`, `intent.bindings`, `intent.resources`, `resources[N].operations`, and `data_sources[N].operations`. The deterministic walk produces a valid `intentv1.Intent` rendered through `intentv1.RenderHCL`; explicit LLM flags can draft resources or read-only data sources before the same validation path.
- `openudon intent icot` subcommand wired through `internal/intentcli/{cli,options,commands}.go` (`Icot()` calls `icot.Run`, writes intent HCL, optionally renders `.tf`). It accepts `--project`, `--provider`, `--model`, `--temperature`, and `--no-llm`; provider/model/temperature flags enable the LLM extractor and use env keys such as `GEMINI_API_KEY`.
- `openudon/go.mod` keeps a normal public `github.com/OpenUdon/apitools` requirement without a committed local `replace`; local sibling development uses a `go.work` file outside the module.
- Test coverage in `openudon/internal/icot/loop_test.go`: full slot walk, slot parsing, readiness reporting, LLM draft merge for a read-only weather data source, `LoadDraft` recovery (pre-written draft drives the loop), `DeleteDraft` on success (draft file + `.icot/` pruned), and autosave-during-loop (cancel preserves the draft for resume). `internal/intentcli` also covers a fake Gemini `--project --provider gemini` run that writes loadable `intent.hcl`.
- Verified: `go vet ./... && go test ./...` clean in openudon, with apitools and ramen still green.
- `NoLLM=true` remains the default so local deterministic authoring works without provider credentials. The LLM path is opt-in and only drafts symbolic public intent fields before normal OpenUdon validation.

## Post-Phase-3 cleanup (2026-05-07)

- Extracted the duplicated `writeFileAtomic` implementation (formerly in both `apitools/icot/draft.go` and `apitools/interactive.go`) into a single `apitools/internal/atomicfile.Write`. Both former call sites now delegate to it. No behavior change.
- All four repos green after the extraction (`apitools`, `openudon`, `ramen`; `udon` shim packages `pkg/rollout` and `pkg/runner` still green — see note below).

## Pre-existing, unrelated to this migration

`udon` has failing tests in `cmd/udon`, `generator`, and `internal/uwsbridge` (`unsupported expression type *light.Expression_Fcexpr/Boexpr` errors during workflow compilation). These were introduced by the `github.com/tabilet/* → github.com/OpenUdon/*` rename in commit `9d89b4e` and are independent of Phases 1–3. The shim-affected packages (`pkg/rollout`, `pkg/runner`) both pass.
