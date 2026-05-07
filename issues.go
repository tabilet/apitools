package apitools

import (
	"fmt"
	"sort"
	"strings"
)

// Issue is a normalized review/readiness finding shape. Callers can map their
// domain diagnostics into this type without moving the checks into apitools.
type Issue struct {
	Kind        string `json:"kind,omitempty"`
	Severity    string `json:"severity"`
	Code        string `json:"code,omitempty"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Slot        string `json:"slot,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// IssueSet is a stable, review-oriented collection of normalized issues.
type IssueSet struct {
	Issues []Issue `json:"issues,omitempty"`
}

// IssuesFromDiagnostics maps generic diagnostics into normalized issues.
func IssuesFromDiagnostics(diagnostics []Diagnostic) []Issue {
	out := make([]Issue, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, Issue{
			Kind:        "diagnostic",
			Severity:    normalizeIssueSeverity(diagnostic.Severity),
			Code:        strings.TrimSpace(diagnostic.Code),
			Message:     strings.TrimSpace(diagnostic.Message),
			Path:        strings.TrimSpace(diagnostic.Path),
			Remediation: strings.TrimSpace(diagnostic.Remediation),
		})
	}
	return SortIssues(out)
}

// IssuesFromReadiness maps readiness issues into normalized issues.
func IssuesFromReadiness(issues []ReadinessIssue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, Issue{
			Kind:        "readiness",
			Severity:    normalizeIssueSeverity(issue.Severity),
			Code:        strings.TrimSpace(issue.Code),
			Message:     strings.TrimSpace(issue.Message),
			Path:        strings.TrimSpace(issue.Path),
			OperationID: strings.TrimSpace(issue.OperationID),
			Slot:        strings.TrimSpace(issue.Slot),
			Remediation: strings.TrimSpace(issue.Remediation),
		})
	}
	return SortIssues(out)
}

// NormalizeIssues trims and sorts issues, dropping empty entries.
func NormalizeIssues(issues []Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		issue.Kind = strings.TrimSpace(issue.Kind)
		issue.Severity = normalizeIssueSeverity(issue.Severity)
		issue.Code = strings.TrimSpace(issue.Code)
		issue.Message = strings.TrimSpace(issue.Message)
		issue.Path = strings.TrimSpace(issue.Path)
		issue.OperationID = strings.TrimSpace(issue.OperationID)
		issue.Slot = strings.TrimSpace(issue.Slot)
		issue.Remediation = strings.TrimSpace(issue.Remediation)
		if issue.Code == "" && issue.Message == "" && issue.Path == "" {
			continue
		}
		out = append(out, issue)
	}
	return SortIssues(out)
}

// SortIssues returns issues in stable severity/code/path/message order.
func SortIssues(issues []Issue) []Issue {
	out := append([]Issue(nil), issues...)
	sort.SliceStable(out, func(i, j int) bool {
		left := issueSortKey(out[i])
		right := issueSortKey(out[j])
		for n := range left {
			if left[n] != right[n] {
				return left[n] < right[n]
			}
		}
		return false
	})
	return out
}

// IssueSetFromFindings builds a normalized issue set from diagnostics and
// readiness issues.
func IssueSetFromFindings(diagnostics []Diagnostic, readiness []ReadinessIssue) IssueSet {
	issues := append(IssuesFromDiagnostics(diagnostics), IssuesFromReadiness(readiness)...)
	return IssueSet{Issues: NormalizeIssues(issues)}
}

// HasBlockingIssues reports whether any issue is blocking/error severity.
func HasBlockingIssues(issues []Issue) bool {
	for _, issue := range issues {
		if blockingSeverity(issue.Severity) {
			return true
		}
	}
	return false
}

// RenderIssuesMarkdown renders issues for a human review artifact.
func RenderIssuesMarkdown(issues []Issue) string {
	issues = NormalizeIssues(issues)
	if len(issues) == 0 {
		return "- No issues reported.\n"
	}
	var b strings.Builder
	for _, issue := range issues {
		label := firstNonEmpty(issue.Code, issue.Severity, "issue")
		fmt.Fprintf(&b, "- `%s`: %s", label, firstNonEmpty(issue.Message, "review required"))
		if issue.Path != "" {
			fmt.Fprintf(&b, " at `%s`", issue.Path)
		}
		if issue.OperationID != "" {
			fmt.Fprintf(&b, " for operation `%s`", issue.OperationID)
		}
		if issue.Slot != "" {
			fmt.Fprintf(&b, " slot `%s`", issue.Slot)
		}
		if issue.Remediation != "" {
			fmt.Fprintf(&b, " %s", issue.Remediation)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func normalizeIssueSeverity(severity string) string {
	severity = strings.ToLower(strings.TrimSpace(severity))
	switch severity {
	case "", "info", "warning", "warn", "error", "critical", "blocking", "fatal":
		if severity == "" {
			return "info"
		}
		if severity == "warn" {
			return "warning"
		}
		return severity
	default:
		return severity
	}
}

func issueSortKey(issue Issue) [7]string {
	return [7]string{
		issueSeverityRank(issue.Severity),
		strings.TrimSpace(issue.Kind),
		strings.TrimSpace(issue.Code),
		strings.TrimSpace(issue.Path),
		strings.TrimSpace(issue.OperationID),
		strings.TrimSpace(issue.Slot),
		strings.TrimSpace(issue.Message),
	}
}

func issueSeverityRank(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "fatal", "critical", "blocking", "error":
		return "0"
	case "warning", "warn":
		return "1"
	default:
		return "2"
	}
}
