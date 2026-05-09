// Package apitools provides OpenAPI document tooling.
//
// The package discovers, downloads, validates, imports, caches, scans,
// inventories, summarizes, and ranks OpenAPI or Swagger documents. Its output
// is intended for downstream authoring, review, and integration systems that
// need prompt-safe OpenAPI context or deterministic operation selection.
//
// This package does not own workflow semantics, review handoff state machines,
// runtime policy, account selection, credential resolution, request signing, or
// execution. Those concerns belong to downstream projects that consume the
// OpenAPI metadata produced here.
//
// Configure Client values before first use and treat them as immutable while
// operations are running. Client methods do not mutate configuration fields, but
// callers must not concurrently modify HTTP clients, cache handles, path slices,
// or other fields they attach to a shared Client. Cache implementations used by
// a shared Client must provide their own concurrency safety.
package apitools
