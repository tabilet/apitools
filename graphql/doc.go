// Package graphql parses metadata-only GraphQL source artifacts.
//
// It accepts local SDL, introspection JSON, and operation documents. The
// package never contacts GraphQL endpoints, runs introspection against live
// services, executes operations, resolves variables, or looks up credentials.
package graphql
