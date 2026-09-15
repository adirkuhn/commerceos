package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// OpenAIWhatsAppIntentParser implements whatsapp.AIIntentParser via
// OpenAI's Responses API -- same structured-JSON-output mechanism as
// OpenAIDocumentParser, a plain text prompt instead of a file input.
type OpenAIWhatsAppIntentParser struct {
	apiKey  string
	model   string
	baseURL string
}

func NewOpenAIWhatsAppIntentParser(apiKey string) *OpenAIWhatsAppIntentParser {
	return &OpenAIWhatsAppIntentParser{apiKey: apiKey, model: "gpt-4o", baseURL: defaultOpenAIBaseURL}
}

func (p *OpenAIWhatsAppIntentParser) WithBaseURL(url string) *OpenAIWhatsAppIntentParser {
	p.baseURL = url
	return p
}

func (p *OpenAIWhatsAppIntentParser) WithModel(model string) *OpenAIWhatsAppIntentParser {
	if model != "" {
		p.model = model
	}
	return p
}

func (p *OpenAIWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	client := openai.NewClient(option.WithAPIKey(p.apiKey), option.WithBaseURL(p.baseURL))

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model:           p.model,
		MaxOutputTokens: openai.Int(whatsappIntentMaxOutputTokens),
		Temperature:     openai.Float(0),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(responses.ResponseInputMessageContentListParam{
					{OfInputText: &responses.ResponseInputTextParam{Text: whatsappIntentPrompt}},
					{OfInputText: &responses.ResponseInputTextParam{Text: "Message: " + text}},
				}, "user"),
			},
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name: "whatsapp_intent", Schema: whatsappIntentJSONSchema, Strict: openai.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("openAIWhatsAppIntentParser: %w", err)
	}

	out := resp.OutputText()
	if out == "" {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("openAIWhatsAppIntentParser: no output_text found in response")
	}

	var payload whatsappIntentPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("openAIWhatsAppIntentParser: unmarshal intent: %w", err)
	}
	return toParsedIntent(payload), nil
}
