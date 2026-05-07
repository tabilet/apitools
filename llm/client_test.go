// Copyright (c) Greetingland LLC

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicBaseURLOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"hi"}]}`))
	}))
	defer srv.Close()

	t.Setenv(envAnthropicBaseURL, srv.URL)
	c := NewAnthropicClient("dummy", "claude-test")
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" {
		t.Fatalf("unexpected response: %q", out)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("expected /v1/messages, got %s", gotPath)
	}
}

func TestOpenAIBaseURLOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()

	t.Setenv(envOpenAIBaseURL, srv.URL)
	c := NewOpenAIClient("dummy", "gpt-test")
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" {
		t.Fatalf("unexpected response: %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("expected /v1/chat/completions, got %s", gotPath)
	}
}

func TestCopilotAPIClientUsesLocalProxyDefault(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if got := r.Header.Get("Authorization"); got != "Bearer dummy" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()

	t.Setenv(envCopilotAPIBaseURL, srv.URL)
	c := NewCopilotAPIClient("dummy", "gpt-4.1")
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" {
		t.Fatalf("unexpected response: %q", out)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("expected /v1/chat/completions, got %s", gotPath)
	}
}

func TestCopilotAPIClientUsesResponsesForGPT5(t *testing.T) {
	var gotPath string
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var payload struct {
			Model string `json:"model"`
			Input []struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		gotModel = payload.Model
		if len(payload.Input) != 1 || payload.Input[0].Role != "user" || payload.Input[0].Content[0].Type != "input_text" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":null,"output":[{"content":[{"type":"output_text","text":"hi"}]}]}`))
	}))
	defer srv.Close()

	t.Setenv(envCopilotAPIBaseURL, srv.URL)
	c := NewCopilotAPIClient("dummy", DefaultCopilotAPIModel)
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" {
		t.Fatalf("unexpected response: %q", out)
	}
	if gotPath != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %s", gotPath)
	}
	if gotModel != DefaultCopilotAPIModel {
		t.Fatalf("model = %q", gotModel)
	}
}

func TestOpenAIEndpointOptionOverridesModelInference(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()

	t.Setenv(envCopilotAPIBaseURL, srv.URL)
	c := NewCopilotAPIClientWithOptions("dummy", DefaultCopilotAPIModel, WithOpenAIEndpoint(OpenAIEndpointChatCompletions))
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" || gotPath != "/v1/chat/completions" {
		t.Fatalf("out/path = %q/%q", out, gotPath)
	}
}

func TestCopilotAPIProviderFromEnvDoesNotRequireKey(t *testing.T) {
	var gotModel string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		gotModel = payload.Model
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":null,"output":[{"content":[{"type":"output_text","text":"hi"}]}]}`))
	}))
	defer srv.Close()

	t.Setenv(envCopilotAPIBaseURL, srv.URL)
	t.Setenv(envCopilotAPIKey, "")
	client, provider, model, err := NewChatClientFromEnv("copilot-api", "")
	if err != nil {
		t.Fatalf("NewChatClientFromEnv: %v", err)
	}
	if provider != "copilot-api" || model != DefaultCopilotAPIModel {
		t.Fatalf("provider/model = %s/%s", provider, model)
	}
	out, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" || gotModel != DefaultCopilotAPIModel || gotPath != "/v1/responses" {
		t.Fatalf("out/model = %q/%q", out, gotModel)
	}
}

func TestGeminiBaseURLOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"hi"}]}}]}`))
	}))
	defer srv.Close()

	t.Setenv(envGeminiBaseURL, srv.URL)
	c := NewGeminiClient("dummy", "gemini-test")
	out, err := c.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if out != "hi" {
		t.Fatalf("unexpected response: %q", out)
	}
	if !strings.HasPrefix(gotPath, "/v1beta/models/gemini-test:") {
		t.Fatalf("unexpected gemini path: %s", gotPath)
	}
}

func TestStructuredChatAnthropicEmitsToolUse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"tool_use","name":"emit_json","input":{"ok":true}}]}`))
	}))
	defer srv.Close()

	t.Setenv(envAnthropicBaseURL, srv.URL)
	c := NewAnthropicClient("dummy", "claude-test")
	out, err := c.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "x"}}, json.RawMessage(`{"type":"object"}`), StructuredOpts{})
	if err != nil {
		t.Fatalf("StructuredChat: %v", err)
	}
	if strings.TrimSpace(out) != `{"ok":true}` {
		t.Fatalf("unexpected output: %q", out)
	}
}
