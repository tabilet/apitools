package apitools

import "strings"

func isAWSDialect(hints OperationSelectionHints, operation OperationSummary) bool {
	if hints.Dialect == "aws" || hints.Provider == "aws" {
		return true
	}
	if operation.Extensions["x-aws-operation-name"] != "" {
		return true
	}
	return false
}

func selectAWSQueryOperation(hints OperationSelectionHints, candidates []OperationSummary) OperationSelection {
	action := awsHintAction(hints)
	if action == "" {
		return OperationSelection{}
	}
	matches := make([]OperationSummary, 0, len(candidates))
	for _, candidate := range candidates {
		if hints.Path != "" && !awsPathMatchesHint(candidate.Path, hints.Path, action) {
			continue
		}
		if hints.Method != "" && strings.ToUpper(candidate.Method) != hints.Method {
			continue
		}
		if hints.Purpose != "" {
			purpose := ClassifyOperationPurpose(candidate, hints)
			if purpose != hints.Purpose {
				continue
			}
		}
		if awsOperationMatchesAction(candidate, action) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		return OperationSelection{}
	}
	if hints.Method == "" {
		preferred := preferredAWSMethod(hints.Purpose)
		if preferred != "" {
			var methodMatches []OperationSummary
			for _, match := range matches {
				if strings.ToUpper(match.Method) == preferred {
					methodMatches = append(methodMatches, match)
				}
			}
			if len(methodMatches) > 0 {
				matches = methodMatches
			}
		}
	}
	if len(matches) == 1 {
		return OperationSelection{Operation: matches[0], Found: true, Score: 100}
	}
	return OperationSelection{Ambiguous: true, Score: 100}
}

func classifyAWSQueryPurpose(operation OperationSummary, hints OperationSelectionHints) string {
	action := firstNonEmptyString(awsOperationName(operation), awsHintAction(hints))
	if action == "" || !awsOperationMatchesAction(operation, action) {
		return ""
	}
	action = strings.ToLower(action)
	switch {
	case hasAnyPrefix(action, "describe", "get", "lookup"):
		return "read"
	case hasAnyPrefix(action, "list", "search"):
		return "list"
	case hasAnyPrefix(action, "create", "run", "allocate", "request", "purchase", "register"):
		return "create"
	case hasAnyPrefix(action, "delete", "terminate", "release", "deregister"):
		return "delete"
	case hasAnyPrefix(action, "update", "modify", "put", "attach", "detach", "associate", "disassociate", "start", "stop", "reboot", "enable", "disable"):
		return "update"
	default:
		return ""
	}
}

func awsHintAction(hints OperationSelectionHints) string {
	for key, value := range hints.Parameters {
		if strings.EqualFold(key, "Action") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func awsOperationName(operation OperationSummary) string {
	if value := strings.TrimSpace(operation.Extensions["x-aws-operation-name"]); value != "" {
		return value
	}
	operationID := strings.TrimSpace(operation.OperationID)
	if head, tail, ok := strings.Cut(operationID, "_"); ok && isSelectionHTTPMethod(head) {
		return tail
	}
	return ""
}

func awsOperationMatchesAction(operation OperationSummary, action string) bool {
	action = strings.TrimSpace(action)
	if action == "" {
		return false
	}
	if strings.EqualFold(awsOperationName(operation), action) {
		return true
	}
	method := strings.ToUpper(strings.TrimSpace(operation.Method))
	if method != "" && strings.EqualFold(operation.OperationID, method+"_"+action) {
		return true
	}
	return strings.HasSuffix(strings.ToLower(operation.OperationID), "_"+strings.ToLower(action))
}

func awsPathMatchesHint(operationPath, hintPath, action string) bool {
	operationPath = strings.TrimSpace(operationPath)
	hintPath = strings.TrimSpace(hintPath)
	if hintPath == "" || operationPath == hintPath {
		return true
	}
	base, query, ok := strings.Cut(operationPath, "#")
	if !ok || base != hintPath {
		return false
	}
	for _, field := range strings.FieldsFunc(query, func(r rune) bool { return r == '&' || r == '?' }) {
		key, value, ok := strings.Cut(field, "=")
		if ok && strings.EqualFold(key, "Action") && strings.EqualFold(value, action) {
			return true
		}
	}
	return false
}

func preferredAWSMethod(purpose string) string {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case "read", "list":
		return "GET"
	case "create", "update", "delete", "replace", "import":
		return "POST"
	default:
		return ""
	}
}

func isSelectionHTTPMethod(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE":
		return true
	default:
		return false
	}
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
