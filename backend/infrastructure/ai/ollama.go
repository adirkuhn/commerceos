package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

const defaultOllamaModel = "llama3.2"

// OllamaDocumentParser talks to a self-hosted Ollama server (BaseURL is
// required -- there's no hosted default the way the other three providers
// have). Unlike OpenAI/Anthropic/Gemini, local models have no reliable
// native PDF ingestion, so this adapter prompts on doc.ExtractedText (the
// PDF's already-extracted text layer) instead of doc.Data.
type OllamaDocumentParser struct {
	baseURL string
	model   string
}

// NewOllamaDocumentParser requires baseURL -- the tenant's self-hosted
// Ollama instance, e.g. http://localhost:11434.
func NewOllamaDocumentParser(baseURL string) *OllamaDocumentParser {
	return &OllamaDocumentParser{baseURL: baseURL, model: defaultOllamaModel}
}

// WithModel overrides the default model -- used when a tenant's
// AIConnection specifies a different one.
func (p *OllamaDocumentParser) WithModel(model string) *OllamaDocumentParser {
	if model != "" {
		p.model = model
	}
	return p
}

func (p *OllamaDocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	if doc.ExtractedText == "" {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("ollamaDocumentParser: document has no extracted text -- local models have no reliable native PDF support")
	}

	base, err := url.Parse(p.baseURL)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("ollamaDocumentParser: invalid base URL: %w", err)
	}
	client := api.NewClient(base, http.DefaultClient)

	schema, err := json.Marshal(lineItemsJSONSchema)
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("ollamaDocumentParser: marshal schema: %w", err)
	}

	stream := false
	prompt := extractLineItemsPrompt + "\n\nDocument text:\n" + doc.ExtractedText

	var payload extractedLineItemsPayload
	var chatErr error
	err = client.Chat(ctx, &api.ChatRequest{
		Model:    p.model,
		Messages: []api.Message{{Role: "user", Content: prompt}},
		Format:   json.RawMessage(schema),
		Stream:   &stream,
	}, func(resp api.ChatResponse) error {
		chatErr = json.Unmarshal([]byte(resp.Message.Content), &payload)
		return nil
	})
	if err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("ollamaDocumentParser: %w", err)
	}
	if chatErr != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("ollamaDocumentParser: unmarshal line items: %w", chatErr)
	}

	items := toExtractedLineItems(payload.LineItems)
	return procurementDomain.DocumentExtractionResult{LineItems: items, SupplierName: payload.SupplierName}, nil
}
