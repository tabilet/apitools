package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptSessionRecordsDefaultsRequiredAndYesNo(t *testing.T) {
	input := strings.NewReader("\n\nvalue\nmaybe\ny\n")
	var out bytes.Buffer
	session := NewPromptSession(input, &out)
	got, err := session.AskDefault("Name", "current")
	if err != nil {
		t.Fatal(err)
	}
	required, err := session.AskDefaultRequired("Required", "")
	if err != nil {
		t.Fatal(err)
	}
	yes, err := session.AskYesNo("Confirm", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "current" || required != "value" || !yes {
		t.Fatalf("answers = %q %q %v", got, required, yes)
	}
	if !strings.Contains(out.String(), "Required is required.") || !strings.Contains(out.String(), "Please answer yes or no.") {
		t.Fatalf("prompt output:\n%s", out.String())
	}
	turns := session.Turns()
	if len(turns) != 5 || turns[0].Answer != "current" || strings.TrimSpace(turns[3].Answer) != "maybe" {
		t.Fatalf("turns = %#v", turns)
	}
	if err := AssertPromptLabelsInOrder(out.String(), turns); err != nil {
		t.Fatal(err)
	}
}

func TestSavePromptTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript.json")
	err := SavePromptTranscript(path, "test.v1", []PromptTurn{{Label: "Goal", Answer: "ship"}}, []PromptEvent{{Kind: "event"}}, map[string]string{"ok": "true"})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %v", info.Mode().Perm())
	}
	var transcript PromptTranscript
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &transcript); err != nil {
		t.Fatal(err)
	}
	if transcript.Version != "test.v1" || len(transcript.Turns) != 1 || len(transcript.Events) != 1 {
		t.Fatalf("transcript = %#v", transcript)
	}
}

func TestRunProgressiveICOT(t *testing.T) {
	var out bytes.Buffer
	extractor := &capturingExtractor{draft: testSession{Step: "drafted"}}
	hooks := ProgressiveLoopHooks[testSession, string, testArtifacts]{
		Session:       testSession{},
		Documents:     []string{"doc"},
		OpeningPrompt: "Tell me.",
		Brief:         "# Project\n\nFull source brief.",
		Extractor:     extractor,
		Normalize: func(session *testSession) {
			session.Goal = strings.TrimSpace(session.Goal)
		},
		ApplyOpeningAnswer: func(session *testSession, answer string, _ []string) error {
			session.Goal = answer
			return nil
		},
		DeterministicPrefill: func(session *testSession, _ []string) bool {
			if session.Output == "" && session.Step != "" {
				session.Output = "result"
				return true
			}
			return false
		},
		LooksLikeSession: func(session testSession) bool { return session.Step != "" },
		MergeDraft: func(base, draft testSession, _ []string) testSession {
			base.Step = draft.Step
			return base
		},
		DraftResultSummary: func(session testSession) any {
			return map[string]any{"step": session.Step}
		},
		CheckReadiness: func(session testSession, _ []string) []ReadinessIssue {
			if session.Step == "" {
				return []ReadinessIssue{{Severity: "blocking", Code: "missing_step", Message: "step missing", Slot: "step", SuggestedAnswer: "create"}}
			}
			return nil
		},
		Ready: func(session testSession, issues []ReadinessIssue) bool {
			return session.Output != "" && len(issues) == 0
		},
		PlanQuestion: func(_ testSession, _ []string, issues []ReadinessIssue) InteractiveQuestion {
			return InteractiveQuestion{Prompt: "Step", SuggestedAnswer: issues[0].SuggestedAnswer, Slots: []string{"step"}}
		},
		ApplyAnswer: func(session *testSession, _ InteractiveQuestion, answer string, _ []string) error {
			session.Step = answer
			return nil
		},
		FinalConfirm: func(prompts *PromptSession, session *testSession, _ []string, events *[]PromptEvent) (testArtifacts, error) {
			answer, err := prompts.AskDefault("Type save", "save")
			if err != nil {
				return testArtifacts{}, err
			}
			if answer != "save" {
				return testArtifacts{}, errors.New("not saved")
			}
			*events = append(*events, PromptEvent{Kind: "confirmed"})
			return testArtifacts{Session: *session}, nil
		},
		SaveTranscript: func(turns []PromptTurn, events []PromptEvent, artifacts testArtifacts) error {
			if len(turns) == 0 || len(events) == 0 || artifacts.Session.Output != "result" {
				return errors.New("missing transcript data")
			}
			return nil
		},
	}
	artifacts, err := RunProgressiveICOT(context.Background(), strings.NewReader("goal\n\n"), &out, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Session.Goal != "goal" || artifacts.Session.Step != "drafted" || artifacts.Session.Output != "result" {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if extractor.request.Brief != "# Project\n\nFull source brief." {
		t.Fatalf("draft brief = %q", extractor.request.Brief)
	}
}

func TestRunProgressiveICOTNoopExtractorDoesNotReplaceSeed(t *testing.T) {
	hooks := ProgressiveLoopHooks[testSession, string, testArtifacts]{
		Session: testSession{Goal: "seed goal", Step: "seed step", Output: "seed output"},
		Opening: "seed goal",
		NoLLM:   true,
		Ready:   func(session testSession, _ []ReadinessIssue) bool { return session.Step != "" && session.Output != "" },
		FinalConfirm: func(_ *PromptSession, session *testSession, _ []string, _ *[]PromptEvent) (testArtifacts, error) {
			return testArtifacts{Session: *session}, nil
		},
	}
	artifacts, err := RunProgressiveICOT(context.Background(), strings.NewReader(""), io.Discard, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Session.Goal != "seed goal" || artifacts.Session.Step != "seed step" || artifacts.Session.Output != "seed output" {
		t.Fatalf("seed was replaced by noop draft: %#v", artifacts.Session)
	}
}

type testSession struct {
	Goal   string
	Step   string
	Output string
}

type testArtifacts struct {
	Session testSession
}

type testExtractor struct{}

func (testExtractor) Kickoff(context.Context, string) (testSession, error) {
	return testSession{}, nil
}

func (testExtractor) Draft(context.Context, InteractiveDraftRequest[testSession, string]) (testSession, error) {
	return testSession{Step: "drafted"}, nil
}

func (testExtractor) Refine(_ context.Context, session testSession) (testSession, error) {
	return session, nil
}

func (testExtractor) Disambiguate(context.Context, string, []string) ([]string, error) {
	return nil, nil
}

type capturingExtractor struct {
	draft   testSession
	request InteractiveDraftRequest[testSession, string]
}

func (e *capturingExtractor) Kickoff(context.Context, string) (testSession, error) {
	return testSession{}, nil
}

func (e *capturingExtractor) Draft(_ context.Context, request InteractiveDraftRequest[testSession, string]) (testSession, error) {
	e.request = request
	return e.draft, nil
}

func (e *capturingExtractor) Refine(_ context.Context, session testSession) (testSession, error) {
	return session, nil
}

func (e *capturingExtractor) Disambiguate(context.Context, string, []string) ([]string, error) {
	return nil, nil
}
