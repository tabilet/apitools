package apitools

import "testing"

func TestScoreText(t *testing.T) {
	cases := []struct {
		query, text string
		want        int
	}{
		{"slack", "slack api", 10 + len("slack")},
		{"slack messages", "search slack messages api", 10 + len("slack") + len("messages")},
		{"", "anything", 0},
		{"x", "anything else", 0},
		{"slack", "", 0},
		{"missing", "no overlap", 0},
	}
	for _, tc := range cases {
		if got := ScoreText(tc.query, tc.text); got != tc.want {
			t.Errorf("ScoreText(%q,%q) = %d, want %d", tc.query, tc.text, got, tc.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "first", "second"); got != "first" {
		t.Errorf("got %q", got)
	}
	if got := firstNonEmpty("", "  "); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestNonEmptyStrings(t *testing.T) {
	got := nonEmptyStrings("a", "", "  ", "b")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %#v", got)
	}
}

func TestLooksLikeCredentialName(t *testing.T) {
	flagged := []string{
		"password", "user.password", "auth_token", "AUTH-TOKEN", "secret",
		"api_key", "apiKey", "X-API-KEY", "credentials.access",
		"groups[].token",
	}
	for _, name := range flagged {
		if !looksLikeCredentialName(name) {
			t.Errorf("expected %q to be flagged", name)
		}
	}
	allowed := []string{
		"name", "email", "user.profile.display_name", "passwordless_login",
		"keystore_path", "tokenizer", "subject",
	}
	for _, name := range allowed {
		if looksLikeCredentialName(name) {
			t.Errorf("did not expect %q to be flagged", name)
		}
	}
}

func TestSortResultsOrdersByScore(t *testing.T) {
	results := []Result{
		{Title: "B", Score: 1, SpecURL: "u/b"},
		{Title: "A", Score: 5, SpecURL: "u/a"},
		{Title: "C", Score: 5, SpecURL: "u/c"},
	}
	sortResults(results)
	if results[0].Title != "A" || results[1].Title != "C" || results[2].Title != "B" {
		t.Errorf("ordering = %#v", results)
	}
}
