package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// OllamaWhatsAppIntentParser implements whatsapp.AIIntentParser via a
// self-hosted Ollama server. BaseURL is required -- there's no hosted
// default the way the other three providers have.
type OllamaWhatsAppIntentParser struct {
	baseURL string
	model   string
}

func NewOllamaWhatsAppIntentParser(baseURL string) *OllamaWhatsAppIntentParser {
	return &OllamaWhatsAppIntentParser{baseURL: baseURL, model: defaultOllamaModel}
}

func (p *OllamaWhatsAppIntentParser) WithModel(model string) *OllamaWhatsAppIntentParser {
	if model != "" {
		p.model = model
	}
	return p
}

func (p *OllamaWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	base, err := url.Parse(p.baseURL)
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("ollamaWhatsAppIntentParser: invalid base URL: %w", err)
	}
	client := api.NewClient(base, http.DefaultClient)

	schema, err := json.Marshal(whatsappIntentJSONSchema)
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("ollamaWhatsAppIntentParser: marshal schema: %w", err)
	}

	stream := false
	numPredict := whatsappIntentMaxOutputTokens
	temperature := float32(0)
	prompt := whatsappIntentPrompt + "\n\nMessage: " + text

	var payload whatsappIntentPayload
	var chatErr error
	err = client.Chat(ctx, &api.ChatRequest{
		Model:    p.model,
		Messages: []api.Message{{Role: "user", Content: prompt}},
		Format:   json.RawMessage(schema),
		Stream:   &stream,
		Options:  map[string]any{"num_predict": numPredict, "temperature": temperature},
	}, func(resp api.ChatResponse) error {
		chatErr = json.Unmarshal([]byte(resp.Message.Content), &payload)
		return nil
	})
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("ollamaWhatsAppIntentParser: %w", err)
	}
	if chatErr != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("ollamaWhatsAppIntentParser: unmarshal intent: %w", chatErr)
	}
	return toParsedIntent(payload), nil
}
