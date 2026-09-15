package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// GeminiWhatsAppIntentParser implements whatsapp.AIIntentParser via
// Gemini's generateContent API.
type GeminiWhatsAppIntentParser struct {
	apiKey  string
	model   string
	baseURL string
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

func (p *GeminiWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	cfg := &genai.ClientConfig{APIKey: p.apiKey, Backend: genai.BackendGeminiAPI}
	if p.baseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: p.baseURL}
	}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("geminiWhatsAppIntentParser: build client: %w", err)
	}

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
	resp, err := client.Models.GenerateContent(ctx, p.model,
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
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("geminiWhatsAppIntentParser: %w", err)
	}

	out := resp.Text()
	if out == "" {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("geminiWhatsAppIntentParser: no text found in response")
	}

	var payload whatsappIntentPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("geminiWhatsAppIntentParser: unmarshal intent: %w", err)
	}
	return toParsedIntent(payload), nil
}
