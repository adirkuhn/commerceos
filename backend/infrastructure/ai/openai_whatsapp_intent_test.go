package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adirkuhn/commerceos/infrastructure/ai"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/responses", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{"type": "output_text", "text": `{"command":"STOCK","product_query":"cha de limao"}`},
					},
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "voce tem estoque de cha de limao?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandStock, intent.Command)
	assert.Equal(t, "cha de limao", intent.ProductQuery)
	assert.Equal(t, "gpt-4o", capturedBody["model"])
	assert.EqualValues(t, 200, capturedBody["max_output_tokens"])
}

func TestOpenAIWhatsAppIntentParser_ParseIntent_unrecognizedCommandFallsBackToUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{"type": "message", "content": []map[string]any{
					{"type": "output_text", "text": `{"command":"CHAT","product_query":"whatever"}`},
				}},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "how's the weather?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandUnknown, intent.Command)
	assert.Empty(t, intent.ProductQuery, "product_query is cleared for an unrecognized/unknown command")
}

func TestOpenAIWhatsAppIntentParser_ParseIntent_returnsErrorOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer server.Close()

	parser := ai.NewOpenAIWhatsAppIntentParser("bad-key").WithBaseURL(server.URL)
	_, err := parser.ParseIntent(context.Background(), "tem estoque?")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
}
