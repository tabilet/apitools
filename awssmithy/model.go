package awssmithy

import (
	"github.com/OpenUdon/apitools/internal/sourceguard"
	upstream "github.com/OpenUdon/awssmithy"
)

const (
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolRestJSON1.
	ProtocolRestJSON1 = upstream.ProtocolRestJSON1
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolRestXML.
	ProtocolRestXML = upstream.ProtocolRestXML
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolAWSQuery.
	ProtocolAWSQuery = upstream.ProtocolAWSQuery
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolEC2Query.
	ProtocolEC2Query = upstream.ProtocolEC2Query
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolAWSJSON10.
	ProtocolAWSJSON10 = upstream.ProtocolAWSJSON10
	// Deprecated: use github.com/OpenUdon/awssmithy.ProtocolAWSJSON11.
	ProtocolAWSJSON11 = upstream.ProtocolAWSJSON11
)

// Model is a parsed AWS Smithy JSON service model.
//
// Deprecated: use github.com/OpenUdon/awssmithy.Model.
type Model = upstream.Model

// Shape is a normalized Smithy shape record.
//
// Deprecated: use github.com/OpenUdon/awssmithy.Shape.
type Shape = upstream.Shape

// Operation is a protocol-aware Smithy operation plan.
//
// Deprecated: use github.com/OpenUdon/awssmithy.Operation.
type Operation = upstream.Operation

// MemberBinding describes how a Smithy structure member participates in an
// operation request or response.
//
// Deprecated: use github.com/OpenUdon/awssmithy.MemberBinding.
type MemberBinding = upstream.MemberBinding

// Parse parses an AWS Smithy JSON service model into native Smithy metadata.
//
// Deprecated: use github.com/OpenUdon/awssmithy.Parse.
func Parse(data []byte) (*Model, error) {
	if err := sourceguard.CheckJSON("aws-smithy", data); err != nil {
		return nil, err
	}
	return upstream.Parse(data)
}

// ParseMap parses an already-decoded AWS Smithy JSON service model into native
// Smithy metadata.
//
// Deprecated: use github.com/OpenUdon/awssmithy.ParseMap.
func ParseMap(raw map[string]any) (*Model, error) {
	if err := sourceguard.CheckValue("aws-smithy", raw); err != nil {
		return nil, err
	}
	return upstream.ParseMap(raw)
}
