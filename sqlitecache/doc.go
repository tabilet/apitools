// Package sqlitecache provides a SQLite-backed cache for apitools search
// results and imported OpenAPI documents.
//
// The cache stores public catalog results and downloaded OpenAPI document
// bytes with freshness checks and integrity metadata. It does not store
// credentials, workflow execution data, or runtime state.
package sqlitecache
