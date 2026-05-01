package apitools

import (
	"regexp"
	"sort"
	"strings"
)

func sortResults(results []Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Title != results[j].Title {
			return results[i].Title < results[j].Title
		}
		return results[i].SpecURL < results[j].SpecURL
	})
}

// ScoreText returns a deterministic relevance score for query terms in text.
// Higher scores indicate stronger textual overlap.
func ScoreText(query, text string) int {
	return scoreText(query, text)
}

func scoreText(query, text string) int {
	query = strings.ToLower(strings.TrimSpace(query))
	text = strings.ToLower(text)
	if query == "" || text == "" {
		return 0
	}
	score := 0
	if strings.Contains(text, query) {
		score += 10
	}
	for _, token := range tokenPattern.FindAllString(query, -1) {
		if len(token) < 2 {
			continue
		}
		if strings.Contains(text, token) {
			score += len(token)
		}
	}
	return score
}

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}
