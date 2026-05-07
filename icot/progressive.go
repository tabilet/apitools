package icot

import (
	"context"
	"io"
	"strings"

	"github.com/OpenUdon/apitools"
)

// ProgressiveLifecycleOptions adds shared draft/transcript lifecycle behavior
// around apitools.RunProgressiveICOT.
type ProgressiveLifecycleOptions[S, D, A any] struct {
	ExampleDir           string
	DraftPath            string
	TranscriptPath       string
	TranscriptVersion    string
	DeleteDraftOnSuccess bool
	Normalize            func(*S)
	LooksLikeSession     func(S) bool
	Opening              func(S) string
	TranscriptSession    func(A) any
}

// RunProgressiveWithLifecycle binds caller-specific progressive iCoT hooks to
// the shared draft, autosave, transcript, and cleanup lifecycle.
func RunProgressiveWithLifecycle[S, D, A any](ctx context.Context, in io.Reader, out io.Writer, hooks apitools.ProgressiveLoopHooks[S, D, A], opts ProgressiveLifecycleOptions[S, D, A]) (A, error) {
	draftPath := strings.TrimSpace(opts.DraftPath)
	if draftPath == "" && strings.TrimSpace(opts.ExampleDir) != "" {
		draftPath = DraftPath(opts.ExampleDir)
	}

	session := hooks.Session
	if draftPath != "" {
		loaded, ok, err := LoadDraft[S](draftPath)
		if err != nil {
			var zero A
			return zero, err
		}
		if ok && (opts.LooksLikeSession == nil || opts.LooksLikeSession(loaded)) {
			session = loaded
		}
	}
	if opts.Normalize != nil {
		opts.Normalize(&session)
	}
	hooks.Session = session
	if strings.TrimSpace(hooks.Opening) == "" && opts.Opening != nil {
		hooks.Opening = opts.Opening(session)
	}

	baseAutosave := hooks.Autosave
	hooks.Autosave = func(session S) error {
		if baseAutosave != nil {
			if err := baseAutosave(session); err != nil {
				return err
			}
		}
		if draftPath == "" {
			return nil
		}
		if opts.LooksLikeSession != nil && !opts.LooksLikeSession(session) {
			return nil
		}
		if opts.Normalize != nil {
			opts.Normalize(&session)
		}
		return SaveDraft(draftPath, session)
	}

	baseTranscript := hooks.SaveTranscript
	hooks.SaveTranscript = func(turns []apitools.PromptTurn, events []apitools.PromptEvent, artifacts A) error {
		if baseTranscript != nil {
			if err := baseTranscript(turns, events, artifacts); err != nil {
				return err
			}
		}
		if strings.TrimSpace(opts.TranscriptPath) == "" {
			return nil
		}
		version := opts.TranscriptVersion
		if strings.TrimSpace(version) == "" {
			version = "apitools.icot-transcript.v1"
		}
		var transcriptSession any = artifacts
		if opts.TranscriptSession != nil {
			transcriptSession = opts.TranscriptSession(artifacts)
		}
		return apitools.SavePromptTranscript(opts.TranscriptPath, version, turns, events, transcriptSession)
	}

	artifacts, err := apitools.RunProgressiveICOT(ctx, in, out, hooks)
	if err == nil && opts.DeleteDraftOnSuccess {
		err = DeleteDraft(draftPath)
	}
	return artifacts, err
}
