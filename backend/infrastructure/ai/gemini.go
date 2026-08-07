package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

const defaultGeminiModel = "gemini-3.6-flash"

// GeminiDocumentParser calls Gemini's generateContent API, which accepts
// PDF input natively as an inline_data part -- same "no separate OCR step"
// shape as the OpenAI/Anthropic adapters. Structured output uses
// ResponseJsonSchema, which (unlike Anthropic) accepts the same plain JSON
// Schema map used for OpenAI's json_schema format directly.
type GeminiDocumentParser struct {
	apiKey  string
	model   string
	baseURL string
}

func NewGeminiDocumentParser(apiKey string) *GeminiDocumentParser {
	return &GeminiDocumentParser{apiKey: apiKey, model: defaultGeminiModel}
}

// WithBaseURL points the client at a different server -- used by tests to
// target an httptest.Server instead of the real Gemini API.
func (p *GeminiDocumentParser) WithBaseURL(url string) *GeminiDocumentParser {
	p.baseURL = url
	return p
}

// WithModel overrides the default model -- used when a tenant's
// AIConnection specifies a different one.
func (p *GeminiDocumentParser) WithModel(model string) *GeminiDocumentParser {
	if model != "" {
		p.model = model
	}
	return p
}

func (p *GeminiDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	cfg := &genai.ClientConfig{APIKey: p.apiKey, Backend: genai.BackendGeminiAPI}
	if p.baseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: p.baseURL}
	}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("geminiDocumentParser: build client: %w", err)
	}

	resp, err := client.Models.GenerateContent(ctx, p.model,
		[]*genai.Content{genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromText(extractLineItemsPrompt),
			genai.NewPartFromBytes(doc.Data, doc.ContentType),
		}, genai.RoleUser)},
		&genai.GenerateContentConfig{
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: lineItemsJSONSchema,
		},
	)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("geminiDocumentParser: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("geminiDocumentParser: no text found in response")
	}

	var payload extractedLineItemsPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("geminiDocumentParser: unmarshal line items: %w", err)
	}

	items := toExtractedLineItems(payload.LineItems)
	return procurementDomain.DocumentExtractionResult{LineItems: items, SupplierName: payload.SupplierName}, nil
}
