# Runtime Integration Guide

`github.com/tabilet/apitools` provides shared OpenAPI search, intent authoring,
iCoT loop, and review helpers. The Go package name is still `apitools`.
A downstream runtime imports those helpers and supplies the product-specific
behavior that cannot live upstream.

The dependency direction should stay simple:

```text
github.com/tabilet/apitools
  shared structs, interfaces, scans, selection, prompt loops, review helpers

runtime package
  imports apitools
  implements chat, parse, render, validate, review, bind, approve, execute
```

Upstream tests should use fakes and contract tests. They should not import a
downstream runtime such as Ramen or OpenUdon.

## What Upstream Owns

Use `apitools` for behavior that is neutral across runtimes:

- OpenAPI search, import, validation, local discovery, caching, and operation
  inventory.
- Prompt-safe operation summaries and operation selection.
- Artifact sets, diagnostics, readiness issues, slots, assumptions, symbolic
  bindings, transcripts, and replay turns.
- Credential-value scanning and symbolic binding audits.
- Chat JSON completion with structured-output fallback.
- Prompt sessions and progressive iCoT loop control.
- Review-only `LeafAdapter` helpers, the public review state machine, the
  runtime-neutral handoff manifest schema, and artifact writing.

These helpers may identify that a binding is needed, that a draft is incomplete,
or that an artifact contains a likely credential value. They must not resolve
secrets, choose production accounts, approve artifacts, or execute workflows.

## What Runtimes Own

A runtime package should keep these responsibilities local:

- Concrete artifact schemas and wire contracts.
- Product-specific prompt text, profile schemas, and validation policy.
- Approval gates, reviewer identity, state persistence, product-specific
  routing, trusted-runner commands, and enforcement.
- Credential lookup, account binding, endpoint selection, and auth.
- Execution, retries, observability, audit logging, and rollback behavior.
- Product-specific test fixtures and examples.

Ramen, udon, OpenUdon, and future runtimes can share upstream behavior without
depending on each other.

## Interfaces To Implement

Most integrations start with small adapters around existing runtime code:

- `ChatClient` and `StructuredChatClient` for LLM calls.
- `Parser[T]`, `Renderer[T]`, `Validator[T]`, `SlotProvider[T]`, and
  `Refiner[T]` for typed authoring flows.
- `LeafRenderer` when the neutral draft must be rendered into runtime-specific
  artifacts.
- `InteractiveExtractor[S,D]` for iCoT model assistance.

Example chat adapter:

```go
type ChatAdapter struct {
	Client RuntimeLLM
}

func (adapter ChatAdapter) Complete(ctx context.Context, transcript []apitools.TranscriptTurn) (apitools.TranscriptTurn, error) {
	reply, err := adapter.Client.Chat(ctx, runtimeMessages(transcript))
	if err != nil {
		return apitools.TranscriptTurn{}, err
	}
	return apitools.TranscriptTurn{Role: "assistant", Content: reply}, nil
}

func (adapter ChatAdapter) CompleteStructured(ctx context.Context, transcript []apitools.TranscriptTurn, schema any, out any) error {
	rawSchema, err := apitools.RawSchema(schema)
	if err != nil {
		return err
	}
	raw, err := adapter.Client.StructuredChat(ctx, runtimeMessages(transcript), rawSchema)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}
```

Use the adapter with upstream fallback logic:

```go
var draft RuntimeIntent
result, err := apitools.CompleteJSONWithFallback(
	ctx,
	ChatAdapter{Client: llm},
	transcript,
	intentSchema,
	&draft,
	apitools.JSONCompletionOptions{FallbackOnStructuredError: true},
)
_, _ = result, err
```

## Authoring Flow Pattern

For typed draft flows, keep parsing and rendering runtime-owned while using the
shared flow shell:

```go
flow := apitools.Flow[RuntimeIntent]{
	Parser:       runtimeParser{},
	Renderer:     runtimeRenderer{},
	Validator:    runtimeValidator{},
	SlotProvider: runtimeSlotProvider{},
	Refiner:      runtimeRefiner{},
}

draft, artifacts, diagnostics, err := flow.ParseValidateRender(ctx, artifact)
_, _, _, _ = draft, artifacts, diagnostics, err
```

For neutral OpenAPI-backed drafting, use the authoring core and then hand the
leaf to runtime-specific rendering:

```go
core := apitools.NewAuthoringCore()
leaf, rendered, diagnostics, err := apitools.DraftAndRender(ctx, core, runtimeRenderer{}, input)
_, _, _, _ = leaf, rendered, diagnostics, err
```

The runtime renderer may use `leaf.MinimumReviewPackage()`, `leaf.BindingAudit()`,
`leaf.CredentialValueDiagnostics()`, `leaf.ReviewMarkdown()`, and
`leaf.ReviewHandoff(...)` as shared review evidence, then append
product-specific routing, approval persistence, and trusted-runner enforcement.

## ICOT Pattern

`RunProgressiveICOT` owns the loop mechanics: prompt recording, model draft
events, readiness checks, question selection, cancellation, final confirmation,
autosave, and transcript persistence.

The runtime supplies hooks for its session type:

```go
artifacts, err := apitools.RunProgressiveICOT(ctx, in, out, apitools.ProgressiveLoopHooks[Session, APIDoc, Artifacts]{
	Session:                seed,
	Documents:              docs,
	Extractor:              runtimeExtractor{},
	Normalize:              normalizeSession,
	DeterministicPrefill:   deterministicPrefill,
	LooksLikeSession:       looksLikeSession,
	MergeDraft:             mergeDraft,
	CheckReadiness:         checkReadiness,
	Ready:                  ready,
	PlanQuestion:           planQuestion,
	ApplyAnswer:            applyAnswer,
	FinalConfirm:           finalConfirm,
	SaveTranscript:         saveTranscript,
})
_, _ = artifacts, err
```

Do not put runtime prompts, reviewer routing, approval persistence, trusted
execution commands, or session schema into `apitools`. Keep those in the runtime
and pass behavior through hooks. The public review state names, allowed
transitions, handoff inputs, and execution-policy fields are the shared schema;
concrete approval decisions and execution remain downstream-owned.

## Testing Strategy

Test in two layers:

- Upstream: use fake clients, fake extractors, and small typed fixtures to prove
  shared logic and interface contracts.
- Downstream: run the same upstream helpers through real runtime adapters and
  assert product-specific policy still holds.

Good upstream tests cover:

- structured JSON success, fallback, malformed output, and unchanged targets on
  error
- operation selection, plural/camel-case matching, and ambiguous candidates
- credential-value scans and symbolic binding allowlists
- artifact path safety and review package summaries
- progressive iCoT loop cancellation, noop extractor behavior, readiness, and
  final confirmation

Good downstream tests cover:

- runtime adapter conformance
- product-specific validation and review evidence
- approval/trusted-runner paths
- no secret values in prompts, artifacts, transcripts, or review summaries

## Module Path Note

The module path is `github.com/tabilet/apitools`. The package declarations
remain `package apitools`, and the CLI lives under `cmd/apitools`. Runtime
authors should import the module with the package name they use in code:

```go
import apitools "github.com/tabilet/apitools"
```
