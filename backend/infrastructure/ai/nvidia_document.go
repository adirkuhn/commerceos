package ai

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// nvidiaDocumentMaxTokens matches the Anthropic adapter's cap -- a long
// supplier invoice can run to dozens of line items.
const nvidiaDocumentMaxTokens = 4096

// NVIDIADocumentParser implements procurement.AIDocumentParser against
// NVIDIA's hosted API. Like Ollama, the open models behind it have no
// native PDF input, so it prompts on the document's extracted text layer.
type NVIDIADocumentParser struct {
	chat chatCompletionsClient
}

func NewNVIDIADocumentParser(apiKey string) *NVIDIADocumentParser {
	return &NVIDIADocumentParser{chat: newNVIDIAChat(apiKey)}
}

// WithBaseURL points the client at a different server (tests, proxies).
func (p *NVIDIADocumentParser) WithBaseURL(url string) *NVIDIADocumentParser {
	p.chat.baseURL = url
	return p
}

// WithModel overrides the default openai/gpt-oss-120b.
func (p *NVIDIADocumentParser) WithModel(model string) *NVIDIADocumentParser {
	p.chat.setModel(model)
	return p
}

// WithFallbackModels sets the models retried on a rate limit -- call after
// WithModel so the primary model is excluded from the list.
func (p *NVIDIADocumentParser) WithFallbackModels(commaSeparated string) *NVIDIADocumentParser {
	p.chat.setFallbackModels(commaSeparated)
	return p
}

func (p *NVIDIADocumentParser) ExtractLineItems(ctx context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	if doc.ExtractedText == "" {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("nvidiaDocumentParser: document has no extracted text -- NVIDIA's hosted models have no native PDF input")
	}
	var payload extractedLineItemsPayload
	prompt := extractLineItemsPrompt + "\n\nDocument text:\n" + doc.ExtractedText
	if err := p.chat.completeJSON(ctx, prompt, lineItemsJSONSchema, nvidiaDocumentMaxTokens, &payload); err != nil {
		return procurementDomain.DocumentExtractionResult{}, fmt.Errorf("nvidiaDocumentParser: %w", err)
	}
	return procurementDomain.DocumentExtractionResult{LineItems: toExtractedLineItems(payload.LineItems), SupplierName: payload.SupplierName}, nil
}
