package ai

import (
	"context"
	"fmt"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// NVIDIAWhatsAppIntentParser implements whatsapp.AIIntentParser against
// NVIDIA's hosted API -- same prompt/schema/validation as the other four
// adapters, over the shared chatCompletionsClient transport.
type NVIDIAWhatsAppIntentParser struct {
	chat chatCompletionsClient
}

func NewNVIDIAWhatsAppIntentParser(apiKey string) *NVIDIAWhatsAppIntentParser {
	return &NVIDIAWhatsAppIntentParser{chat: newNVIDIAChat(apiKey)}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (p *NVIDIAWhatsAppIntentParser) WithBaseURL(url string) *NVIDIAWhatsAppIntentParser {
	p.chat.baseURL = url
	return p
}

// WithModel overrides the default meta/llama-3.3-70b-instruct.
func (p *NVIDIAWhatsAppIntentParser) WithModel(model string) *NVIDIAWhatsAppIntentParser {
	p.chat.setModel(model)
	return p
}

// WithFallbackModels sets the models retried on a rate limit -- call after
// WithModel so the primary model is excluded from the list.
func (p *NVIDIAWhatsAppIntentParser) WithFallbackModels(commaSeparated string) *NVIDIAWhatsAppIntentParser {
	p.chat.setFallbackModels(commaSeparated)
	return p
}

func (p *NVIDIAWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	var payload whatsappIntentPayload
	prompt := whatsappIntentPrompt + "\n\nMessage: " + text
	if err := p.chat.completeJSON(ctx, prompt, whatsappIntentJSONSchema, whatsappIntentMaxOutputTokens, &payload); err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("nvidiaWhatsAppIntentParser: %w", err)
	}
	return toParsedIntent(payload), nil
}
