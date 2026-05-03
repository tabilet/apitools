# AI-Assisted Authoring Core

`apitools` provides domain-neutral authoring primitives for tools that
draft OpenAPI-backed workflow artifacts. It searches and imports OpenAPI
documents, turns them into prompt-safe operation context, and supports draft
`project.md` and `intent.hcl` generation without committing to a concrete
runtime, credential source, product workflow, or infrastructure domain.

The shared flow is:

```text
brief + OpenAPI docs -> prompt-safe operation context -> project.md / intent.hcl draft -> caller-specific leaf renderer
```

Draft artifacts require caller-side validation and review before they can be
used. This package does not execute APIs, inject credentials, resolve concrete
accounts, or perform production side effects.

## Shared Concepts

- Transcript: the ordered authoring conversation, tool observations, imported
  OpenAPI references, accepted assumptions, questions, answers, and artifact
  revisions that explain how a draft was produced.
- Diagnostic: a structured message with severity, location, and remediation
  guidance for an authoring or validation issue.
- Slot: a named missing or variable value needed by a draft, such as a resource
  identifier, region, environment name, request field, or confirmation answer.
- Assumption: a provisional choice made to keep drafting when the brief or
  OpenAPI document is ambiguous. Assumptions must be visible to the caller and
  reviewable before execution.
- Symbolic binding: a stable logical name for a runtime-provided capability,
  credential, account, environment, or endpoint. Symbolic bindings are declared
  during authoring and resolved by the caller at execution time.
- Operation inventory: a normalized, prompt-safe summary of candidate OpenAPI
  operations, including methods, paths, operation identifiers, schemas,
  required fields, security declarations, and provenance.
- Documentation context: optional advisory snippets from documentation
  providers. These snippets can improve reviewer understanding but must not
  override OpenAPI operations, schemas, required fields, or security
  declarations.
- Readiness issue: a blocker or warning that explains why a draft is not ready
  for validation, review, rendering, or execution.
- Question plan: a prioritized list of questions that would reduce ambiguity,
  fill required slots, or remove readiness issues.
- Artifact set: the draft files and metadata produced by an authoring loop,
  commonly including `project.md`, `intent.hcl`, diagnostics, transcript
  references, selected operation inventory, and declared symbolic bindings.

## Ownership

`apitools` owns shared OpenAPI search, import, validation, local discovery,
cache access, operation inventory, ranking, prompt-safe summaries, and generic
authoring interfaces. It may define common fields, methods, interfaces,
transcripts, diagnostics, symbolic binding declarations, and draft artifact
contracts when those concepts are neutral across callers. It also owns the
public review state machine and runtime-neutral handoff manifest schema used by
downstream leaf adapters.

Ramen owns Ramen-specific `project.md`, workflow `intent.hcl`, `workflow.hcl`,
UWS generation, Symphony routing policy, trusted runner wrappers, evals, and
private udon integration.

OpenUdon owns concrete IaC intent models, Terraform generation,
graph/profile/planning/state/drift/handoff bundles, and w8m-facing public IaC
artifacts.

Ramen, udon, and OpenUdon depend on or embed `apitools` concepts directly.
They do not inherit from each other.

## Binding Model

Authoring uses symbolic names because concrete credentials, accounts,
environments, runtimes, and side-effect permissions are caller responsibilities.
Binding happens at execution time, after the caller has validated and reviewed
the draft artifacts.

`apitools` can declare that an artifact requires a binding such as
`github_default`, `aws_workload_dev`, or `slack_workspace_ops`, but it must not
resolve those names to tokens, accounts, endpoints, or executable clients.

```go
type AuthoringCore interface {
	BuildContext(ctx context.Context, brief Brief, docs []OpenAPIDoc) (OperationContext, error)
	Draft(ctx context.Context, input DraftInput) (ArtifactSet, error)
}

type LeafAdapter struct {
	Name                    string
	Source                  string
	ArtifactSet             ArtifactSet
	ReviewPackage           ReviewPackage
	DeferredExecutionPolicy DeferredExecutionPolicy
}

type LeafRenderer interface {
	RenderLeaf(ctx context.Context, leaf LeafAdapter) (ArtifactSet, []Diagnostic, error)
}

type RuntimeBinder interface {
	Bind(ctx context.Context, names []SymbolicBinding) (BoundRuntime, error)
}

func DraftAndRender(ctx context.Context, core AuthoringCore, renderer LeafRenderer, input DraftInput) (LeafAdapter, ArtifactSet, []Diagnostic, error) {
	artifacts, err := core.Draft(ctx, input)
	if err != nil {
		return LeafAdapter{}, ArtifactSet{}, nil, err
	}
	leaf := NewLeafAdapter(artifacts, LeafOptions{Name: input.Brief.ProjectName})
	rendered, diagnostics, err := renderer.RenderLeaf(ctx, leaf)
	return leaf, rendered, diagnostics, err
}

func ExecuteReviewed(ctx context.Context, binder RuntimeBinder, leaf ExecutableLeaf, artifacts ArtifactSet) error {
	runtime, err := binder.Bind(ctx, artifacts.SymbolicBindings)
	if err != nil {
		return err
	}
	return leaf.Execute(ctx, runtime, artifacts)
}
```

In this model, `apitools` participates in `DraftAndRender` by supplying
prompt-safe context and domain-neutral draft structures. The caller owns
validation, review, concrete leaf rendering, runtime binding, and execution.

The package exposes this model as a neutral authoring facade:

- `AuthoringCore` builds prompt-safe `OperationContext` values and drafts
  neutral artifact sets.
- `NeutralAuthoringCore` is the default implementation over
  `BuildOperationInventory` and `DraftArtifacts`.
- `DraftFromOpenAPI` runs the full neutral pipeline for callers that already
  have OpenAPI documents.
- `LeafAdapter` is a concrete embeddable review-only object with shared
  artifact, readiness, binding-audit, and review helpers.
- `DraftAndRender` hands the neutral leaf to a caller-owned `LeafRenderer`
  without performing credential binding or execution.
- `ArtifactOutputPath` and `WriteArtifactSet` provide filesystem-safe artifact
  writing for callers that need to persist rendered artifact sets.
- `DocumentationContextProvider` is a generic optional hook for advisory
  context providers. The `context7` subpackage is one implementation; callers
  may use other providers or pass already-fetched snippets.

## Prompt Safety

Prompt-safe OpenAPI metadata should include only the information needed for
selection and drafting: operation names, descriptions, methods, paths, schemas,
required fields, security scheme labels, and source provenance. Examples,
defaults, secrets, tokens, credential values, production identifiers, and
workflow execution data must not be cached or included in prompts.

Advisory documentation snippets are treated as untrusted input. Callers should
cap snippet count and size, preserve provenance, and reject credential-looking
content before rendering review artifacts. OpenAPI remains authoritative.
