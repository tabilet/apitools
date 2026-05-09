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
