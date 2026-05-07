// Copyright (c) Greetingland LLC

package llm

import (
	"fmt"
	"os"
	"strings"
)

const (
	envDefaultModel       = "LLM_MODEL"
	envLegacyRolloutModel = "ROLLOUT_MODEL"
	envCopilotAPIKey      = "COPILOT_API_KEY"
)

// Options carries optional knobs (currently just temperature) for the
// env-driven factory functions below.
type Options struct {
	Temperature *float64
}

// NewClientFromEnv constructs a provider client using the matching API-key
// environment variable. Provider names are case-insensitive; "anthropic" is
// the default when blank. "copilot-api" uses an OpenAI-compatible local proxy
// and falls back to a dummy token when COPILOT_API_KEY is unset. The returned
// model is the resolved string actually used (after defaults and environment
// fallbacks).
func NewClientFromEnv(provider, model string) (Client, string, string, error) {
	return NewClientFromEnvWithOptions(provider, model, Options{})
}

func NewClientFromEnvWithOptions(provider, model string, opts Options) (Client, string, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "anthropic"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModelFromEnv()
	}
	switch provider {
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, "", "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		if model == "" {
			model = "claude-3-opus-20240229"
		}
		return NewClientWithOptions(provider, model, apiKey, opts)
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, "", "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		if model == "" {
			model = "gpt-4-turbo"
		}
		return NewClientWithOptions(provider, model, apiKey, opts)
	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, "", "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
		}
		if model == "" {
			model = "gemini-1.5-pro"
		}
		return NewClientWithOptions(provider, model, apiKey, opts)
	case "copilot-api":
		apiKey := os.Getenv(envCopilotAPIKey)
		if apiKey == "" {
			apiKey = "copilot-api"
		}
		if model == "" {
			model = DefaultCopilotAPIModel
		}
		return NewClientWithOptions(provider, model, apiKey, opts)
	default:
		return nil, "", "", fmt.Errorf("unknown provider: %s", provider)
	}
}

func defaultModelFromEnv() string {
	if envModel := os.Getenv(envDefaultModel); strings.TrimSpace(envModel) != "" {
		return strings.TrimSpace(envModel)
	}
	if envModel := os.Getenv(envLegacyRolloutModel); strings.TrimSpace(envModel) != "" {
		return strings.TrimSpace(envModel)
	}
	return ""
}

func NewClient(provider, model, token string) (Client, string, string, error) {
	return NewClientWithOptions(provider, model, token, Options{})
}

func NewClientWithOptions(provider, model, token string, opts Options) (Client, string, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.TrimSpace(model)
	token = strings.TrimSpace(token)
	if provider == "" {
		return nil, "", "", fmt.Errorf("LLM provider is required")
	}
	if token == "" {
		return nil, "", "", fmt.Errorf("LLM token is required")
	}
	switch provider {
	case "anthropic":
		if model == "" {
			model = "claude-3-opus-20240229"
		}
		return NewAnthropicClientWithOptions(token, model, optionList(opts)...), provider, model, nil
	case "openai":
		if model == "" {
			model = "gpt-4-turbo"
		}
		return NewOpenAIClientWithOptions(token, model, optionList(opts)...), provider, model, nil
	case "gemini":
		if model == "" {
			model = "gemini-1.5-pro"
		}
		return NewGeminiClientWithOptions(token, model, optionList(opts)...), provider, model, nil
	case "copilot-api":
		if model == "" {
			model = DefaultCopilotAPIModel
		}
		return NewCopilotAPIClientWithOptions(token, model, optionList(opts)...), provider, model, nil
	default:
		return nil, "", "", fmt.Errorf("unknown provider: %s", provider)
	}
}

func optionList(opts Options) []ClientOption {
	var out []ClientOption
	if opts.Temperature != nil {
		out = append(out, WithTemperature(*opts.Temperature))
	}
	return out
}

// NewChatClientFromEnv mirrors NewClientFromEnv but returns a ChatClient.
// All current providers implement ChatClient, so the type assertion is
// expected to succeed.
func NewChatClientFromEnv(provider, model string) (ChatClient, string, string, error) {
	return NewChatClientFromEnvWithOptions(provider, model, Options{})
}

func NewChatClientFromEnvWithOptions(provider, model string, opts Options) (ChatClient, string, string, error) {
	client, resolvedProvider, resolvedModel, err := NewClientFromEnvWithOptions(provider, model, opts)
	if err != nil {
		return nil, "", "", err
	}
	chatClient, ok := client.(ChatClient)
	if !ok {
		return nil, "", "", fmt.Errorf("selected provider %s does not support chat", resolvedProvider)
	}
	return chatClient, resolvedProvider, resolvedModel, nil
}

func NewChatClient(provider, model, token string) (ChatClient, string, string, error) {
	return NewChatClientWithOptions(provider, model, token, Options{})
}

func NewChatClientWithOptions(provider, model, token string, opts Options) (ChatClient, string, string, error) {
	client, resolvedProvider, resolvedModel, err := NewClientWithOptions(provider, model, token, opts)
	if err != nil {
		return nil, "", "", err
	}
	chatClient, ok := client.(ChatClient)
	if !ok {
		return nil, "", "", fmt.Errorf("selected provider %s does not support chat", resolvedProvider)
	}
	return chatClient, resolvedProvider, resolvedModel, nil
}
