package whatsapp

import "context"

// IntentCommand is the fixed, closed set an AIIntentParser may classify a
// message into -- deliberately only the read-only lookup commands
// (STOCK/PRICE/EXPIRY/INFO already handled by
// application/whatsapp.HandleInboundMessageUseCase.buildReply's rigid
// "COMMAND CODE" parser). ORDER/CART/CHECKOUT/CANCEL are stateful
// multi-turn flows and are never AI-classified -- too risky to hand a
// customer's free text for those in v1.
type IntentCommand string

const (
	IntentCommandStock   IntentCommand = "STOCK"
	IntentCommandPrice   IntentCommand = "PRICE"
	IntentCommandExpiry  IntentCommand = "EXPIRY"
	IntentCommandInfo    IntentCommand = "INFO"
	IntentCommandUnknown IntentCommand = "UNKNOWN"
)

// ParsedIntent is what an AIIntentParser extracts from one free-text
// WhatsApp message. ProductQuery is the product the customer named,
// verbatim (a SKU code or a product name/description, whichever they
// typed) -- empty when Command is IntentCommandUnknown. The parser never
// composes reply text itself; ProductQuery only ever feeds
// application/whatsapp.ResolveSKUFromQueryUseCase, and the actual reply is
// always the same deterministic formatter the rigid command path already
// uses.
type ParsedIntent struct {
	Command      IntentCommand
	ProductQuery string
}

// AIIntentParser is a provider-agnostic port for classifying one WhatsApp
// message that didn't match the rigid "COMMAND CODE" format -- implemented
// per-vendor in infrastructure/ai/ (same OpenAI/Anthropic/Gemini/Ollama
// adapters procurement.AIDocumentParser already has). Deliberately a
// single, narrow, structured-output call: the model classifies and
// extracts, it never free-generates a reply, and it must work regardless
// of the message's language (Portuguese/Dutch/English/etc) -- see each
// adapter's prompt.
type AIIntentParser interface {
	ParseIntent(ctx context.Context, text string) (ParsedIntent, error)
}

// AIIntentParserResolver resolves whichever AIIntentParser the tenant
// active in ctx has configured (see integrations.AIConnection) -- same
// contract as procurement.AIParserResolver: returns nil, nil (not an
// error) when the tenant hasn't configured an AI connection.
type AIIntentParserResolver interface {
	Resolve(ctx context.Context) (AIIntentParser, error)
}
