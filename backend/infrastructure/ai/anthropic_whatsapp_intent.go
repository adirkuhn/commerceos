package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// whatsappIntentToolName is the forced tool Claude must call -- same
// "tool use forcing" structured-output mechanism as
// extractLineItemsToolName.
const whatsappIntentToolName = "classify_whatsapp_intent"

// AnthropicWhatsAppIntentParser implements whatsapp.AIIntentParser via
// Claude's Messages API.
type AnthropicWhatsAppIntentParser struct {
	apiKey  string
	model   anthropic.Model
	baseURL string
}

func NewAnthropicWhatsAppIntentParser(apiKey string) *AnthropicWhatsAppIntentParser {
	return &AnthropicWhatsAppIntentParser{apiKey: apiKey, model: anthropic.ModelClaudeSonnet5}
}

func (p *AnthropicWhatsAppIntentParser) WithBaseURL(url string) *AnthropicWhatsAppIntentParser {
	p.baseURL = url
	return p
}

func (p *AnthropicWhatsAppIntentParser) WithModel(model string) *AnthropicWhatsAppIntentParser {
	if model != "" {
		p.model = anthropic.Model(model)
	}
	return p
}

func (p *AnthropicWhatsAppIntentParser) ParseIntent(ctx context.Context, text string) (whatsappDomain.ParsedIntent, error) {
	opts := []option.RequestOption{option.WithAPIKey(p.apiKey)}
	if p.baseURL != "" {
		opts = append(opts, option.WithBaseURL(p.baseURL))
	}
	client := anthropic.NewClient(opts...)

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       p.model,
		MaxTokens:   whatsappIntentMaxOutputTokens,
		Temperature: anthropic.Float(0),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				whatsappIntentPrompt + "\n\nMessage: " + text + "\n\nCall the " + whatsappIntentToolName + " tool with the result.",
			)),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name: whatsappIntentToolName,
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: whatsappIntentJSONSchema["properties"],
					Required:   []string{"command", "product_query"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(whatsappIntentToolName),
	})
	if err != nil {
		return whatsappDomain.ParsedIntent{}, fmt.Errorf("anthropicWhatsAppIntentParser: %w", err)
	}

	for _, block := range message.Content {
		if block.Type != "tool_use" || block.Name != whatsappIntentToolName {
			continue
		}
		var payload whatsappIntentPayload
		if err := json.Unmarshal(block.Input, &payload); err != nil {
			return whatsappDomain.ParsedIntent{}, fmt.Errorf("anthropicWhatsAppIntentParser: unmarshal intent: %w", err)
		}
		return toParsedIntent(payload), nil
	}
	return whatsappDomain.ParsedIntent{}, fmt.Errorf("anthropicWhatsAppIntentParser: no %s tool_use block found in response", whatsappIntentToolName)
}
