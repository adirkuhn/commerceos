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
	// defaultNVIDIAModel: strong multi-step tool calling on the free
	// catalog (late 2026); qwen3-next is the fast fallback.
	defaultNVIDIAModel = "openai/gpt-oss-120b"
)

var defaultNVIDIAFallbacks = []string{"qwen/qwen3-next-80b-a3b-instruct"}

// chatCompletionsClient is the shared transport for OpenAI-compatible Chat
// Completions: every NVIDIA adapter and the OpenAI assistant model. NVIDIA's
// API catalog (build.nvidia.com) is OpenAI-compatible only for Chat
// Completions -- not the Responses API the OpenAI document/intent adapters
// use -- so it can't be reached by pointing those at a different base URL.
//
// Structured output goes through NVIDIA's own nvext.guided_json extension
// (constrained decoding on their side); the schema is also spelled out in
// the prompt and the reply is parsed defensively, since not every model in
// the catalog honours guided decoding.
type chatCompletionsClient struct {
	// label names the provider in errors and logs.
	label          string
	apiKey         string
	baseURL        string
	model          string
	fallbackModels []string
	// defaultFallbacks are used when no fallback models are configured.
	defaultFallbacks []string
	// reasoningEffort returns the reasoning_effort to send for a model, ""
	// for none -- only models that understand it get the field.
	reasoningEffort func(model string) string
}

func newNVIDIAChat(apiKey string) chatCompletionsClient {
	return chatCompletionsClient{
		label: "nvidia", apiKey: apiKey, baseURL: defaultNVIDIABaseURL, model: defaultNVIDIAModel,
		fallbackModels: defaultNVIDIAFallbacks, defaultFallbacks: defaultNVIDIAFallbacks,
		reasoningEffort: nvidiaReasoningEffort,
	}
}

// nvidiaReasoningEffort keeps gpt-oss's thinking short, so the answer (or
// tool call) fits the token budget; other models get no field, since some
// reject unknown fields.
func nvidiaReasoningEffort(model string) string {
	if strings.HasPrefix(model, "openai/gpt-oss") {
		return "low"
	}
	return ""
}

// requestOptions are the per-model extra body fields.
func (c chatCompletionsClient) requestOptions(model string) []option.RequestOption {
	if c.reasoningEffort == nil {
		return nil
	}
	if effort := c.reasoningEffort(model); effort != "" {
		return []option.RequestOption{option.WithJSONSet("reasoning_effort", effort)}
	}
	return nil
}

func (c *chatCompletionsClient) setModel(model string) {
	if model != "" {
		c.model = model
	}
}

// setFallbackModels parses a comma-separated list of models retried, in
// order, when the current one is rate-limited (the free tier caps requests
// per minute). Blank entries and the primary model itself are dropped; an
// empty list means the provider's default fallbacks.
func (c *chatCompletionsClient) setFallbackModels(commaSeparated string) {
	if strings.TrimSpace(commaSeparated) == "" {
		c.fallbackModels = c.defaultFallbacks
		return
	}
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
// out.
func (c chatCompletionsClient) completeJSON(ctx context.Context, prompt string, schema map[string]any, maxTokens int64, out any) error {
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("nvidia: marshal schema: %w", err)
	}
	client := c.newClient()
	fullPrompt := prompt + "\n\nReply with JSON only, matching this JSON schema:\n" + string(schemaJSON)

	return c.withFallback(func(model string) error {
		content, err := c.complete(ctx, client, model, fullPrompt, schema, maxTokens)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(stripCodeFence(content)), out); err != nil {
			return fmt.Errorf("nvidia: unmarshal reply: %w", err)
		}
		return nil
	})
}

func (c chatCompletionsClient) newClient() openai.Client {
	return openai.NewClient(
		option.WithAPIKey(c.apiKey),
		option.WithBaseURL(c.baseURL),
		// Retrying is the fallback list's job (and, for the assistant, the
		// request queue's) -- the SDK's own retries would hammer a model
		// that just said it's over its limit.
		option.WithMaxRetries(0),
	)
}

// withFallback runs call with the primary model, then each fallback model
// -- moving on only after HTTP 429, since any other failure (auth, bad
// request, malformed output) would repeat on every model.
func (c chatCompletionsClient) withFallback(call func(model string) error) error {
	models := []string{c.model}
	for _, f := range c.fallbackModels {
		if f != c.model {
			models = append(models, f)
		}
	}
	var lastErr error
	for i, model := range models {
		err := call(model)
		if err == nil {
			return nil
		}
		if len(models) > 1 {
			err = fmt.Errorf("model %s: %w", model, err)
		}
		lastErr = err
		if httpStatus(err) != http.StatusTooManyRequests {
			return lastErr
		}
		if i < len(models)-1 {
			log.Printf("%s: %s rate-limited, falling back to %s", c.label, model, models[i+1])
		}
	}
	return lastErr
}

// httpStatus is the HTTP status of an API error in err's chain, 0 if none.
func httpStatus(err error) int {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func (c chatCompletionsClient) complete(ctx context.Context, client openai.Client, model, prompt string, schema map[string]any, maxTokens int64) (string, error) {
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
		Temperature: openai.Float(0),
		MaxTokens:   openai.Int(maxTokens),
	}, append(c.requestOptions(model), option.WithJSONSet("nvext", map[string]any{"guided_json": schema}))...)
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
