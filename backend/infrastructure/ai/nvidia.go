package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	defaultNVIDIABaseURL = "https://integrate.api.nvidia.com/v1"
	defaultNVIDIAModel   = "meta/llama-3.3-70b-instruct"
)

// nvidiaChat is the shared transport behind every NVIDIA adapter. NVIDIA's
// API catalog (build.nvidia.com) is OpenAI-compatible only for Chat
// Completions -- not the Responses API the OpenAI adapters use -- so it
// can't be reached by pointing an OpenAI adapter at a different base URL.
//
// Structured output goes through NVIDIA's own nvext.guided_json extension
// (constrained decoding on their side); the schema is also spelled out in
// the prompt and the reply is parsed defensively, since not every model in
// the catalog honours guided decoding.
type nvidiaChat struct {
	apiKey         string
	baseURL        string
	model          string
	fallbackModels []string
}

func newNVIDIAChat(apiKey string) nvidiaChat {
	return nvidiaChat{apiKey: apiKey, baseURL: defaultNVIDIABaseURL, model: defaultNVIDIAModel}
}

func (c *nvidiaChat) setModel(model string) {
	if model != "" {
		c.model = model
	}
}

// setFallbackModels parses a comma-separated list of models retried, in
// order, when the current one is rate-limited (the free tier caps requests
// per minute). Blank entries and the primary model itself are dropped.
func (c *nvidiaChat) setFallbackModels(commaSeparated string) {
	c.fallbackModels = nil
	for _, m := range strings.Split(commaSeparated, ",") {
		m = strings.TrimSpace(m)
		if m == "" || m == c.model {
			continue
		}
		c.fallbackModels = append(c.fallbackModels, m)
	}
}

// completeJSON sends prompt, forces schema, and unmarshals the reply into
// out -- falling back to the next model only on HTTP 429, since any other
// failure (auth, bad request, malformed output) would repeat on every model.
func (c nvidiaChat) completeJSON(ctx context.Context, prompt string, schema map[string]any, maxTokens int64, out any) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("nvidia: marshal schema: %w", err)
	}
	client := openai.NewClient(
		option.WithAPIKey(c.apiKey),
		option.WithBaseURL(c.baseURL),
		// Retrying is the fallback list's job (and, for the assistant, the
		// request queue's) -- the SDK's own retries would hammer a model
		// that just said it's over its limit.
		option.WithMaxRetries(0),
	)
	fullPrompt := prompt + "\n\nReply with JSON only, matching this JSON schema:\n" + string(schemaJSON)

	models := append([]string{c.model}, c.fallbackModels...)
	var lastErr error
	for i, model := range models {
		content, err := c.complete(ctx, client, model, fullPrompt, schema, maxTokens)
		if err == nil {
			if err := json.Unmarshal([]byte(stripCodeFence(content)), out); err != nil {
				return fmt.Errorf("nvidia: unmarshal reply: %w", err)
			}
			return nil
		}
		lastErr = err
		var apiErr *openai.Error
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
			return lastErr
		}
		if i < len(models)-1 {
			log.Printf("nvidia: %s rate-limited, falling back to %s", model, models[i+1])
		}
	}
	return lastErr
}

func (c nvidiaChat) complete(ctx context.Context, client openai.Client, model, prompt string, schema map[string]any, maxTokens int64) (string, error) {
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
		Temperature: openai.Float(0),
		MaxTokens:   openai.Int(maxTokens),
	}, option.WithJSONSet("nvext", map[string]any{"guided_json": schema}))
	if err != nil {
		return "", fmt.Errorf("nvidia: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("nvidia: empty response")
	}
	return resp.Choices[0].Message.Content, nil
}

// stripCodeFence unwraps a ```json ... ``` block -- models without guided
// decoding commonly wrap JSON in one despite being told not to.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.Index(s, "\n"); nl >= 0 {
		s = s[nl+1:]
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
}
