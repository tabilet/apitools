// Package sourceguard defines the common resource contract for parsers that
// consume untrusted API-description artifacts.
package sourceguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// MaxDocumentBytes is the maximum size accepted by direct parser entry
	// points. Discovery callers may impose a smaller limit before parsing.
	MaxDocumentBytes = 20 << 20
	// MaxNestingDepth bounds structural and recursive parser nesting.
	MaxNestingDepth = 100
	// MaxWorkItems bounds tokens, wire fields, or decoded nodes handled by one
	// parse operation.
	MaxWorkItems = 100_000
)

// CheckDocument enforces the shared direct-parser byte limit.
func CheckDocument(kind string, data []byte) error {
	if len(data) > MaxDocumentBytes {
		return fmt.Errorf("%s: document exceeds maximum size %d bytes", kind, MaxDocumentBytes)
	}
	return nil
}

// CheckJSON performs a streaming structural pass before callers decode JSON
// into recursive maps. It limits nesting and total tokens without retaining
// attacker-controlled values.
func CheckJSON(kind string, data []byte) error {
	if err := CheckDocument(kind, data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	depth := 0
	work := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%s: decode JSON structure: %w", kind, err)
		}
		work++
		if work > MaxWorkItems {
			return fmt.Errorf("%s: JSON token count exceeds maximum %d", kind, MaxWorkItems)
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				depth++
				if depth > MaxNestingDepth {
					return fmt.Errorf("%s: JSON nesting exceeds maximum depth %d", kind, MaxNestingDepth)
				}
			case '}', ']':
				depth--
			}
		}
	}
	return nil
}

// Budget tracks work shared by recursive or nested binary decoders.
type Budget struct {
	kind  string
	used  int
	limit int
}

// NewBudget returns a work budget using the shared default limit.
func NewBudget(kind string) *Budget {
	return &Budget{kind: kind, limit: MaxWorkItems}
}

// Use accounts for n work items.
func (b *Budget) Use(n int) error {
	if b == nil || n <= 0 {
		return nil
	}
	b.used += n
	if b.used > b.limit {
		return fmt.Errorf("%s: parser work exceeds maximum %d items", b.kind, b.limit)
	}
	return nil
}
