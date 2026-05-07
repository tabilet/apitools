package apitools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/apitools/internal/atomicfile"
)

// PromptTurn records one local prompt and answer.
type PromptTurn struct {
	Label  string `json:"label"`
	Answer string `json:"answer"`
}

// PromptEvent records a structured event from an interactive authoring loop.
type PromptEvent struct {
	Kind string `json:"kind"`
	Data any    `json:"data,omitempty"`
}

// PromptTranscript is a persisted local transcript for replay and review.
type PromptTranscript struct {
	Version string        `json:"version"`
	TimeUTC string        `json:"time_utc"`
	Turns   []PromptTurn  `json:"turns"`
	Events  []PromptEvent `json:"events,omitempty"`
	Session any           `json:"session,omitempty"`
}

// ReplayScript is a deterministic prompt replay fixture.
type ReplayScript struct {
	Turns []PromptTurn `json:"turns"`
	Input string       `json:"input"`
}

// PromptSession prompts on a reader/writer pair and records prompt turns.
type PromptSession struct {
	reader *bufio.Reader
	out    io.Writer
	turns  []PromptTurn
}

// NewPromptSession creates a local prompt session.
func NewPromptSession(in io.Reader, out io.Writer) *PromptSession {
	if in == nil {
		in = strings.NewReader("")
	}
	if out == nil {
		out = io.Discard
	}
	if reader, ok := in.(*bufio.Reader); ok {
		return &PromptSession{reader: reader, out: out}
	}
	return &PromptSession{reader: bufio.NewReader(in), out: out}
}

// Ask prompts for a required free-form value.
func (session *PromptSession) Ask(label string) (string, error) {
	fmt.Fprintf(session.out, "%s: ", label)
	value, err := session.next()
	session.record(label, value, err)
	return value, err
}

// AskDefault prompts for a value, returning current when the answer is blank.
func (session *PromptSession) AskDefault(label, current string) (string, error) {
	current = strings.TrimSpace(current)
	if current == "" {
		fmt.Fprintf(session.out, "%s: ", label)
	} else {
		fmt.Fprintf(session.out, "%s [%s]: ", label, OneLine(current))
	}
	value, err := session.next()
	if err != nil {
		session.record(label, value, err)
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		session.record(label, current, nil)
		return current, nil
	}
	session.record(label, value, nil)
	return value, nil
}

// AskDefaultRequired prompts until a non-empty value is available.
func (session *PromptSession) AskDefaultRequired(label, current string) (string, error) {
	for {
		value, err := session.AskDefault(label, current)
		if err != nil {
			return "", fmt.Errorf("%s: %w", label, err)
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, nil
		}
		fmt.Fprintf(session.out, "%s is required.\n", label)
	}
}

// AskYesNo prompts for a yes/no answer with a default.
func (session *PromptSession) AskYesNo(label string, defaultYes bool) (bool, error) {
	suffix := "y/N"
	if defaultYes {
		suffix = "Y/n"
	}
	for {
		fmt.Fprintf(session.out, "%s [%s]: ", label, suffix)
		value, err := session.next()
		if err != nil {
			session.record(label, value, err)
			return false, err
		}
		raw := value
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			session.record(label, raw, nil)
			return defaultYes, nil
		}
		switch value {
		case "y", "yes", "true", "allow", "allowed", "approve", "approved":
			session.record(label, raw, nil)
			return true, nil
		case "n", "no", "false", "deny", "denied":
			session.record(label, raw, nil)
			return false, nil
		default:
			session.record(label, raw, nil)
			fmt.Fprintln(session.out, "Please answer yes or no.")
		}
	}
}

// Turns returns a copy of recorded prompt turns.
func (session *PromptSession) Turns() []PromptTurn {
	if session == nil {
		return nil
	}
	return append([]PromptTurn(nil), session.turns...)
}

func (session *PromptSession) next() (string, error) {
	line, err := session.reader.ReadString('\n')
	if err == io.EOF && line != "" {
		return strings.TrimRight(line, "\r\n"), nil
	}
	if err != nil {
		if err == io.EOF {
			return "", io.ErrUnexpectedEOF
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func (session *PromptSession) record(label, answer string, err error) {
	if err != nil && strings.TrimSpace(answer) == "" {
		return
	}
	session.turns = append(session.turns, PromptTurn{Label: label, Answer: answer})
}

// OneLine normalizes a prompt default for display.
func OneLine(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

// AssertPromptLabelsInOrder verifies that prompt labels were emitted in replay
// order.
func AssertPromptLabelsInOrder(output string, turns []PromptTurn) error {
	offset := 0
	for _, turn := range turns {
		index := strings.Index(output[offset:], turn.Label)
		if index < 0 {
			return fmt.Errorf("prompt label %q not found after offset %d", turn.Label, offset)
		}
		offset += index + len(turn.Label)
	}
	return nil
}

// SavePromptTranscript writes a prompt transcript with private-file
// permissions. Empty paths are ignored.
func SavePromptTranscript(path, version string, turns []PromptTurn, events []PromptEvent, session any) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if strings.TrimSpace(version) == "" {
		version = "apitools.prompt-transcript.v1"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	transcript := PromptTranscript{
		Version: version,
		TimeUTC: time.Now().UTC().Format(time.RFC3339),
		Turns:   append([]PromptTurn(nil), turns...),
		Events:  append([]PromptEvent(nil), events...),
		Session: session,
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(path, data, 0o600)
}

// InteractiveDraftRequest is the model-facing input for an interactive draft.
type InteractiveDraftRequest[S, D any] struct {
	Opening           string           `json:"opening"`
	Brief             string           `json:"brief,omitempty"`
	Session           S                `json:"session"`
	Docs              []D              `json:"docs"`
	TranscriptTurns   []PromptTurn     `json:"transcript_turns,omitempty"`
	ReadinessFeedback []ReadinessIssue `json:"readiness_feedback,omitempty"`
}

// InteractiveExtractor provides optional AI assistance for an interactive
// authoring loop.
type InteractiveExtractor[S, D any] interface {
	Kickoff(context.Context, string) (S, error)
	Draft(context.Context, InteractiveDraftRequest[S, D]) (S, error)
	Refine(context.Context, S) (S, error)
	Disambiguate(context.Context, string, []D) ([]string, error)
}

// NoopInteractiveExtractor disables AI assistance.
type NoopInteractiveExtractor[S, D any] struct{}

func (NoopInteractiveExtractor[S, D]) noopInteractiveExtractor() {}

func (NoopInteractiveExtractor[S, D]) Kickoff(context.Context, string) (S, error) {
	var zero S
	return zero, nil
}

func (NoopInteractiveExtractor[S, D]) Draft(context.Context, InteractiveDraftRequest[S, D]) (S, error) {
	var zero S
	return zero, nil
}

func (NoopInteractiveExtractor[S, D]) Refine(_ context.Context, session S) (S, error) {
	return session, nil
}

func (NoopInteractiveExtractor[S, D]) Disambiguate(context.Context, string, []D) ([]string, error) {
	return nil, nil
}

// ProgressiveLoopHooks supplies product-specific behavior for the generic iCoT
// loop.
type ProgressiveLoopHooks[S, D, A any] struct {
	Session       S
	Documents     []D
	Opening       string
	Brief         string
	NoLLM         bool
	MaxAttempts   int
	OpeningPrompt string

	Extractor InteractiveExtractor[S, D]

	Normalize            func(*S)
	ApplyOpeningAnswer   func(*S, string, []D) error
	Autosave             func(S) error
	RankDocuments        func([]D, []string) []D
	DeterministicPrefill func(*S, []D) bool
	LooksLikeSession     func(S) bool
	MergeDraft           func(S, S, []D) S
	AfterDraft           func(S) error
	DraftResultSummary   func(S) any
	OnDraftError         func(error)
	CheckReadiness       func(S, []D) []ReadinessIssue
	Ready                func(S, []ReadinessIssue) bool
	PlanQuestion         func(S, []D, []ReadinessIssue) InteractiveQuestion
	ApplyAnswer          func(*S, InteractiveQuestion, string, []D) error
	FinalConfirm         func(*PromptSession, *S, []D, *[]PromptEvent) (A, error)
	FinalResultSummary   func(A) any
	SaveTranscript       func([]PromptTurn, []PromptEvent, A) error
}

// RunProgressiveICOT runs the domain-neutral progressive iCoT control loop.
func RunProgressiveICOT[S, D, A any](ctx context.Context, in io.Reader, out io.Writer, hooks ProgressiveLoopHooks[S, D, A]) (A, error) {
	var zero A
	prompts := NewPromptSession(in, out)
	extractor := hooks.Extractor
	if extractor == nil {
		extractor = NoopInteractiveExtractor[S, D]{}
	}
	_, noopExtractor := extractor.(interface{ noopInteractiveExtractor() })
	attempts := hooks.MaxAttempts
	if attempts <= 0 {
		attempts = 20
	}
	session := hooks.Session
	docs := append([]D(nil), hooks.Documents...)
	if hooks.Normalize != nil {
		hooks.Normalize(&session)
	}
	var events []PromptEvent
	record := func(kind string, data any) {
		events = append(events, PromptEvent{Kind: kind, Data: data})
	}

	opening := strings.TrimSpace(hooks.Opening)
	if opening == "" {
		if hooks.OpeningPrompt != "" {
			fmt.Fprintln(out, hooks.OpeningPrompt)
		}
		answer, err := prompts.Ask("Workflow goal")
		if err != nil {
			return zero, err
		}
		opening = strings.TrimSpace(answer)
		if hooks.ApplyOpeningAnswer != nil {
			if err := hooks.ApplyOpeningAnswer(&session, opening, docs); err != nil {
				return zero, err
			}
		}
		if hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		if hooks.Autosave != nil {
			if err := hooks.Autosave(session); err != nil {
				return zero, err
			}
		}
		record("progressive_question", InteractiveQuestion{Prompt: "Workflow goal", Slots: []string{"workflow.goal"}})
		record("progressive_answer", PromptTurn{Label: "Workflow goal", Answer: answer})
	}
	if !hooks.NoLLM && len(docs) > 1 && opening != "" {
		ranked, err := extractor.Disambiguate(ctx, opening, docs)
		if err == nil && hooks.RankDocuments != nil {
			docs = hooks.RankDocuments(docs, ranked)
		} else if err != nil && hooks.OnDraftError != nil {
			hooks.OnDraftError(fmt.Errorf("OpenAPI ranking skipped: %w", err))
		}
	}

	var issues []ReadinessIssue
	for attempt := 0; attempt < attempts; attempt++ {
		if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		request := InteractiveDraftRequest[S, D]{
			Opening:           opening,
			Brief:             hooks.Brief,
			Session:           session,
			Docs:              docs,
			TranscriptTurns:   prompts.Turns(),
			ReadinessFeedback: append([]ReadinessIssue(nil), issues...),
		}
		record("model_draft_call", map[string]any{
			"opening":          request.Opening,
			"turn_count":       len(request.TranscriptTurns),
			"readiness_issues": request.ReadinessFeedback,
		})
		var (
			draft    S
			draftErr error
		)
		if !noopExtractor {
			draft, draftErr = extractor.Draft(ctx, request)
		}
		if !noopExtractor && draftErr == nil && (hooks.LooksLikeSession == nil || hooks.LooksLikeSession(draft)) {
			if hooks.MergeDraft != nil {
				session = hooks.MergeDraft(session, draft, docs)
			} else {
				session = draft
			}
			if hooks.Normalize != nil {
				hooks.Normalize(&session)
			}
			if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
				hooks.Normalize(&session)
			}
			if hooks.DraftResultSummary != nil {
				record("model_draft_result", hooks.DraftResultSummary(session))
			}
			if hooks.Autosave != nil {
				if err := hooks.Autosave(session); err != nil {
					return zero, err
				}
			}
			if hooks.AfterDraft != nil {
				if err := hooks.AfterDraft(session); err != nil {
					return zero, err
				}
			}
		} else if !noopExtractor && draftErr != nil {
			record("model_draft_error", draftErr.Error())
			if hooks.OnDraftError != nil {
				hooks.OnDraftError(draftErr)
			}
		}

		if hooks.CheckReadiness != nil {
			issues = hooks.CheckReadiness(session, docs)
		}
		record("readiness_decision", issues)
		if hooks.Ready != nil && hooks.Ready(session, issues) {
			record("next_question_decision", InteractiveQuestion{
				Prompt:          "Confirm first valid intent",
				SuggestedAnswer: "save",
				Slots:           []string{"confirmation"},
			})
			if hooks.FinalConfirm == nil {
				return zero, fmt.Errorf("final confirmation hook is required")
			}
			artifacts, err := hooks.FinalConfirm(prompts, &session, docs, &events)
			if err == nil {
				if hooks.FinalResultSummary != nil {
					record("final_generated_artifacts", hooks.FinalResultSummary(artifacts))
				}
				if hooks.SaveTranscript != nil {
					if saveErr := hooks.SaveTranscript(prompts.Turns(), events, artifacts); saveErr != nil {
						return artifacts, saveErr
					}
				}
			}
			return artifacts, err
		}
		if hooks.PlanQuestion == nil || hooks.ApplyAnswer == nil {
			return zero, fmt.Errorf("question planning and answer hooks are required")
		}
		question := hooks.PlanQuestion(session, docs, issues)
		record("next_question_decision", question)
		answer, err := prompts.AskDefault(question.Prompt, question.SuggestedAnswer)
		if err != nil {
			return zero, err
		}
		if strings.EqualFold(strings.TrimSpace(answer), "cancel") {
			return zero, ErrCanceled
		}
		if err := hooks.ApplyAnswer(&session, question, answer, docs); err != nil {
			return zero, err
		}
		if hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		record("progressive_question", question)
		record("progressive_answer", PromptTurn{Label: question.Prompt, Answer: answer})
		if hooks.Autosave != nil {
			if err := hooks.Autosave(session); err != nil {
				return zero, err
			}
		}
	}
	return zero, fmt.Errorf("progressive iCoT could not reach a valid intent after %d draft attempts", attempts)
}

// ErrCanceled reports user cancellation from a generic interactive loop.
var ErrCanceled = fmt.Errorf("interactive authoring canceled")
