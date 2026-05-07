package icot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestRunProgressiveWithLifecycleLoadsAutosavesTranscriptAndDeletesDraft(t *testing.T) {
	dir := t.TempDir()
	draftPath := DraftPath(dir)
	transcriptPath := filepath.Join(dir, ".icot", "transcript.json")
	if err := SaveDraft(draftPath, lifecycleSession{Goal: "loaded"}); err != nil {
		t.Fatal(err)
	}
	hooks := apitools.ProgressiveLoopHooks[lifecycleSession, string, lifecycleArtifacts]{
		Documents: []string{"doc"},
		Opening:   "loaded",
		NoLLM:     true,
		Ready: func(session lifecycleSession, _ []apitools.ReadinessIssue) bool {
			return session.Goal == "loaded"
		},
		FinalConfirm: func(_ *apitools.PromptSession, session *lifecycleSession, _ []string, _ *[]apitools.PromptEvent) (lifecycleArtifacts, error) {
			session.Done = true
			return lifecycleArtifacts{Session: *session}, nil
		},
	}
	artifacts, err := RunProgressiveWithLifecycle(context.Background(), strings.NewReader(""), nil, hooks, ProgressiveLifecycleOptions[lifecycleSession, string, lifecycleArtifacts]{
		ExampleDir:           dir,
		TranscriptPath:       transcriptPath,
		TranscriptVersion:    "test.lifecycle.v1",
		DeleteDraftOnSuccess: true,
		Normalize: func(session *lifecycleSession) {
			session.Goal = strings.TrimSpace(session.Goal)
		},
		Opening: func(session lifecycleSession) string {
			return session.Goal
		},
		LooksLikeSession: func(session lifecycleSession) bool {
			return strings.TrimSpace(session.Goal) != ""
		},
		TranscriptSession: func(artifacts lifecycleArtifacts) any {
			return artifacts.Session
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Session.Goal != "loaded" || !artifacts.Session.Done {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft should be deleted, err=%v", err)
	}
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test.lifecycle.v1") || !strings.Contains(string(data), "loaded") {
		t.Fatalf("transcript = %s", data)
	}
}

type lifecycleSession struct {
	Goal string `json:"goal" yaml:"goal"`
	Done bool   `json:"done" yaml:"done"`
}

type lifecycleArtifacts struct {
	Session lifecycleSession `json:"session"`
}
