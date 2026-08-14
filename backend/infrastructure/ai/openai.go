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
	"time"

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

// extractLineItemsPrompt is the shared instruction text across all four
// provider adapters -- Anthropic/Ollama append their own tool-call/
// document-text suffix, see their respective ExtractLineItems.
const extractLineItemsPrompt = "Extract every line item from this purchase order document. Return the SKU/barcode if printed, the description, the quantity, and the unit cost in minor currency units (cents). Also return the name of the supplier company that issued this document (the letterhead/sender, not the addressee/buyer) -- empty string if there's no clear one. " +
	"For each line item, also classify it with line_type: \"product\" for an actual item being purchased, or one of \"shipping\", \"discount\", \"credit_note\", \"fee\", \"other_charge\" for a non-product row like a delivery fee, a discount/rebate line, a credit note, or any other charge/adjustment that isn't an item being bought -- those must never be classified as \"product\". " +
	"Some documents (commonly supplier invoices) also print a lot/batch code and an expiry date next to each line item, often as the first two columns before the product code -- e.g. \"PT26004 01-mar-2027 100131 FAROFA PREMIUM...\" has batch code \"PT26004\" and expiry date 1 March 2027. When present, return batch_code as printed and expiry_date normalized to ISO 8601 (YYYY-MM-DD), converting any local-language month abbreviation (e.g. Portuguese jan/fev/mar/abr/mai/jun/jul/ago/set/out/nov/dez) to the correct month. Leave both empty when the document doesn't print them -- never guess or infer a batch code or expiry date that isn't explicitly printed."

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
					// line_type: "product" for an actual item being
					// purchased; anything else is a charge that must never
					// be sent through product matching.
					"line_type": map[string]any{
						"type": "string",
						"enum": []string{"product", "shipping", "discount", "credit_note", "fee", "other_charge"},
					},
					// batch_code/expiry_date are optional -- most documents
					// don't print them (see extractLineItemsPrompt).
					"batch_code":  map[string]any{"type": "string"},
					"expiry_date": map[string]any{"type": "string"},
				},
				"required":             []string{"raw_sku", "raw_description", "quantity", "unit_cost_minor_units", "line_type", "batch_code", "expiry_date"},
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
	LineType           string `json:"line_type"`
	BatchCode          string `json:"batch_code"`
	// ExpiryDate is the model's own ISO-8601 (YYYY-MM-DD) normalization of
	// whatever date format the document printed (see extractLineItemsPrompt)
	// -- kept as a raw string here so a malformed value can degrade to nil
	// in toExtractedLineItems instead of failing the whole extraction.
	ExpiryDate string `json:"expiry_date"`
}

// validLineTypes guards against a model returning something outside the
// schema's enum despite Strict/tool-forced output (seen in practice with
// local Ollama models, which don't enforce the JSON schema server-side) --
// falls back to "" (product, see LineItemType.IsProduct) rather than
// persisting a value the rest of the app doesn't recognize.
var validLineTypes = map[string]procurementDomain.LineItemType{
	string(procurementDomain.LineItemTypeProduct):     procurementDomain.LineItemTypeProduct,
	string(procurementDomain.LineItemTypeShipping):    procurementDomain.LineItemTypeShipping,
	string(procurementDomain.LineItemTypeDiscount):    procurementDomain.LineItemTypeDiscount,
	string(procurementDomain.LineItemTypeCreditNote):  procurementDomain.LineItemTypeCreditNote,
	string(procurementDomain.LineItemTypeFee):         procurementDomain.LineItemTypeFee,
	string(procurementDomain.LineItemTypeOtherCharge): procurementDomain.LineItemTypeOtherCharge,
}

// toExtractedLineItems converts the shared payload shape into domain
// objects -- used by all four provider adapters so the line_type
// validation/fallback lives in exactly one place.
func toExtractedLineItems(payload []extractedLineItemPayload) []procurementDomain.ExtractedLineItem {
	items := make([]procurementDomain.ExtractedLineItem, 0, len(payload))
	for _, li := range payload {
		items = append(items, procurementDomain.ExtractedLineItem{
			RawSKU:             li.RawSKU,
			RawDescription:     li.RawDescription,
			Quantity:           li.Quantity,
			UnitCostMinorUnits: li.UnitCostMinorUnits,
			LineType:           validLineTypes[li.LineType],
			BatchCode:          li.BatchCode,
			ExpiryDate:         parseISODate(li.ExpiryDate),
		})
	}
	return items
}

// parseISODate parses the model's YYYY-MM-DD normalization of a document's
// expiry date. A model can still return a malformed or non-ISO value despite
// the prompt's instruction (seen in practice with line_type, see
// validLineTypes) -- degrades to nil rather than failing the whole
// extraction, since an unconfirmed suggestion is safe to drop but not safe
// to guess at.
func parseISODate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func (p *OpenAIDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	client := openai.NewClient(option.WithAPIKey(p.apiKey), option.WithBaseURL(p.baseURL))

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: p.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(responses.ResponseInputMessageContentListParam{
					{OfInputText: &responses.ResponseInputTextParam{
						Text: extractLineItemsPrompt,
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

	items := toExtractedLineItems(payload.LineItems)
	return procurementDomain.DocumentExtractionResult{LineItems: items, SupplierName: payload.SupplierName}, nil
}
