// Package ai holds per-vendor adapters implementing the
// procurement.AIDocumentParser port. OpenAIDocumentParser is the first
// adapter, on the official github.com/openai/openai-go SDK; Anthropic/
// Gemini/Ollama are added as siblings without touching application code,
// since callers only ever depend on the AIDocumentParser interface.
package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIDocumentParser calls OpenAI's Responses API with GPT-4o, which
// accepts PDF input natively via an input_file content part -- no separate
// OCR/text-extraction step needed for this provider (PDFTextExtractor is
// only consulted upstream to skip the AI call entirely when a deterministic
// text layer already extracted cleanly).
type OpenAIDocumentParser struct {
	apiKey  string
	model   string
	baseURL string
}

func NewOpenAIDocumentParser(apiKey string) *OpenAIDocumentParser {
	return &OpenAIDocumentParser{apiKey: apiKey, model: "gpt-4o", baseURL: defaultOpenAIBaseURL}
}

// WithBaseURL points the client at a different server -- used by tests to
// target an httptest.Server instead of the real OpenAI API.
func (p *OpenAIDocumentParser) WithBaseURL(url string) *OpenAIDocumentParser {
	p.baseURL = url
	return p
}

// WithModel overrides the default "gpt-4o" -- used when a tenant's
// AIConnection specifies a different model.
func (p *OpenAIDocumentParser) WithModel(model string) *OpenAIDocumentParser {
	if model != "" {
		p.model = model
	}
	return p
}

var lineItemsJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"line_items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"raw_sku":               map[string]any{"type": "string"},
					"raw_description":       map[string]any{"type": "string"},
					"quantity":              map[string]any{"type": "integer"},
					"unit_cost_minor_units": map[string]any{"type": "integer"},
				},
				"required":             []string{"raw_sku", "raw_description", "quantity", "unit_cost_minor_units"},
				"additionalProperties": false,
			},
		},
		// supplier_name is the document's issuing company -- the letterhead
		// name, not the addressee -- empty string when the model can't find
		// a clear one, never fabricated.
		"supplier_name": map[string]any{"type": "string"},
	},
	"required":             []string{"line_items", "supplier_name"},
	"additionalProperties": false,
}

type extractedLineItemsPayload struct {
	LineItems    []extractedLineItemPayload `json:"line_items"`
	SupplierName string                     `json:"supplier_name"`
}

type extractedLineItemPayload struct {
	RawSKU             string `json:"raw_sku"`
	RawDescription     string `json:"raw_description"`
	Quantity           int64  `json:"quantity"`
	UnitCostMinorUnits int64  `json:"unit_cost_minor_units"`
}

func (p *OpenAIDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	client := openai.NewClient(option.WithAPIKey(p.apiKey), option.WithBaseURL(p.baseURL))

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: p.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(responses.ResponseInputMessageContentListParam{
					{OfInputText: &responses.ResponseInputTextParam{
						Text: "Extract every line item from this purchase order document. Return the SKU/barcode if printed, the description, the quantity, and the unit cost in minor currency units (cents). Also return the name of the supplier company that issued this document (the letterhead/sender, not the addressee/buyer) -- empty string if there's no clear one.",
					}},
					{OfInputFile: &responses.ResponseInputFileParam{
						Filename: openai.String(doc.FileName),
						FileData: openai.String(fmt.Sprintf("data:%s;base64,%s", doc.ContentType, base64.StdEncoding.EncodeToString(doc.Data))),
					}},
				}, "user"),
			},
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name: "extracted_line_items", Schema: lineItemsJSONSchema, Strict: openai.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: %w", err)
	}

	text := resp.OutputText()
	if text == "" {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: no output_text found in response")
	}

	var payload extractedLineItemsPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: unmarshal line items: %w", err)
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
