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
