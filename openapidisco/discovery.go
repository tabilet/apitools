// Copyright (c) Greetingland LLC

// Package openapidisco is a thin compatibility wrapper over the OpenAPI
// discovery primitives in `github.com/OpenUdon/apitools`.
package openapidisco

import (
	"context"

	"github.com/OpenUdon/apitools"
)

type Candidate = apitools.DiscoveryCandidate
type DiscoveryReport = apitools.DiscoveryReport
type DiscoveryAttempt = apitools.DiscoveryAttempt
type Discoverer = apitools.Discoverer

// LocalFiles discovers OpenAPI documents under openAPIDir, scoring them against
// projectText. baseDir is used to compute relative paths in the returned
// candidates.
//
// Deprecated: use LocalFilesContext so cancellation and deadlines can be
// forwarded to the underlying scan.
func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return LocalFilesContext(context.Background(), openAPIDir, baseDir, projectText)
}

// LocalFilesContext discovers OpenAPI documents under openAPIDir, forwarding
// ctx to the underlying scan.
func LocalFilesContext(ctx context.Context, openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	return apitools.DiscoverOpenAPI(ctx, openAPIDir, baseDir, projectText)
}

// SelectPrimary returns the highest-scoring candidate, or an error if the
// slice is empty.
func SelectPrimary(candidates []Candidate) (Candidate, error) {
	return apitools.SelectPrimaryDiscoveryCandidate(candidates)
}
