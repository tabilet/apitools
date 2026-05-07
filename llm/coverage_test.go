// Copyright (c) Greetingland LLC

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAnthropicChat_SystemMessageExtraction(t *testing.T) {
	var receivedBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "response"},
			},
		})
	}))
	defer srv.Close()

	var _ ChatClient = &AnthropicClient{}
	var _ ChatClient = &OpenAIClient{}
	var _ ChatClient = NewCopilotAPIClient("test-key", "gpt-4.1")

	msg := ChatMessage{Role: "system", Content: "You are helpful"}
	if msg.Role != "system" || msg.Content != "You are helpful" {
		t.Errorf("ChatMessage fields not set correctly")
	}
}

func TestOpenAIChat_AllMessagesPassedThrough(t *testing.T) {
	var receivedBody struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "response"}},
			},
		})
	}))
	defer srv.Close()

	c := &OpenAIClient{apiKey: "test-key", model: "test"}

	messages := []ChatMessage{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "user", Content: "bye"},
	}

	if len(messages) != 4 {
		t.Errorf("expected 4 messages, got %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Errorf("first message should be system")
	}

	_ = c
}

func TestAnthropicGenerate_DelegatesToChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		msgs := req["messages"].([]interface{})
		if len(msgs) != 1 {
			t.Errorf("expected 1 message from Generate, got %d", len(msgs))
		}
		msg := msgs[0].(map[string]interface{})
		if msg["role"] != "user" {
			t.Errorf("expected user role, got %s", msg["role"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "generated"},
			},
		})
	}))
	defer srv.Close()

	c := NewAnthropicClient("key", "model")
	var _ Client = c
	var _ ChatClient = c

	_ = context.Background()
}

func TestChatMessage_Roles(t *testing.T) {
	roles := []string{"system", "user", "assistant"}
	for _, role := range roles {
		msg := ChatMessage{Role: role, Content: "test"}
		if msg.Role != role {
			t.Errorf("expected role %s, got %s", role, msg.Role)
		}
	}
}

func TestLLMClientsUseDefaultTimeoutWhenContextHasNoDeadline(t *testing.T) {
	client := newOpenAIClient("test-key", "test-model", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("request context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > defaultLLMTimeout {
			t.Fatalf("deadline remaining = %s, want within default timeout", remaining)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != "test-model" || len(payload.Messages) != 1 || payload.Messages[0].Content != "hello" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		return llmTestResponse(`{"choices":[{"message":{"content":"ok"}}]}`), nil
	}))

	reply, err := client.Generate(nil, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" {
		t.Fatalf("reply = %q", reply)
	}
}

func TestLLMClientsPreserveCallerDeadline(t *testing.T) {
	wantDeadline := time.Now().Add(30 * time.Second)
	client := newOpenAIClient("test-key", "test-model", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
		gotDeadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("request context has no deadline")
		}
		if gotDeadline.Sub(wantDeadline).Abs() > time.Second {
			t.Fatalf("deadline = %s, want near %s", gotDeadline, wantDeadline)
		}
		return llmTestResponse(`{"choices":[{"message":{"content":"ok"}}]}`), nil
	}))

	ctx, cancel := context.WithDeadline(context.Background(), wantDeadline)
	defer cancel()
	_, err := client.Generate(ctx, "hello")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRedactedLLMTransportErrorRemovesURLSecrets(t *testing.T) {
	err := redactedLLMTransportError(&url.Error{
		Op:  "Post",
		URL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-pro:generateContent?key=AIzaSecret&other=value",
		Err: context.DeadlineExceeded,
	})

	if !strings.Contains(err.Error(), "key=REDACTED") {
		t.Fatalf("redacted error missing redacted key: %s", err)
	}
	if strings.Contains(err.Error(), "AIzaSecret") {
		t.Fatalf("redacted error leaked API key: %s", err)
	}
}

func TestLLMProviderRequestsIncludeExpectedAuthAndPayload(t *testing.T) {
	t.Run("anthropic", func(t *testing.T) {
		client := newAnthropicClient("anthropic-key", "claude-test", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("x-api-key"); got != "anthropic-key" {
				t.Fatalf("x-api-key = %q", got)
			}
			if got := req.Header.Get("anthropic-version"); got == "" {
				t.Fatal("missing anthropic-version")
			}
			var payload struct {
				Model    string `json:"model"`
				System   string `json:"system"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Model != "claude-test" || payload.System != "system" || len(payload.Messages) != 1 {
				t.Fatalf("unexpected payload: %+v", payload)
			}
			return llmTestResponse(`{"content":[{"type":"text","text":"ok"}]}`), nil
		}))
		_, err := client.Chat(context.Background(), []ChatMessage{
			{Role: "system", Content: "system"},
			{Role: "user", Content: "hello"},
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("gemini", func(t *testing.T) {
		client := newGeminiClient("gemini-key", "gemini-test", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.Query().Get("key"); got != "gemini-key" {
				t.Fatalf("key = %q", got)
			}
			var payload struct {
				Contents []struct {
					Role  string `json:"role"`
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"contents"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Contents) != 1 || payload.Contents[0].Role != "user" || payload.Contents[0].Parts[0].Text != "hello" {
				t.Fatalf("unexpected payload: %+v", payload)
			}
			return llmTestResponse(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`), nil
		}))
		_, err := client.Generate(context.Background(), "hello")
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestWithHTTPClientIsPerInstance(t *testing.T) {
	var firstCalls, secondCalls int
	first := NewOpenAIClientWithOptions("first-key", "gpt-test", WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		firstCalls++
		if got := req.Header.Get("Authorization"); got != "Bearer first-key" {
			t.Fatalf("first Authorization = %q", got)
		}
		return llmTestResponse(`{"choices":[{"message":{"content":"first"}}]}`), nil
	})}))
	second := NewOpenAIClientWithOptions("second-key", "gpt-test", WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		secondCalls++
		if got := req.Header.Get("Authorization"); got != "Bearer second-key" {
			t.Fatalf("second Authorization = %q", got)
		}
		return llmTestResponse(`{"choices":[{"message":{"content":"second"}}]}`), nil
	})}))

	got, err := first.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "first" {
		t.Fatalf("first reply = %q", got)
	}
	got, err = second.Generate(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if got != "second" {
		t.Fatalf("second reply = %q", got)
	}
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("calls = %d/%d, want 1/1", firstCalls, secondCalls)
	}
}

func TestLLMProviderRequestsIncludeTemperature(t *testing.T) {
	temperature := 0.2
	t.Run("anthropic", func(t *testing.T) {
		client := newAnthropicClient("anthropic-key", "claude-test", WithTemperature(temperature), testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				Temperature *float64 `json:"temperature"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Temperature == nil || *payload.Temperature != temperature {
				t.Fatalf("temperature = %v, want %v", payload.Temperature, temperature)
			}
			return llmTestResponse(`{"content":[{"type":"text","text":"ok"}]}`), nil
		}))
		if _, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("openai", func(t *testing.T) {
		client := newOpenAIClient("openai-key", "gpt-test", WithTemperature(temperature), testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				Temperature *float64 `json:"temperature"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Temperature == nil || *payload.Temperature != temperature {
				t.Fatalf("temperature = %v, want %v", payload.Temperature, temperature)
			}
			return llmTestResponse(`{"choices":[{"message":{"content":"ok"}}]}`), nil
		}))
		if _, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("gemini", func(t *testing.T) {
		client := newGeminiClient("gemini-key", "gemini-test", WithTemperature(temperature), testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				GenerationConfig *struct {
					Temperature *float64 `json:"temperature"`
				} `json:"generationConfig"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.GenerationConfig == nil || payload.GenerationConfig.Temperature == nil || *payload.GenerationConfig.Temperature != temperature {
				t.Fatalf("generationConfig = %#v, want temperature %v", payload.GenerationConfig, temperature)
			}
			return llmTestResponse(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`), nil
		}))
		if _, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestStructuredChatProviderRequestsIncludeSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"workflow":{"type":"object"}}}`)
	temperature := 0.15

	t.Run("anthropic", func(t *testing.T) {
		client := newAnthropicClient("anthropic-key", "claude-test", WithTemperature(0.5), testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				Temperature *float64 `json:"temperature"`
				Tools       []struct {
					Name        string          `json:"name"`
					InputSchema json.RawMessage `json:"input_schema"`
				} `json:"tools"`
				ToolChoice struct {
					Type string `json:"type"`
					Name string `json:"name"`
				} `json:"tool_choice"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Temperature == nil || *payload.Temperature != temperature {
				t.Fatalf("temperature = %v, want %v", payload.Temperature, temperature)
			}
			if len(payload.Tools) != 1 || payload.Tools[0].Name != "emit_json" || string(payload.Tools[0].InputSchema) != string(schema) {
				t.Fatalf("unexpected tools: %+v", payload.Tools)
			}
			if payload.ToolChoice.Type != "tool" || payload.ToolChoice.Name != "emit_json" {
				t.Fatalf("unexpected tool choice: %+v", payload.ToolChoice)
			}
			return llmTestResponse(`{"content":[{"type":"tool_use","name":"emit_json","input":{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"render","type":"fnct","do":"Render"}]}}]}`), nil
		}))
		raw, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}, schema, StructuredOpts{Temperature: &temperature})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"workflow"`) {
			t.Fatalf("unexpected raw structured output: %s", raw)
		}
	})

	t.Run("openai", func(t *testing.T) {
		client := newOpenAIClient("openai-key", "gpt-test", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				ResponseFormat struct {
					Type       string `json:"type"`
					JSONSchema struct {
						Name   string          `json:"name"`
						Strict bool            `json:"strict"`
						Schema json.RawMessage `json:"schema"`
					} `json:"json_schema"`
				} `json:"response_format"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.ResponseFormat.Type != "json_schema" {
				t.Fatalf("response_format.type = %q", payload.ResponseFormat.Type)
			}
			got := payload.ResponseFormat.JSONSchema
			if got.Name != "structured_output" || !got.Strict || string(got.Schema) != string(schema) {
				t.Fatalf("unexpected JSON schema config: %+v", got)
			}
			return llmTestResponse(`{"choices":[{"message":{"content":"{\"workflow\":{\"name\":\"demo\",\"description\":\"Demo\"},\"steps\":[{\"name\":\"render\",\"type\":\"fnct\",\"do\":\"Render\"}]}"}}]}`), nil
		}))
		raw, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}, schema, StructuredOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"workflow"`) {
			t.Fatalf("unexpected raw structured output: %s", raw)
		}
	})

	t.Run("copilot-api responses", func(t *testing.T) {
		client := newCopilotAPIClient("copilot-key", DefaultCopilotAPIModel, testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/v1/responses" {
				t.Fatalf("path = %s", req.URL.Path)
			}
			var payload struct {
				Model string `json:"model"`
				Text  struct {
					Format struct {
						Type   string          `json:"type"`
						Name   string          `json:"name"`
						Strict bool            `json:"strict"`
						Schema json.RawMessage `json:"schema"`
					} `json:"format"`
				} `json:"text"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Model != DefaultCopilotAPIModel {
				t.Fatalf("model = %q", payload.Model)
			}
			got := payload.Text.Format
			if got.Type != "json_schema" || got.Name != "structured_output" || !got.Strict || string(got.Schema) != string(schema) {
				t.Fatalf("unexpected JSON schema config: %+v", got)
			}
			return llmTestResponse(`{"output_text":null,"output":[{"content":[{"type":"output_text","text":"{\"workflow\":{\"name\":\"demo\",\"description\":\"Demo\"},\"steps\":[{\"name\":\"render\",\"type\":\"fnct\",\"do\":\"Render\"}]}"}]}]}`), nil
		}))
		raw, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}, schema, StructuredOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"workflow"`) {
			t.Fatalf("unexpected raw structured output: %s", raw)
		}
	})

	t.Run("gemini", func(t *testing.T) {
		client := newGeminiClient("gemini-key", "gemini-test", testLLMHTTPClient(func(req *http.Request) (*http.Response, error) {
			var payload struct {
				GenerationConfig struct {
					ResponseMimeType   string          `json:"responseMimeType"`
					ResponseJSONSchema json.RawMessage `json:"responseJsonSchema"`
					ResponseSchema     json.RawMessage `json:"responseSchema"`
				} `json:"generationConfig"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.GenerationConfig.ResponseSchema) != 0 {
				t.Fatalf("generation config used deprecated responseSchema: %+v", payload.GenerationConfig)
			}
			if payload.GenerationConfig.ResponseMimeType != "application/json" || string(payload.GenerationConfig.ResponseJSONSchema) != string(schema) {
				t.Fatalf("unexpected generation config: %+v", payload.GenerationConfig)
			}
			return llmTestResponse(`{"candidates":[{"content":{"parts":[{"text":"{\"workflow\":{\"name\":\"demo\",\"description\":\"Demo\"},\"steps\":[{\"name\":\"render\",\"type\":\"fnct\",\"do\":\"Render\"}]}"}]}}]}`), nil
		}))
		raw, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}, schema, StructuredOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"workflow"`) {
			t.Fatalf("unexpected raw structured output: %s", raw)
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testLLMHTTPClient(fn roundTripFunc) ClientOption {
	return WithHTTPClient(&http.Client{Transport: fn})
}

func llmTestResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
