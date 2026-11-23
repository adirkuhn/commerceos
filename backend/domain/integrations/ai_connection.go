package integrations

import (
	"sort"
	"time"
	"unicode/utf8"
)

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

// AIConnection is one AI provider a tenant configured. A tenant keeps an
// ordered list of them and every AI feature routes through it: the first
// enabled connection is tried, and any failure moves on to the next
// (Position ascending). The same provider may appear more than once (two
// free-tier keys double the quota).
type AIConnection struct {
	ID       string
	TenantID string
	// Position orders the list: 0 is tried first.
	Position int
	// Label tells two connections of the same provider apart ("NVIDIA key
	// 2"); optional.
	Label string
	// Enabled false keeps the connection configured but skips it.
	Enabled  bool
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

// MaxAILabelLength caps AIConnection.Label.
const MaxAILabelLength = 80

// Validate checks what a connection needs to be usable.
func (c AIConnection) Validate() error {
	switch {
	case !c.Provider.Valid():
		return ErrProviderNotSupported
	case c.Provider.RequiresAPIKey() && c.APIKeyEncrypted == "":
		return ErrAPIKeyRequired
	case c.Provider == AIProviderOllama && c.BaseURL == "":
		return ErrBaseURLRequired
	case utf8.RuneCountInString(c.Label) > MaxAILabelLength:
		return ErrAILabelTooLong
	}
	return nil
}

// NextAIPosition is where a new connection goes: after the last one.
func NextAIPosition(existing []AIConnection) int {
	next := 0
	for _, c := range existing {
		if c.Position >= next {
			next = c.Position + 1
		}
	}
	return next
}

// ReorderAIConnections puts existing in the order of ids, which must name
// every connection exactly once, and renumbers positions from 0.
func ReorderAIConnections(existing []AIConnection, ids []string) ([]AIConnection, error) {
	if len(ids) != len(existing) {
		return nil, ErrInvalidAIOrder
	}
	byID := make(map[string]AIConnection, len(existing))
	for _, c := range existing {
		byID[c.ID] = c
	}
	out := make([]AIConnection, 0, len(ids))
	for i, id := range ids {
		c, ok := byID[id]
		if !ok {
			return nil, ErrInvalidAIOrder
		}
		delete(byID, id)
		c.Position = i
		out = append(out, c)
	}
	return out, nil
}

// CompactAIPositions sorts by position and renumbers from 0 -- after a
// delete leaves a gap.
func CompactAIPositions(cs []AIConnection) []AIConnection {
	out := append([]AIConnection(nil), cs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	for i := range out {
		out[i].Position = i
	}
	return out
}
