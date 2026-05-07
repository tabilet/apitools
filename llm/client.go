// Copyright (c) Greetingland LLC
//
// Package llm provides HTTP clients for Anthropic, OpenAI, and Gemini LLM
// APIs. The base URL for each provider can be overridden by setting the
// matching environment variable (ANTHROPIC_BASE_URL, OPENAI_BASE_URL,
// GEMINI_BASE_URL, COPILOT_API_BASE_URL) so traffic can be routed through a
// local proxy such as copilot-api. Path suffixes (e.g. /v1/messages) stay fixed.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultAnthropicBaseURL  = "https://api.anthropic.com"
	defaultOpenAIBaseURL     = "https://api.openai.com"
	defaultGeminiBaseURL     = "https://generativelanguage.googleapis.com"
	defaultCopilotAPIBaseURL = "http://localhost:4141"

	// DefaultCopilotAPIModel is the local copilot-api model used when callers
	// request provider "copilot-api" without an explicit model.
	DefaultCopilotAPIModel = "gpt-5.4-mini"

	envAnthropicBaseURL  = "ANTHROPIC_BASE_URL"
	envOpenAIBaseURL     = "OPENAI_BASE_URL"
	envGeminiBaseURL     = "GEMINI_BASE_URL"
	envCopilotAPIBaseURL = "COPILOT_API_BASE_URL"

	structuredToolName        = "emit_json"
	structuredToolDescription = "Emit JSON matching the supplied schema."
	structuredSchemaName      = "structured_output"
)

// OpenAIEndpoint selects which OpenAI-compatible HTTP API an OpenAIClient uses.
type OpenAIEndpoint string

const (
	// OpenAIEndpointAuto keeps the provider default endpoint selection.
	OpenAIEndpointAuto OpenAIEndpoint = ""
	// OpenAIEndpointChatCompletions forces POST /v1/chat/completions.
	OpenAIEndpointChatCompletions OpenAIEndpoint = "chat_completions"
	// OpenAIEndpointResponses forces POST /v1/responses.
	OpenAIEndpointResponses OpenAIEndpoint = "responses"
)

func anthropicBaseURL() string { return baseURLFromEnv(envAnthropicBaseURL, defaultAnthropicBaseURL) }
func openaiBaseURL() string    { return baseURLFromEnv(envOpenAIBaseURL, defaultOpenAIBaseURL) }
func geminiBaseURL() string    { return baseURLFromEnv(envGeminiBaseURL, defaultGeminiBaseURL) }
func copilotAPIBaseURL() string {
	return baseURLFromEnv(envCopilotAPIBaseURL, defaultCopilotAPIBaseURL)
}

func baseURLFromEnv(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return strings.TrimRight(v, "/")
	}
	return fallback
}

var defaultLLMTimeout = 60 * time.Second

type llmClientConfig struct {
	httpClient     *http.Client
	timeout        time.Duration
	temperature    *float64
	openAIEndpoint OpenAIEndpoint
}

type ClientOption func(*llmClientConfig)

func defaultLLMClientConfig() llmClientConfig {
	return llmClientConfig{
		httpClient: &http.Client{},
		timeout:    defaultLLMTimeout,
	}
}

// WithHTTPClient configures the HTTP client used by a provider client
// instance. A nil client leaves the default client for that instance unchanged.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(config *llmClientConfig) {
		if client != nil {
			config.httpClient = client
		}
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(config *llmClientConfig) {
		if timeout > 0 {
			config.timeout = timeout
		}
	}
}

func WithTemperature(value float64) ClientOption {
	return func(config *llmClientConfig) {
		config.temperature = &value
	}
}

// WithOpenAIEndpoint forces the OpenAI-compatible endpoint used by OpenAIClient.
// Leave it unset for the provider default: OpenAI uses chat completions, while
// copilot-api selects Responses for known gpt-5 models.
func WithOpenAIEndpoint(endpoint OpenAIEndpoint) ClientOption {
	return func(config *llmClientConfig) {
		config.openAIEndpoint = endpoint
	}
}

// ChatMessage represents a single message in a multi-turn conversation.
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ChatClient defines the interface for multi-turn LLM conversations.
type ChatClient interface {
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
}

// StructuredChatClient extends ChatClient with provider-native structured
// output. It returns a JSON string that conforms to the supplied schema.
type StructuredChatClient interface {
	ChatClient
	StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error)
}

type StructuredOpts struct {
	Temperature *float64
	MaxTokens   int
}

// Client defines the minimal interface common to all providers.
type Client interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// AnthropicClient implements Client for Claude.
type AnthropicClient struct {
	apiKey      string
	model       string
	httpClient  *http.Client
	timeout     time.Duration
	temperature *float64
}

func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	return newAnthropicClient(apiKey, model)
}

func NewAnthropicClientWithOptions(apiKey, model string, opts ...ClientOption) *AnthropicClient {
	return newAnthropicClient(apiKey, model, opts...)
}

func newAnthropicClient(apiKey, model string, opts ...ClientOption) *AnthropicClient {
	config := defaultLLMClientConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &AnthropicClient{
		apiKey:      apiKey,
		model:       model,
		httpClient:  config.httpClient,
		timeout:     config.timeout,
		temperature: config.temperature,
	}
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	apiURL := anthropicBaseURL() + "/v1/messages"
	const version = "2023-06-01"

	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	var systemText string
	apiMessages := make([]apiMsg, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += m.Content
		} else {
			apiMessages = append(apiMessages, apiMsg{Role: m.Role, Content: m.Content})
		}
	}

	type reqShape struct {
		Model       string   `json:"model"`
		MaxTokens   int      `json:"max_tokens"`
		Temperature *float64 `json:"temperature,omitempty"`
		System      string   `json:"system,omitempty"`
		Messages    []apiMsg `json:"messages"`
	}

	reqBody := reqShape{
		Model:       c.model,
		MaxTokens:   8192,
		Temperature: c.temperature,
		System:      systemText,
		Messages:    apiMessages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", version)

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	var result strings.Builder
	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	return result.String(), nil
}

func (c *AnthropicClient) StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	apiURL := anthropicBaseURL() + "/v1/messages"
	const version = "2023-06-01"

	if len(schema) == 0 {
		return "", fmt.Errorf("structured output schema is required")
	}

	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var systemText string
	apiMessages := make([]apiMsg, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += m.Content
		} else {
			apiMessages = append(apiMessages, apiMsg{Role: m.Role, Content: m.Content})
		}
	}

	type toolShape struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		InputSchema json.RawMessage `json:"input_schema"`
	}
	type toolChoiceShape struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	type reqShape struct {
		Model       string          `json:"model"`
		MaxTokens   int             `json:"max_tokens"`
		Temperature *float64        `json:"temperature,omitempty"`
		System      string          `json:"system,omitempty"`
		Messages    []apiMsg        `json:"messages"`
		Tools       []toolShape     `json:"tools"`
		ToolChoice  toolChoiceShape `json:"tool_choice"`
	}
	reqBody := reqShape{
		Model:       c.model,
		MaxTokens:   structuredMaxTokens(opts, 8192),
		Temperature: structuredTemperature(c.temperature, opts),
		System:      systemText,
		Messages:    apiMessages,
		Tools: []toolShape{{
			Name:        structuredToolName,
			Description: structuredToolDescription,
			InputSchema: schema,
		}},
		ToolChoice: toolChoiceShape{Type: "tool", Name: structuredToolName},
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", version)

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if claudeResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}
	for _, content := range claudeResp.Content {
		if content.Type == "tool_use" && content.Name == structuredToolName && len(content.Input) > 0 {
			return string(content.Input), nil
		}
	}
	return "", fmt.Errorf("structured output unsupported: Anthropic response did not include %s tool input", structuredToolName)
}

func (c *AnthropicClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

// OpenAIClient implements Client for GPT models.
type OpenAIClient struct {
	apiKey                string
	model                 string
	baseURL               string
	httpClient            *http.Client
	timeout               time.Duration
	temperature           *float64
	endpoint              OpenAIEndpoint
	inferResponsesByModel bool
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return newOpenAIClient(apiKey, model)
}

func NewOpenAIClientWithOptions(apiKey, model string, opts ...ClientOption) *OpenAIClient {
	return newOpenAIClient(apiKey, model, opts...)
}

func newOpenAIClient(apiKey, model string, opts ...ClientOption) *OpenAIClient {
	return newOpenAICompatibleClient(apiKey, model, openaiBaseURL(), false, opts...)
}

// NewCopilotAPIClient constructs an OpenAI-compatible client for a local
// copilot-api proxy.
func NewCopilotAPIClient(apiKey, model string) *OpenAIClient {
	return newCopilotAPIClient(apiKey, model)
}

// NewCopilotAPIClientWithOptions constructs a copilot-api proxy client with
// optional overrides for HTTP client, timeout, and temperature.
func NewCopilotAPIClientWithOptions(apiKey, model string, opts ...ClientOption) *OpenAIClient {
	return newCopilotAPIClient(apiKey, model, opts...)
}

func newCopilotAPIClient(apiKey, model string, opts ...ClientOption) *OpenAIClient {
	return newOpenAICompatibleClient(apiKey, model, copilotAPIBaseURL(), true, opts...)
}

func newOpenAICompatibleClient(apiKey, model, baseURL string, inferResponsesByModel bool, opts ...ClientOption) *OpenAIClient {
	config := defaultLLMClientConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &OpenAIClient{
		apiKey:                apiKey,
		model:                 model,
		baseURL:               strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient:            config.httpClient,
		timeout:               config.timeout,
		temperature:           config.temperature,
		endpoint:              config.openAIEndpoint,
		inferResponsesByModel: inferResponsesByModel,
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if c.openAIEndpoint() == OpenAIEndpointResponses {
		return c.responsesChat(ctx, messages)
	}
	return c.chatCompletionsChat(ctx, messages)
}

func (c *OpenAIClient) chatCompletionsChat(ctx context.Context, messages []ChatMessage) (string, error) {
	apiURL := c.openAICompatibleBaseURL() + "/v1/chat/completions"

	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	apiMessages := make([]apiMsg, len(messages))
	for i, m := range messages {
		apiMessages[i] = apiMsg{Role: m.Role, Content: m.Content}
	}

	reqBody := struct {
		Model       string   `json:"model"`
		Messages    []apiMsg `json:"messages"`
		Temperature *float64 `json:"temperature,omitempty"`
	}{
		Model:       c.model,
		Messages:    apiMessages,
		Temperature: c.temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", openAIResp.Error.Type, openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	if c.openAIEndpoint() == OpenAIEndpointResponses {
		return c.responsesStructuredChat(ctx, messages, schema, opts)
	}
	return c.chatCompletionsStructuredChat(ctx, messages, schema, opts)
}

func (c *OpenAIClient) chatCompletionsStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	apiURL := c.openAICompatibleBaseURL() + "/v1/chat/completions"

	if len(schema) == 0 {
		return "", fmt.Errorf("structured output schema is required")
	}

	type apiMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	apiMessages := make([]apiMsg, len(messages))
	for i, m := range messages {
		apiMessages[i] = apiMsg{Role: m.Role, Content: m.Content}
	}

	type jsonSchemaShape struct {
		Name   string          `json:"name"`
		Strict bool            `json:"strict"`
		Schema json.RawMessage `json:"schema"`
	}
	type responseFormatShape struct {
		Type       string          `json:"type"`
		JSONSchema jsonSchemaShape `json:"json_schema"`
	}
	reqBody := struct {
		Model          string              `json:"model"`
		Messages       []apiMsg            `json:"messages"`
		Temperature    *float64            `json:"temperature,omitempty"`
		MaxTokens      int                 `json:"max_tokens,omitempty"`
		ResponseFormat responseFormatShape `json:"response_format"`
	}{
		Model:       c.model,
		Messages:    apiMessages,
		Temperature: structuredTemperature(c.temperature, opts),
		MaxTokens:   opts.MaxTokens,
		ResponseFormat: responseFormatShape{
			Type: "json_schema",
			JSONSchema: jsonSchemaShape{
				Name:   structuredSchemaName,
				Strict: true,
				Schema: schema,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if openAIResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", openAIResp.Error.Type, openAIResp.Error.Message)
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return openAIResp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) responsesChat(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := struct {
		Model       string                  `json:"model"`
		Input       []responsesInputMessage `json:"input"`
		Temperature *float64                `json:"temperature,omitempty"`
	}{
		Model:       c.model,
		Input:       responsesInput(messages),
		Temperature: c.temperature,
	}
	return c.doResponsesRequest(ctx, reqBody)
}

func (c *OpenAIClient) responsesStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	if len(schema) == 0 {
		return "", fmt.Errorf("structured output schema is required")
	}
	type jsonSchemaShape struct {
		Type   string          `json:"type"`
		Name   string          `json:"name"`
		Strict bool            `json:"strict"`
		Schema json.RawMessage `json:"schema"`
	}
	reqBody := struct {
		Model           string                  `json:"model"`
		Input           []responsesInputMessage `json:"input"`
		Temperature     *float64                `json:"temperature,omitempty"`
		MaxOutputTokens int                     `json:"max_output_tokens,omitempty"`
		Text            struct {
			Format jsonSchemaShape `json:"format"`
		} `json:"text"`
	}{
		Model:           c.model,
		Input:           responsesInput(messages),
		Temperature:     structuredTemperature(c.temperature, opts),
		MaxOutputTokens: opts.MaxTokens,
	}
	reqBody.Text.Format = jsonSchemaShape{
		Type:   "json_schema",
		Name:   structuredSchemaName,
		Strict: true,
		Schema: schema,
	}
	return c.doResponsesRequest(ctx, reqBody)
}

type responsesInputMessage struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func responsesInput(messages []ChatMessage) []responsesInputMessage {
	input := make([]responsesInputMessage, 0, len(messages))
	for _, message := range messages {
		content := message.Content
		if strings.TrimSpace(content) == "" {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role == "" {
			role = "user"
		}
		input = append(input, responsesInputMessage{
			Role:    role,
			Content: []responsesInputContent{{Type: "input_text", Text: content}},
		})
	}
	return input
}

func (c *OpenAIClient) doResponsesRequest(ctx context.Context, reqBody any) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", c.openAICompatibleBaseURL()+"/v1/responses", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	return parseResponsesOutput(body)
}

func parseResponsesOutput(body []byte) (string, error) {
	var response struct {
		OutputText *string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if response.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", response.Error.Type, response.Error.Message)
	}
	if response.OutputText != nil && strings.TrimSpace(*response.OutputText) != "" {
		return *response.OutputText, nil
	}
	var parts []string
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Text != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return strings.Join(parts, ""), nil
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

func (c *OpenAIClient) openAICompatibleBaseURL() string {
	if baseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/"); baseURL != "" {
		return baseURL
	}
	return openaiBaseURL()
}

func (c *OpenAIClient) openAIEndpoint() OpenAIEndpoint {
	switch c.endpoint {
	case OpenAIEndpointChatCompletions, OpenAIEndpointResponses:
		return c.endpoint
	}
	if c.inferResponsesByModel && openAIModelUsesResponses(c.model) {
		return OpenAIEndpointResponses
	}
	return OpenAIEndpointChatCompletions
}

func openAIModelUsesResponses(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-5")
}

// GeminiClient implements Client for Gemini.
type GeminiClient struct {
	apiKey      string
	model       string
	httpClient  *http.Client
	timeout     time.Duration
	temperature *float64
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	return newGeminiClient(apiKey, model)
}

func NewGeminiClientWithOptions(apiKey, model string, opts ...ClientOption) *GeminiClient {
	return newGeminiClient(apiKey, model, opts...)
}

func newGeminiClient(apiKey, model string, opts ...ClientOption) *GeminiClient {
	config := defaultLLMClientConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &GeminiClient{
		apiKey:      apiKey,
		model:       model,
		httpClient:  config.httpClient,
		timeout:     config.timeout,
		temperature: config.temperature,
	}
}

func (c *GeminiClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	model := strings.TrimPrefix(strings.TrimSpace(c.model), "models/")
	apiURL := fmt.Sprintf(
		"%s/v1beta/models/%s:generateContent?key=%s",
		geminiBaseURL(),
		url.PathEscape(model),
		url.QueryEscape(c.apiKey),
	)

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}

	var systemText string
	contents := make([]content, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += text
		case "assistant":
			contents = append(contents, content{Role: "model", Parts: []part{{Text: text}}})
		default:
			contents = append(contents, content{Role: "user", Parts: []part{{Text: text}}})
		}
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("at least one non-system message is required")
	}

	reqBody := struct {
		SystemInstruction *content  `json:"system_instruction,omitempty"`
		Contents          []content `json:"contents"`
		GenerationConfig  *struct {
			Temperature *float64 `json:"temperature,omitempty"`
		} `json:"generationConfig,omitempty"`
	}{
		Contents: contents,
	}
	if c.temperature != nil {
		reqBody.GenerationConfig = &struct {
			Temperature *float64 `json:"temperature,omitempty"`
		}{Temperature: c.temperature}
	}
	if systemText != "" {
		reqBody.SystemInstruction = &content{Parts: []part{{Text: systemText}}}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if geminiResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", geminiResp.Error.Status, geminiResp.Error.Message)
	}
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	var result strings.Builder
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		result.WriteString(part.Text)
	}
	return result.String(), nil
}

func (c *GeminiClient) StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	if len(schema) == 0 {
		return "", fmt.Errorf("structured output schema is required")
	}

	model := strings.TrimPrefix(strings.TrimSpace(c.model), "models/")
	apiURL := fmt.Sprintf(
		"%s/v1beta/models/%s:generateContent?key=%s",
		geminiBaseURL(),
		url.PathEscape(model),
		url.QueryEscape(c.apiKey),
	)

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	var systemText string
	contents := make([]content, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += text
		case "assistant":
			contents = append(contents, content{Role: "model", Parts: []part{{Text: text}}})
		default:
			contents = append(contents, content{Role: "user", Parts: []part{{Text: text}}})
		}
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("at least one non-system message is required")
	}

	type generationConfigShape struct {
		Temperature        *float64        `json:"temperature,omitempty"`
		MaxOutputTokens    int             `json:"maxOutputTokens,omitempty"`
		ResponseMimeType   string          `json:"responseMimeType"`
		ResponseJSONSchema json.RawMessage `json:"responseJsonSchema"`
	}
	reqBody := struct {
		SystemInstruction *content              `json:"system_instruction,omitempty"`
		Contents          []content             `json:"contents"`
		GenerationConfig  generationConfigShape `json:"generationConfig"`
	}{
		Contents: contents,
		GenerationConfig: generationConfigShape{
			Temperature:        structuredTemperature(c.temperature, opts),
			MaxOutputTokens:    opts.MaxTokens,
			ResponseMimeType:   "application/json",
			ResponseJSONSchema: schema,
		},
	}
	if systemText != "" {
		reqBody.SystemInstruction = &content{Parts: []part{{Text: systemText}}}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}
	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", redactedLLMTransportError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if geminiResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", geminiResp.Error.Status, geminiResp.Error.Message)
	}
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	var result strings.Builder
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		result.WriteString(part.Text)
	}
	return result.String(), nil
}

func (c *GeminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

type llmTransportError struct {
	err     error
	message string
}

func (e *llmTransportError) Error() string {
	return e.message
}

func (e *llmTransportError) Unwrap() error {
	return e.err
}

func redactedLLMTransportError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr != nil {
		redacted := *urlErr
		redacted.URL = redactURLSecret(redacted.URL)
		message = redacted.Error()
	}
	return &llmTransportError{err: err, message: message}
}

func redactURLSecret(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return raw
	}
	query := parsed.Query()
	changed := false
	for _, key := range []string{"key", "api_key", "access_token"} {
		if _, ok := query[key]; ok {
			query.Set(key, "REDACTED")
			changed = true
		}
	}
	if !changed {
		return raw
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (c *AnthropicClient) do(req *http.Request) (*http.Response, error) {
	client := c.httpClient
	if client == nil {
		client = &http.Client{}
	}
	return client.Do(req)
}

func (c *OpenAIClient) do(req *http.Request) (*http.Response, error) {
	client := c.httpClient
	if client == nil {
		client = &http.Client{}
	}
	return client.Do(req)
}

func (c *GeminiClient) do(req *http.Request) (*http.Response, error) {
	client := c.httpClient
	if client == nil {
		client = &http.Client{}
	}
	return client.Do(req)
}

func contextWithLLMDeadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	if timeout <= 0 {
		timeout = defaultLLMTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func structuredTemperature(defaultValue *float64, opts StructuredOpts) *float64 {
	if opts.Temperature != nil {
		return opts.Temperature
	}
	return defaultValue
}

func structuredMaxTokens(opts StructuredOpts, fallback int) int {
	if opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return fallback
}
