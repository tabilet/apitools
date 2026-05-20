// Package awssmithy parses official AWS Smithy JSON service models into native
// protocol-aware metadata.
//
// The parser is intentionally metadata-only. It preserves Smithy service,
// operation, HTTP binding, shape, AWS protocol, and signing traits without
// executing API operations, resolving credentials, signing requests, fetching
// tokens, or choosing AWS accounts or regions.
package awssmithy
