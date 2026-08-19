// Package sourceguard defines the common resource contract for parsers that
// consume untrusted API-description artifacts.
package sourceguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
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
	// MaxStructuralItems bounds the streaming JSON/YAML preflight. Structural
	// scans are linear in the already-bounded source bytes and need a higher
	// ceiling than semantic parser work so valid large OpenAPI documents (for
	// example Kubernetes Swagger) remain inspectable.
	MaxStructuralItems = 1_000_000
)

// CheckDocument enforces the shared direct-parser byte limit.
func CheckDocument(kind string, data []byte) error {
	if len(data) > MaxDocumentBytes {
		return fmt.Errorf("%s: document exceeds maximum size %d bytes", kind, MaxDocumentBytes)
	}
	return nil
}

// CheckYAML validates YAML structure without expanding aliases into recursive
// Go values. Alias nodes are rejected because expansion can multiply work
// beyond the source byte and node budgets.
func CheckYAML(kind string, data []byte) error {
	if err := CheckDocument(kind, data); err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("%s: decode YAML structure: %w", kind, err)
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%s: YAML contains multiple documents", kind)
		}
		return fmt.Errorf("%s: decode YAML trailing content: %w", kind, err)
	}
	type item struct {
		node  *yaml.Node
		depth int
	}
	stack := []item{{node: &document, depth: 0}}
	work := 0
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		if current.node == nil {
			continue
		}
		work++
		if work > MaxStructuralItems {
			return fmt.Errorf("%s: YAML node count exceeds maximum %d", kind, MaxStructuralItems)
		}
		if current.depth > MaxNestingDepth {
			return fmt.Errorf("%s: YAML nesting exceeds maximum depth %d", kind, MaxNestingDepth)
		}
		if current.node.Kind == yaml.AliasNode {
			return fmt.Errorf("%s: YAML aliases are not supported in untrusted source metadata", kind)
		}
		for i := len(current.node.Content) - 1; i >= 0; i-- {
			stack = append(stack, item{node: current.node.Content[i], depth: current.depth + 1})
		}
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
		if work > MaxStructuralItems {
			return fmt.Errorf("%s: JSON token count exceeds maximum %d", kind, MaxStructuralItems)
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

// CheckValue bounds an already-decoded JSON-shaped value before a parser
// traverses it. ParseMap-style compatibility APIs cannot apply a source-byte
// limit, so this guard accounts for container/scalar work, nesting, and the
// aggregate bytes retained in string keys and values. Cyclic maps or slices
// terminate at the nesting limit instead of recursing indefinitely.
func CheckValue(kind string, value any) error {
	type item struct {
		value any
		depth int
	}
	stack := []item{{value: value}}
	work := 0
	retainedBytes := 0
	addBytes := func(n int) error {
		if n > MaxDocumentBytes-retainedBytes {
			return fmt.Errorf("%s: decoded string data exceeds maximum size %d bytes", kind, MaxDocumentBytes)
		}
		retainedBytes += n
		return nil
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]
		work++
		if work > MaxWorkItems {
			return fmt.Errorf("%s: decoded value work exceeds maximum %d items", kind, MaxWorkItems)
		}
		if current.depth > MaxNestingDepth {
			return fmt.Errorf("%s: decoded value nesting exceeds maximum depth %d", kind, MaxNestingDepth)
		}
		switch typed := current.value.(type) {
		case map[string]any:
			if len(typed) > MaxWorkItems-work-len(stack) {
				return fmt.Errorf("%s: decoded value work exceeds maximum %d items", kind, MaxWorkItems)
			}
			for key, child := range typed {
				if err := addBytes(len(key)); err != nil {
					return err
				}
				stack = append(stack, item{value: child, depth: current.depth + 1})
			}
		case []any:
			if len(typed) > MaxWorkItems-work-len(stack) {
				return fmt.Errorf("%s: decoded value work exceeds maximum %d items", kind, MaxWorkItems)
			}
			for _, child := range typed {
				stack = append(stack, item{value: child, depth: current.depth + 1})
			}
		case string:
			if err := addBytes(len(typed)); err != nil {
				return err
			}
		case []byte:
			if err := addBytes(len(typed)); err != nil {
				return err
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
