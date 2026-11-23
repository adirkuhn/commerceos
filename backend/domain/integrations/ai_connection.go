package integrations

import "time"

// AIProvider is deliberately its own type from Provider (WooCommerce et al)
// -- an AIConnection has nothing in common with an e-commerce sync
// Connection (no store URL, no OAuth callback, a different credential
// shape), so it's a separate entity/table rather than overloading Connection
// with AI-only fields.
type AIProvider string

const (
	AIProviderOpenAI    AIProvider = "openai"
	AIProviderAnthropic AIProvider = "anthropic"
	AIProviderGemini    AIProvider = "gemini"
	AIProviderOllama    AIProvider = "ollama"
	// AIProviderNVIDIA is NVIDIA's hosted API catalog (build.nvidia.com),
	// whose free developer tier serves open models over an
	// OpenAI-compatible Chat Completions endpoint.
	AIProviderNVIDIA AIProvider = "nvidia"
)

func (p AIProvider) Valid() bool {
	switch p {
	case AIProviderOpenAI, AIProviderAnthropic, AIProviderGemini, AIProviderOllama, AIProviderNVIDIA:
		return true
	default:
		return false
	}
}

// AIConnection is the one AI provider a tenant has configured for document
// parsing (see procurement.AIDocumentParser) -- one row per tenant
// (UNIQUE tenant_id), switching providers overwrites it rather than adding
// a second row, per the "one active connection at a time" decision.
type AIConnection struct {
	ID       string
	TenantID string
	Provider AIProvider
	// APIKeyEncrypted is empty for AIProviderOllama -- a self-hosted server
	// reachable at BaseURL needs no credential.
	APIKeyEncrypted string
	// BaseURL overrides the provider's default API endpoint -- required for
	// AIProviderOllama (points at the tenant's self-hosted instance),
	// optional for the others (proxies/private endpoints).
	BaseURL string
	// Model is the provider-specific model identifier -- empty means "use
	// the adapter's built-in default" (e.g. OpenAI's gpt-4o).
	Model string
	// FallbackModels is a comma-separated list of model IDs to retry, in
	// order, when Model comes back rate-limited (HTTP 429) -- e.g. Gemini's
	// free tier caps some models at 20 requests/day and others at 500.
	// Empty means no fallback: a rate limit just fails, same as before this
	// field existed. Only infrastructure/ai's Gemini adapter reads it
	// today.
	FallbackModels string
	Status         Status
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RequiresAPIKey reports whether Provider needs a non-empty APIKeyEncrypted
// to connect -- false only for Ollama.
func (p AIProvider) RequiresAPIKey() bool {
	return p != AIProviderOllama
}
