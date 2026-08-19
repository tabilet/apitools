package googlediscovery

import (
	"github.com/OpenUdon/apitools/internal/sourceguard"
	upstream "github.com/OpenUdon/googlediscovery"
)

// Model is a parsed Google Discovery document.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.Model.
type Model = upstream.Model

// Operation is a flattened Discovery method with inherited metadata applied.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.Operation.
type Operation = upstream.Operation

// Parameter describes a Discovery method parameter after inheritance.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.Parameter.
type Parameter = upstream.Parameter

// MediaUpload describes a Discovery media upload protocol.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.MediaUpload.
type MediaUpload = upstream.MediaUpload

// Parse parses a Google Discovery document into native Discovery metadata.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.Parse.
func Parse(data []byte) (*Model, error) {
	if err := sourceguard.CheckJSON("google-discovery", data); err != nil {
		return nil, err
	}
	return upstream.Parse(data)
}

// ParseMap parses an already-decoded Google Discovery document into native
// Discovery metadata.
//
// Deprecated: use github.com/OpenUdon/googlediscovery.ParseMap.
func ParseMap(raw map[string]any) (*Model, error) {
	if err := sourceguard.CheckValue("google-discovery", raw); err != nil {
		return nil, err
	}
	return upstream.ParseMap(raw)
}
