package apitools

import (
	"context"
	"encoding/json"
	"fmt"

	llmpkg "github.com/OpenUdon/apitools/llm"
)

// LLMChatAdapter adapts apitools/llm provider clients to the root authoring
// ChatClient interfaces used by CompleteJSONWithFallback.
type LLMChatAdapter struct {
	Client      llmpkg.ChatClient
	Temperature *float64
	MaxTokens   int
}

func (adapter LLMChatAdapter) Complete(ctx context.Context, transcript []TranscriptTurn) (TranscriptTurn, error) {
	if adapter.Client == nil {
		return TranscriptTurn{}, fmt.Errorf("llm chat client is required")
	}
	content, err := adapter.Client.Chat(ctx, transcriptToLLMMessages(transcript))
	if err != nil {
		return TranscriptTurn{}, err
	}
	return TranscriptTurn{Role: "assistant", Content: content}, nil
}

func (adapter LLMChatAdapter) CompleteStructured(ctx context.Context, transcript []TranscriptTurn, schema any, out any) error {
	if adapter.Client == nil {
		return fmt.Errorf("llm chat client is required")
	}
	structured, ok := adapter.Client.(llmpkg.StructuredChatClient)
	if !ok {
		return fmt.Errorf("structured chat unavailable")
	}
	rawSchema, err := RawSchema(schema)
	if err != nil {
		return err
	}
	raw, err := structured.StructuredChat(ctx, transcriptToLLMMessages(transcript), rawSchema, llmpkg.StructuredOpts{
		Temperature: adapter.Temperature,
		MaxTokens:   adapter.MaxTokens,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

func transcriptToLLMMessages(transcript []TranscriptTurn) []llmpkg.ChatMessage {
	messages := make([]llmpkg.ChatMessage, 0, len(transcript))
	for _, turn := range transcript {
		messages = append(messages, llmpkg.ChatMessage{Role: turn.Role, Content: turn.Content})
	}
	return messages
}
