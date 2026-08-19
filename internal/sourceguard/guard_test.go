package sourceguard

import (
	"strings"
	"testing"
)

func TestCheckJSONUsesStructuralBudgetSeparateFromSemanticWork(t *testing.T) {
	data := []byte("[" + strings.Repeat("0,", MaxWorkItems) + "0]")
	if err := CheckJSON("openapi", data); err != nil {
		t.Fatalf("linear JSON above the semantic work budget was rejected: %v", err)
	}
}

func TestCheckJSONRejectsStructuralBudgetOverflow(t *testing.T) {
	data := []byte("[" + strings.Repeat("0,", MaxStructuralItems) + "0]")
	err := CheckJSON("openapi", data)
	if err == nil || !strings.Contains(err.Error(), "JSON token count exceeds maximum") {
		t.Fatalf("structural overflow error = %v", err)
	}
}

func TestCheckValueRejectsDepthWorkAndRetainedBytes(t *testing.T) {
	deep := map[string]any{}
	cursor := deep
	for range MaxNestingDepth + 1 {
		next := map[string]any{}
		cursor["next"] = next
		cursor = next
	}
	if err := CheckValue("decoded", deep); err == nil || !strings.Contains(err.Error(), "nesting exceeds") {
		t.Fatalf("depth error = %v", err)
	}

	wide := make([]any, MaxWorkItems)
	if err := CheckValue("decoded", wide); err == nil || !strings.Contains(err.Error(), "work exceeds") {
		t.Fatalf("work error = %v", err)
	}

	large := map[string]any{"value": strings.Repeat("x", MaxDocumentBytes+1)}
	if err := CheckValue("decoded", large); err == nil || !strings.Contains(err.Error(), "string data exceeds") {
		t.Fatalf("retained-byte error = %v", err)
	}
}
