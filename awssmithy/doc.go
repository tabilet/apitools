// Package awssmithy converts official AWS Smithy JSON service models into a
// bounded OpenAPI 3.0.1 shape.
//
// The converter is intentionally metadata-only. It maps Smithy service,
// operation, HTTP binding, schema, and AWS protocol/signing traits into
// OpenAPI-shaped metadata without executing API operations, resolving
// credentials, signing requests, fetching tokens, or choosing AWS accounts or
// regions.
package awssmithy
