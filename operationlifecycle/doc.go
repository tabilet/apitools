// Package operationlifecycle conservatively ranks same-source API operations
// into desired-state lifecycle roles. It inspects prompt-safe operation
// summaries only; it does not fetch sources or execute operations.
package operationlifecycle
