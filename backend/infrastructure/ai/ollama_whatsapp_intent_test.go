package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adirkuhn/commerceos/infrastructure/ai"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{
				"role":    "assistant",
				"content": `{"command":"STOCK","product_query":"abc123"}`,
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOllamaWhatsAppIntentParser(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "heb je abc123 op voorraad?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandStock, intent.Command)
	assert.Equal(t, "abc123", intent.ProductQuery)
}

func TestOllamaWhatsAppIntentParser_ParseIntent_returnsErrorOnMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{"role": "assistant", "content": "not json"},
		})
	}))
	defer server.Close()

	parser := ai.NewOllamaWhatsAppIntentParser(server.URL)
	_, err := parser.ParseIntent(context.Background(), "oi")

	require.Error(t, err)
}
