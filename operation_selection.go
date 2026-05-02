package apitools

import (
	"sort"
	"strings"
	"unicode"
)

// OperationSelection describes the deterministic result of matching text to
// OpenAPI operation summaries.
type OperationSelection struct {
	Operation OperationSummary `json:"operation,omitempty"`
	Found     bool             `json:"found,omitempty"`
	Ambiguous bool             `json:"ambiguous,omitempty"`
	Score     int              `json:"score,omitempty"`
}

// OperationSelectionHints carries caller-owned context for selecting an
// OpenAPI operation without embedding product-specific runtime behavior.
type OperationSelectionHints struct {
	Provider   string            `json:"provider,omitempty"`
	Dialect    string            `json:"dialect,omitempty"`
	Purpose    string            `json:"purpose,omitempty"`
	Target     string            `json:"target,omitempty"`
	Method     string            `json:"method,omitempty"`
	Path       string            `json:"path,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// SortedOperationSummaries returns operation summaries ordered by operationId.
func SortedOperationSummaries(operations map[string]OperationSummary) []OperationSummary {
	out := make([]OperationSummary, 0, len(operations))
	for _, operation := range operations {
		out = append(out, operation)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
}

// SelectOperationByHints selects one operation using prompt-safe caller hints
// and OpenAPI metadata. Provider dialects are limited to public wire-shape
// metadata; credential, signing, endpoint, and runtime policy stay downstream.
func SelectOperationByHints(hints OperationSelectionHints, candidates []OperationSummary) OperationSelection {
	if len(candidates) == 0 {
		return OperationSelection{}
	}
	hints = normalizeOperationSelectionHints(hints)
	if isAWSDialect(hints, OperationSummary{}) {
		if selected := selectAWSQueryOperation(hints, candidates); selected.Found || selected.Ambiguous {
			return selected
		}
	}
	filtered := filterOperationsByHints(hints, candidates)
	if len(filtered) == 0 {
		return OperationSelection{}
	}
	if hints.Target != "" {
		return SelectOperationByText(hints.Target, filtered)
	}
	if len(filtered) == 1 {
		return OperationSelection{Operation: filtered[0], Found: true, Score: 1}
	}
	return OperationSelection{Ambiguous: true, Score: 1}
}

// ClassifyOperationPurpose maps an operation to a generic authoring purpose.
// AWS Query action names override HTTP-method classification because many
// read-only AWS operations are available through POST.
func ClassifyOperationPurpose(operation OperationSummary, hints OperationSelectionHints) string {
	hints = normalizeOperationSelectionHints(hints)
	if isAWSDialect(hints, operation) {
		if purpose := classifyAWSQueryPurpose(operation, hints); purpose != "" {
			return purpose
		}
	}
	switch strings.ToUpper(strings.TrimSpace(operation.Method)) {
	case "POST":
		return "create"
	case "GET":
		return "read"
	case "PATCH", "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return ""
	}
}

// SelectOperationByText selects the single best candidate whose operationId or
// path overlaps target text. It tokenizes camelCase, separators, and simple
// plural forms so callers can match resource names to operation summaries.
func SelectOperationByText(target string, candidates []OperationSummary) OperationSelection {
	if len(candidates) == 0 {
		return OperationSelection{}
	}
	targetTokens := operationTokenSet(target)
	bestScore := -1
	var best OperationSummary
	ambiguous := false
	for _, candidate := range candidates {
		score := operationTokenOverlap(targetTokens, operationTokenSet(candidate.OperationID+" "+candidate.Path))
		if score > bestScore {
			bestScore = score
			best = candidate
			ambiguous = false
			continue
		}
		if score == bestScore {
			ambiguous = true
		}
	}
	if bestScore <= 0 {
		return OperationSelection{}
	}
	if ambiguous {
		return OperationSelection{Ambiguous: true, Score: bestScore}
	}
	return OperationSelection{Operation: best, Found: true, Score: bestScore}
}

func normalizeOperationSelectionHints(hints OperationSelectionHints) OperationSelectionHints {
	hints.Provider = strings.ToLower(strings.TrimSpace(hints.Provider))
	if root, _, ok := strings.Cut(hints.Provider, "."); ok {
		hints.Provider = root
	}
	hints.Dialect = strings.ToLower(strings.TrimSpace(hints.Dialect))
	hints.Purpose = strings.ToLower(strings.TrimSpace(hints.Purpose))
	hints.Target = strings.TrimSpace(hints.Target)
	hints.Method = strings.ToUpper(strings.TrimSpace(hints.Method))
	hints.Path = strings.TrimSpace(hints.Path)
	if len(hints.Parameters) > 0 {
		params := make(map[string]string, len(hints.Parameters))
		for key, value := range hints.Parameters {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key != "" && value != "" {
				params[key] = value
			}
		}
		hints.Parameters = params
	}
	return hints
}

func filterOperationsByHints(hints OperationSelectionHints, candidates []OperationSummary) []OperationSummary {
	out := make([]OperationSummary, 0, len(candidates))
	for _, candidate := range candidates {
		if hints.Method != "" && strings.ToUpper(candidate.Method) != hints.Method {
			continue
		}
		if hints.Path != "" && candidate.Path != hints.Path {
			continue
		}
		if hints.Purpose != "" {
			purpose := ClassifyOperationPurpose(candidate, hints)
			if purpose != hints.Purpose {
				continue
			}
		}
		out = append(out, candidate)
	}
	return out
}

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

func operationTokenOverlap(left, right map[string]struct{}) int {
	score := 0
	for token := range left {
		if _, ok := right[token]; ok {
			score++
		}
		if singular := strings.TrimSuffix(token, "s"); singular != token {
			if _, ok := right[singular]; ok {
				score++
			}
		}
	}
	return score
}

func operationTokenSet(value string) map[string]struct{} {
	tokens := map[string]struct{}{}
	for _, token := range splitOperationTokens(value) {
		if token == "" {
			continue
		}
		tokens[token] = struct{}{}
		if strings.HasSuffix(token, "s") && len(token) > 1 {
			tokens[strings.TrimSuffix(token, "s")] = struct{}{}
		}
	}
	return tokens
}

func splitOperationTokens(value string) []string {
	var tokens []string
	var current []rune
	var previous rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = nil
		}
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if len(current) > 0 && unicode.IsLower(previous) && unicode.IsUpper(r) {
				flush()
			}
			current = append(current, unicode.ToLower(r))
			previous = r
			continue
		}
		flush()
		previous = 0
	}
	flush()
	return tokens
}
