// Package ai holds per-vendor adapters implementing the
// procurement.AIDocumentParser port via plain net/http REST calls -- no
// vendor SDKs, per the plan's "keeps the abstraction thin" decision.
// OpenAIDocumentParser is the first adapter; Claude/DeepSeek/etc can be
// added later as siblings without touching application code, since callers
// only ever depend on the AIDocumentParser interface.
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

// OpenAIDocumentParser calls OpenAI's Responses API with GPT-4o, which
// accepts PDF input natively via an input_file content part -- no separate
// OCR/text-extraction step needed for this provider (PDFTextExtractor is
// only consulted upstream to skip the AI call entirely when a deterministic
// text layer already extracted cleanly).
type OpenAIDocumentParser struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewOpenAIDocumentParser(apiKey string) *OpenAIDocumentParser {
	return &OpenAIDocumentParser{
		apiKey:     apiKey,
		model:      "gpt-4o",
		baseURL:    defaultOpenAIBaseURL,
		httpClient: http.DefaultClient,
	}
}

// WithBaseURL points the client at a different server -- used by tests to
// target an httptest.Server instead of the real OpenAI API.
func (p *OpenAIDocumentParser) WithBaseURL(url string) *OpenAIDocumentParser {
	p.baseURL = url
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

type openAIResponsesRequest struct {
	Model string                   `json:"model"`
	Input []openAIResponsesMessage `json:"input"`
	Text  openAIResponsesTextOpts  `json:"text"`
}

type openAIResponsesMessage struct {
	Role    string                   `json:"role"`
	Content []openAIResponsesContent `json:"content"`
}

type openAIResponsesContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	FileName string `json:"filename,omitempty"`
	FileData string `json:"file_data,omitempty"`
}

type openAIResponsesTextOpts struct {
	Format openAIResponsesFormat `json:"format"`
}

type openAIResponsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type openAIResponsesResponse struct {
	Output []openAIResponsesOutputItem `json:"output"`
	Error  *openAIError                `json:"error"`
}

type openAIResponsesOutputItem struct {
	Type    string                   `json:"type"`
	Content []openAIResponsesContent `json:"content"`
}

type openAIError struct {
	Message string `json:"message"`
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
	reqBody := openAIResponsesRequest{
		Model: p.model,
		Input: []openAIResponsesMessage{{
			Role: "user",
			Content: []openAIResponsesContent{
				{Type: "input_text", Text: "Extract every line item from this purchase order document. Return the SKU/barcode if printed, the description, the quantity, and the unit cost in minor currency units (cents). Also return the name of the supplier company that issued this document (the letterhead/sender, not the addressee/buyer) -- empty string if there's no clear one."},
				{
					Type:     "input_file",
					FileName: doc.FileName,
					FileData: fmt.Sprintf("data:%s;base64,%s", doc.ContentType, base64.StdEncoding.EncodeToString(doc.Data)),
				},
			},
		}},
		Text: openAIResponsesTextOpts{
			Format: openAIResponsesFormat{
				Type:   "json_schema",
				Name:   "extracted_line_items",
				Schema: lineItemsJSONSchema,
				Strict: true,
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: read response: %w", err)
	}

	var parsed openAIResponsesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: API error (%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("openAIDocumentParser: API error (%d): %s", resp.StatusCode, string(respBody))
	}

	text, err := extractOutputText(parsed)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, err
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

func extractOutputText(resp openAIResponsesResponse) (string, error) {
	for _, item := range resp.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				return c.Text, nil
			}
		}
	}
	return "", fmt.Errorf("openAIDocumentParser: no output_text found in response")
}
