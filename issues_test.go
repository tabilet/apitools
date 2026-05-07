package apitools

import (
	"strings"
	"testing"
)

func TestIssueSetFromFindingsNormalizesSortsAndRenders(t *testing.T) {
	set := IssueSetFromFindings(
		[]Diagnostic{{Severity: "warn", Code: "z.warning", Message: "later"}, {Severity: "error", Code: "a.error", Message: "first", Path: "intent.hcl"}},
		[]ReadinessIssue{{Severity: "blocking", Code: "slot.missing", Message: "needs answer", Slot: "goal"}},
	)
	if len(set.Issues) != 3 {
		t.Fatalf("issues = %#v", set.Issues)
	}
	if set.Issues[0].Code != "a.error" || set.Issues[1].Code != "slot.missing" || set.Issues[2].Severity != "warning" {
		t.Fatalf("issue order = %#v", set.Issues)
	}
	if !HasBlockingIssues(set.Issues) {
		t.Fatal("expected blocking issue")
	}
	md := RenderIssuesMarkdown(set.Issues)
	for _, want := range []string{"`a.error`", "intent.hcl", "`slot.missing`", "slot `goal`"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}
