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
