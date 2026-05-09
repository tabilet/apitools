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
package apitools
