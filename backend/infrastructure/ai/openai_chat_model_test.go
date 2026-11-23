package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
)

func TestOpenAIChatModel_usesChatCompletionsWithTools(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, toolCallCompletion("call_2", "get_sales_order", `{"order":"CO-13"}`))
	}))
	defer server.Close()

	reply, err := ai.NewOpenAIChatModel("sk-test").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	assert.Equal(t, []assistant.ToolCall{{ID: "call_2", Name: "get_sales_order", Arguments: `{"order":"CO-13"}`}}, reply.ToolCalls)
	assert.Equal(t, "gpt-4o", body["model"])
	assert.Len(t, body["messages"], 6)
	assert.Len(t, body["tools"], 1)
}

func TestOpenAIChatModel_rateLimitIsBusyAndCustomModelIsUsed(t *testing.T) {
	var model string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		model = body["model"].(string)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"message": "quota"}})
	}))
	defer server.Close()

	_, err := ai.NewOpenAIChatModel("k").WithBaseURL(server.URL).WithModel("gpt-5-mini").Next(context.Background(), sampleTurn())

	assert.ErrorIs(t, err, assistant.ErrModelBusy)
	assert.Equal(t, "gpt-5-mini", model)
	assert.Contains(t, err.Error(), "openai")
}
