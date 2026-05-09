package apitools

import (
	"fmt"
	"strings"
)

// Diagnostic describes an OpenAPI discovery, inventory, or validation issue.
type Diagnostic struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// ReadinessIssue explains why an OpenAPI operation summary needs more caller
// review before it can be used safely in generated artifacts.
type ReadinessIssue struct {
	Severity        string `json:"severity"`
	Code            string `json:"code,omitempty"`
	Message         string `json:"message"`
	OperationID     string `json:"operation_id,omitempty"`
	Path            string `json:"path,omitempty"`
	Slot            string `json:"slot,omitempty"`
	SuggestedAnswer string `json:"suggested_answer,omitempty"`
	Remediation     string `json:"remediation,omitempty"`
}

// DiagnosticError wraps diagnostics as an error value.
type DiagnosticError struct {
	Diagnostics []Diagnostic
}

func (err DiagnosticError) Error() string {
	if len(err.Diagnostics) == 0 {
		return "diagnostics failed"
	}
	var parts []string
	for _, diagnostic := range err.Diagnostics {
		if strings.TrimSpace(diagnostic.Message) != "" {
			parts = append(parts, strings.TrimSpace(diagnostic.Message))
		}
	}
	if len(parts) == 0 {
		return "diagnostics failed"
	}
	return fmt.Sprintf("diagnostics failed: %s", strings.Join(parts, "; "))
}
