package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// GeminiWhatsAppIntentParser implements whatsapp.AIIntentParser via
// Gemini's generateContent API.
type GeminiWhatsAppIntentParser struct {
	apiKey         string
	model          string
	fallbackModels []string
	baseURL        string
}

func NewGeminiWhatsAppIntentParser(apiKey string) *GeminiWhatsAppIntentParser {
	return &GeminiWhatsAppIntentParser{apiKey: apiKey, model: defaultGeminiModel}
}

func (p *GeminiWhatsAppIntentParser) WithBaseURL(url string) *GeminiWhatsAppIntentParser {
	p.baseURL = url
	return p
}

func (p *GeminiWhatsAppIntentParser) WithModel(model string) *GeminiWhatsAppIntentParser {
	if model != "" {
		p.model = model
	}
	return p
}

// WithFallbackModels sets the models ParseIntent retries, in order, when
// the current one comes back rate-limited (HTTP 429) -- see
// domain/integrations.AIConnection.FallbackModels. commaSeparated may be
// "", meaning no fallback. Blank entries and ones identical to the
// primary model are dropped, so a sloppy "gemini-x,,gemini-x,gemini-y"
// config can't retry the exact model that just got rate-limited or loop
// on an empty model string.
func (p *GeminiWhatsAppIntentParser) WithFallbackModels(commaSeparated string) *GeminiWhatsAppIntentParser {
	p.fallbackModels = nil
	for _, m := range strings.Split(commaSeparated, ",") {
		m = strings.TrimSpace(m)
		if m == "" || m == p.model {
			continue
		}
		p.fallbackModels = append(p.fallbackModels, m)
	}
	return p
}

func (p *GeminiWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	cfg := &genai.ClientConfig{APIKey: p.apiKey, Backend: genai.BackendGeminiAPI}
	if p.baseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: p.baseURL}
	}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("geminiWhatsAppIntentParser: build client: %w", err)
	}

	models := append([]string{p.model}, p.fallbackModels...)
	var lastErr error
	for i, model := range models {
		payload, err := p.generateContent(ctx, client, model, text)
		if err == nil {
			return toParsedIntent(payload), nil
		}
		lastErr = err
		var apiErr genai.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != 429 {
			// Only a rate limit is worth burning the next model on -- any
			// other failure (bad schema, truncated response, auth) will
			// fail identically against every model, so retrying would just
			// add latency for the same outcome.
			return whatsappDomain.ParsedIntent{}, lastErr
		}
		if i < len(models)-1 {
			log.Printf("geminiWhatsAppIntentParser: %s rate-limited, falling back to %s", model, models[i+1])
		}
	}
	return whatsappDomain.ParsedIntent{}, lastErr
}

func (p *GeminiWhatsAppIntentParser) generateContent(ctx context.Context, client *genai.Client, model, text string) (whatsappIntentPayload, error) {
	maxTokens := int32(whatsappIntentMaxOutputTokens)
	temperature := float32(0)
	// Gemini's newer models spend MaxOutputTokens on internal "thinking"
	// before the actual answer by default -- with a cap this small for a
	// one-word classification call, thinking alone can exhaust the whole
	// budget and truncate the response before it ever reaches valid JSON
	// (seen in practice: "unmarshal intent: invalid character ... looking
	// for beginning of value"). ThinkingBudget: 0 disables it -- this call
	// needs no reasoning, only strict classification.
	thinkingBudget := int32(0)
	resp, err := client.Models.GenerateContent(ctx, model,
		[]*genai.Content{genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromText(whatsappIntentPrompt),
			genai.NewPartFromText("Message: " + text),
		}, genai.RoleUser)},
		&genai.GenerateContentConfig{
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: whatsappIntentJSONSchema,
			MaxOutputTokens:    maxTokens,
			Temperature:        &temperature,
			ThinkingConfig:     &genai.ThinkingConfig{ThinkingBudget: &thinkingBudget},
		},
	)
	if err != nil {
		return whatsappIntentPayload{}, fmt.Errorf("geminiWhatsAppIntentParser: %w", err)
	}

	out := resp.Text()
	if out == "" {
		return whatsappIntentPayload{}, fmt.Errorf("geminiWhatsAppIntentParser: no text found in response")
	}

	var payload whatsappIntentPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return whatsappIntentPayload{}, fmt.Errorf("geminiWhatsAppIntentParser: unmarshal intent: %w", err)
	}
	return payload, nil
}
