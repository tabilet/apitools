// Package odata parses metadata-only OData CSDL/service artifacts.
//
// It accepts local CSDL XML and JSON CSDL/service metadata. The package never
// calls tenant $metadata endpoints, logs in to ERP tenants, executes $batch,
// executes OData requests, resolves credentials, or selects accounts/tenants.
package odata
