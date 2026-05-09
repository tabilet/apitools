// Package sqlitecache provides a SQLite-backed cache for apitools search
// results and imported OpenAPI documents.
//
// The cache stores public catalog results and downloaded OpenAPI document
// bytes with freshness checks and integrity metadata. It does not store
// credentials, workflow execution data, or runtime state.
//
// SQLite permits multiple Cache values to open the same database file. Callers
// should close caches when finished and should not rely on this package to
// enforce a process-local singleton for any path.
package sqlitecache
