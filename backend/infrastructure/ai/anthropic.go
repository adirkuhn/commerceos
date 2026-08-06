package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// extractLineItemsToolName is the forced tool Claude must call -- Anthropic
// has no OpenAI-style json_schema response format, so structured extraction
// goes through "tool use forcing": define one tool shaped like the desired
// output, force tool_choice to it, and read the result back from the
// resulting tool_use block's Input instead of a text response.
const extractLineItemsToolName = "extract_line_items"

// AnthropicDocumentParser calls Claude's Messages API, which accepts PDF
// input natively via a document content block (Base64PDFSourceParam) --
// same "no separate OCR step" shape as OpenAIDocumentParser.
type AnthropicDocumentParser struct {
	apiKey  string
	model   anthropic.Model
	baseURL string
}

func NewAnthropicDocumentParser(apiKey string) *AnthropicDocumentParser {
	return &AnthropicDocumentParser{apiKey: apiKey, model: anthropic.ModelClaudeSonnet5}
}

// WithBaseURL points the client at a different server -- used by tests to
// target an httptest.Server instead of the real Anthropic API.
func (p *AnthropicDocumentParser) WithBaseURL(url string) *AnthropicDocumentParser {
	p.baseURL = url
	return p
}

// WithModel overrides the default model -- used when a tenant's
// AIConnection specifies a different one.
func (p *AnthropicDocumentParser) WithModel(model string) *AnthropicDocumentParser {
	if model != "" {
		p.model = anthropic.Model(model)
	}
	return p
}

func (p *AnthropicDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	opts := []option.RequestOption{option.WithAPIKey(p.apiKey)}
	if p.baseURL != "" {
		opts = append(opts, option.WithBaseURL(p.baseURL))
	}
	client := anthropic.NewClient(opts...)

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock("Extract every line item from this purchase order document. Return the SKU/barcode if printed, the description, the quantity, and the unit cost in minor currency units (cents). Also return the name of the supplier company that issued this document (the letterhead/sender, not the addressee/buyer) -- empty string if there's no clear one. Call the extract_line_items tool with the result."),
				anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{Data: base64.StdEncoding.EncodeToString(doc.Data)}),
			),
		},
		Tools: []anthropic.ToolUnionParam{{
			OfTool: &anthropic.ToolParam{
				Name: extractLineItemsToolName,
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: lineItemsJSONSchema["properties"],
					Required:   []string{"line_items", "supplier_name"},
				},
			},
		}},
		ToolChoice: anthropic.ToolChoiceParamOfTool(extractLineItemsToolName),
	})
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("anthropicDocumentParser: %w", err)
	}

	for _, block := range message.Content {
		if block.Type != "tool_use" || block.Name != extractLineItemsToolName {
			continue
		}
		var payload extractedLineItemsPayload
		if err := json.Unmarshal(block.Input, &payload); err != nil {
			return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("anthropicDocumentParser: unmarshal line items: %w", err)
		}
		items := make([]procurementDomain.ExtractedLineItem, 0, len(payload.LineItems))
		for _, li := range payload.LineItems {
			items = append(items, procurementDomain.ExtractedLineItem{
				RawSKU:             li.RawSKU,
				RawDescription:     li.RawDescription,
				Quantity:           li.Quantity,
				UnitCostMinorUnits: li.UnitCostMinorUnits,
			})
		}
		return procurementDomain.DocumentExtractionResult{LineItems: items, SupplierName: payload.SupplierName}, nil
	}
	return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("anthropicDocumentParser: no %s tool_use block found in response", extractLineItemsToolName)
}
