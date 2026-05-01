package apitools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const (
	JSONCompletionModeStructured = "structured"
	JSONCompletionModeLegacy     = "legacy"
)

// JSONCompletionOptions configures structured-chat fallback behavior.
type JSONCompletionOptions struct {
	LegacyInstruction           string
	FallbackOnStructuredError   bool
	DisableStructuredCompletion bool
}

// JSONCompletionResult reports which completion path was used.
type JSONCompletionResult struct {
	Mode string `json:"mode,omitempty"`
	Raw  string `json:"raw,omitempty"`
}

// CompleteJSONWithFallback tries structured completion first, then optionally
// falls back to legacy chat plus JSON extraction.
func CompleteJSONWithFallback(ctx context.Context, client ChatClient, transcript []TranscriptTurn, schema any, out any, opts JSONCompletionOptions) (JSONCompletionResult, error) {
	if client == nil {
		return JSONCompletionResult{}, fmt.Errorf("chat client is required")
	}
	target, err := jsonCompletionTarget(out)
	if err != nil {
		return JSONCompletionResult{}, err
	}
	if !opts.DisableStructuredCompletion {
		if structured, ok := client.(StructuredChatClient); ok {
			scratch := reflect.New(target.Elem().Type())
			err := structured.CompleteStructured(ctx, transcript, schema, scratch.Interface())
			if err == nil {
				target.Elem().Set(scratch.Elem())
				return JSONCompletionResult{Mode: JSONCompletionModeStructured}, nil
			}
			if !opts.FallbackOnStructuredError {
				return JSONCompletionResult{Mode: JSONCompletionModeStructured}, err
			}
		}
	}
	transcript = AppendLegacyJSONInstruction(transcript, opts.LegacyInstruction)
	reply, err := client.Complete(ctx, transcript)
	if err != nil {
		return JSONCompletionResult{Mode: JSONCompletionModeLegacy}, err
	}
	jsonText, err := ExtractJSONBlock(reply.Content)
	if err != nil {
		return JSONCompletionResult{Mode: JSONCompletionModeLegacy, Raw: reply.Content}, err
	}
	scratch := reflect.New(target.Elem().Type())
	if err := json.Unmarshal([]byte(jsonText), scratch.Interface()); err != nil {
		return JSONCompletionResult{Mode: JSONCompletionModeLegacy, Raw: jsonText}, err
	}
	target.Elem().Set(scratch.Elem())
	return JSONCompletionResult{Mode: JSONCompletionModeLegacy, Raw: jsonText}, nil
}

func jsonCompletionTarget(out any) (reflect.Value, error) {
	if out == nil {
		return reflect.Value{}, fmt.Errorf("completion output target is required")
	}
	target := reflect.ValueOf(out)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return reflect.Value{}, fmt.Errorf("completion output target must be a non-nil pointer")
	}
	return target, nil
}

// ExtractJSONBlock extracts a JSON object from a raw model response.
func ExtractJSONBlock(response string) (string, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return "", fmt.Errorf("empty model response")
	}
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 3 {
			lines = lines[1 : len(lines)-1]
			return strings.TrimSpace(strings.Join(lines, "\n")), nil
		}
	}
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("no JSON object found in model response")
	}
	return response[start : end+1], nil
}

// DecodeJSONBlock extracts and decodes a JSON object from a model response.
func DecodeJSONBlock(raw string, target any) error {
	jsonText, err := ExtractJSONBlock(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonText), target)
}

// AppendLegacyJSONInstruction appends a JSON-only instruction to the last user
// turn if one is not already present.
func AppendLegacyJSONInstruction(transcript []TranscriptTurn, instruction string) []TranscriptTurn {
	if instruction == "" {
		instruction = "Return only JSON. Do not include Markdown."
	}
	out := append([]TranscriptTurn(nil), transcript...)
	for i := range out {
		if strings.Contains(out[i].Content, instruction) {
			return out
		}
	}
	for i := len(out) - 1; i >= 0; i-- {
		if strings.TrimSpace(out[i].Role) == "user" {
			out[i].Content = strings.TrimSpace(out[i].Content) + "\n\n" + instruction
			return out
		}
	}
	return out
}

// RenderTranscriptSnapshot renders a markdown-ish transcript snapshot.
func RenderTranscriptSnapshot(transcript []TranscriptTurn) string {
	var b strings.Builder
	for _, turn := range transcript {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", strings.ToUpper(strings.TrimSpace(turn.Role)), strings.TrimSpace(turn.Content))
	}
	return strings.TrimSpace(b.String())
}

// RawSchema normalizes supported schema inputs to JSON bytes.
func RawSchema(schema any) (json.RawMessage, error) {
	switch typed := schema.(type) {
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...), nil
	case []byte:
		return append(json.RawMessage(nil), typed...), nil
	case string:
		return json.RawMessage(typed), nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(data), nil
	}
}
