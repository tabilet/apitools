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

// SortedOperationSummaries returns operation summaries ordered by operationId.
func SortedOperationSummaries(operations map[string]OperationSummary) []OperationSummary {
	out := make([]OperationSummary, 0, len(operations))
	for _, operation := range operations {
		out = append(out, operation)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
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
