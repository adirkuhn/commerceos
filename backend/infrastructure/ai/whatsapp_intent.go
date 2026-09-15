package ai

import whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"

// whatsappIntentPrompt is the shared instruction text across all four
// provider adapters implementing whatsapp.AIIntentParser -- deliberately
// tiny (this is a per-message classification call, not a chat), and
// explicit that it must never free-generate a reply: the actual reply
// text always comes from application/whatsapp's own deterministic
// formatters, never from the model.
const whatsappIntentPrompt = `You classify one inbound WhatsApp message for a store's automated product-lookup bot. Reply with JSON ONLY, matching the schema exactly -- no explanation, no greeting, no extra text. The message can be written in any language (Portuguese, Dutch, English, or others).

Classify it into exactly one command:
- STOCK: asking whether a product is in stock / available / has quantity (e.g. "tem estoque de X?", "voce tem X?", "is X available?")
- PRICE: asking a product's price/cost (e.g. "quanto custa X?", "quanto ta o X?", "what's the price of X?")
- EXPIRY: asking a product's expiry/validity date
- INFO: asking for stock, price and expiry together, or a general "tell me about X"
- UNKNOWN: anything else -- greetings, small talk, an unrelated question, or no product mentioned at all

Also extract product_query: the product the customer named, written exactly as they wrote it (their own wording -- a product code or a product name/description, verbatim, do not translate or correct it). Empty string when command is UNKNOWN.`

// whatsappIntentJSONSchema is the strict, closed-schema structured-output
// shape every adapter forces -- same "the model can only classify+extract,
// never freely generate" discipline as lineItemsJSONSchema.
var whatsappIntentJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"command": map[string]any{
			"type": "string",
			"enum": []string{"STOCK", "PRICE", "EXPIRY", "INFO", "UNKNOWN"},
		},
		"product_query": map[string]any{"type": "string"},
	},
	"required":             []string{"command", "product_query"},
	"additionalProperties": false,
}

// whatsappIntentMaxOutputTokens caps every provider's response -- this is a
// one-word-command-plus-a-short-phrase extraction, never a long-form
// answer, so a small cap both bounds cost and gives a malformed/runaway
// completion nowhere to go.
const whatsappIntentMaxOutputTokens = 100

type whatsappIntentPayload struct {
	Command      string `json:"command"`
	ProductQuery string `json:"product_query"`
}

// validIntentCommands guards against a model returning something outside
// the schema's enum despite structured/forced output (seen in practice
// with local Ollama models, which don't enforce the JSON schema
// server-side, see validLineTypes's doc comment for the same issue on the
// document-parsing adapters) -- falls back to Unknown rather than
// persisting/acting on a value the rest of the app doesn't recognize.
var validIntentCommands = map[string]whatsappDomain.IntentCommand{
	string(whatsappDomain.IntentCommandStock):   whatsappDomain.IntentCommandStock,
	string(whatsappDomain.IntentCommandPrice):   whatsappDomain.IntentCommandPrice,
	string(whatsappDomain.IntentCommandExpiry):  whatsappDomain.IntentCommandExpiry,
	string(whatsappDomain.IntentCommandInfo):    whatsappDomain.IntentCommandInfo,
	string(whatsappDomain.IntentCommandUnknown): whatsappDomain.IntentCommandUnknown,
}

// toParsedIntent converts the shared payload shape into the domain type --
// used by all four provider adapters so the command-validation fallback
// lives in exactly one place. ProductQuery is always cleared for an
// Unknown command, regardless of what the model returned for it, so a
// caller never accidentally resolves a SKU for a message that wasn't
// actually about one.
func toParsedIntent(payload whatsappIntentPayload) whatsappDomain.ParsedIntent {
	cmd, ok := validIntentCommands[payload.Command]
	if !ok {
		cmd = whatsappDomain.IntentCommandUnknown
	}
	query := payload.ProductQuery
	if cmd == whatsappDomain.IntentCommandUnknown {
		query = ""
	}
	return whatsappDomain.ParsedIntent{Command: cmd, ProductQuery: query}
}
