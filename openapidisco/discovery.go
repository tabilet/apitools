// Copyright (c) Greetingland LLC

// Package openapidisco is a thin convenience wrapper over the OpenAPI
// discovery primitives in `github.com/OpenUdon/apitools`. It exists so
// callers that want local-file discovery and primary-candidate selection do
// not need to thread a context through every call site.
package openapidisco

import (
	"context"

	"github.com/OpenUdon/apitools"
)

type Candidate = apitools.DiscoveryCandidate
type DiscoveryReport = apitools.DiscoveryReport
type DiscoveryAttempt = apitools.DiscoveryAttempt
type Discoverer = apitools.Discoverer

// LocalFiles discovers OpenAPI documents under openAPIDir, scoring them
// against projectText. baseDir is used to compute relative paths in the
// returned candidates.
func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return apitools.DiscoverOpenAPI(context.Background(), openAPIDir, baseDir, projectText)
}

// SelectPrimary returns the highest-scoring candidate, or an error if the
// slice is empty.
func SelectPrimary(candidates []Candidate) (Candidate, error) {
	return apitools.SelectPrimaryDiscoveryCandidate(candidates)
}
