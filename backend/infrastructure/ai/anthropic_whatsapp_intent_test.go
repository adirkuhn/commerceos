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

func TestAnthropicWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, "test-key", r.Header.Get("X-Api-Key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content": []map[string]any{
				{
					"type": "tool_use", "id": "toolu_1", "name": "classify_whatsapp_intent",
					"input": map[string]any{"command": "PRICE", "product_query": "camiseta brasil"},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "quanto ta a camiseta brasil?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandPrice, intent.Command)
	assert.Equal(t, "camiseta brasil", intent.ProductQuery)
	assert.EqualValues(t, 100, capturedBody["max_tokens"])
}

func TestAnthropicWhatsAppIntentParser_ParseIntent_returnsErrorWhenNoToolUseBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content":     []map[string]any{{"type": "text", "text": "hi there"}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicWhatsAppIntentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no classify_whatsapp_intent tool_use block")
}
