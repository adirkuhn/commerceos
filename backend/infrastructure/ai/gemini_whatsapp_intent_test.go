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

func TestGeminiWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role": "model",
						"parts": []map[string]any{
							{"text": `{"command":"INFO","product_query":"cha verde"}`},
						},
					},
					"finishReason": "STOP",
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "me fala sobre o cha verde")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandInfo, intent.Command)
	assert.Equal(t, "cha verde", intent.ProductQuery)
}

func TestGeminiWhatsAppIntentParser_ParseIntent_returnsErrorWhenNoText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []map[string]any{}})
	}))
	defer server.Close()

	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no text found")
}
