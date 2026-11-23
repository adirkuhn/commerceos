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

func anthropicMessage(content ...map[string]any) map[string]any {
	return map[string]any{
		"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
		"content": content, "stop_reason": "tool_use", "stop_sequence": nil,
		"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
	}
}

func TestAnthropicChatModel_mapsConversationToolsAndToolUse(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, "sk-ant", r.Header.Get("X-Api-Key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, anthropicMessage(
			map[string]any{"type": "text", "text": "Checking CO-13."},
			map[string]any{"type": "tool_use", "id": "toolu_9", "name": "get_sales_order", "input": map[string]any{"order": "CO-13"}},
		))
	}))
	defer server.Close()

	reply, err := ai.NewAnthropicChatModel("sk-ant").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	assert.Equal(t, "Checking CO-13.", reply.Content)
	require.Len(t, reply.ToolCalls, 1)
	assert.Equal(t, "toolu_9", reply.ToolCalls[0].ID)
	assert.Equal(t, "get_sales_order", reply.ToolCalls[0].Name)
	assert.JSONEq(t, `{"order":"CO-13"}`, reply.ToolCalls[0].Arguments)

	assert.Equal(t, "claude-sonnet-5", body["model"])
	assert.Equal(t, []any{map[string]any{"type": "text", "text": "be helpful"}}, body["system"])

	msgs := body["messages"].([]any)
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.(map[string]any)["role"].(string)
	}
	// Anthropic needs alternating turns: the tool result rides in a user
	// turn, and the notice + follow-up merge with their neighbours.
	assert.Equal(t, []string{"user", "assistant", "user", "assistant", "user"}, roles)

	call := msgs[1].(map[string]any)["content"].([]any)
	assert.Equal(t, "text", call[0].(map[string]any)["type"])
	use := call[1].(map[string]any)
	assert.Equal(t, "tool_use", use["type"])
	assert.Equal(t, "call_1", use["id"])
	assert.Equal(t, map[string]any{"order": "CO-12"}, use["input"])

	result := msgs[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	assert.Equal(t, "tool_result", result["type"])
	assert.Equal(t, "call_1", result["tool_use_id"])

	notice := msgs[3].(map[string]any)["content"].([]any)[0].(map[string]any)
	assert.NotEmpty(t, notice["text"])

	tools := body["tools"].([]any)
	tool := tools[0].(map[string]any)
	assert.Equal(t, "get_sales_order", tool["name"])
	assert.Equal(t, "Look up one order.", tool["description"])
	assert.Equal(t, "object", tool["input_schema"].(map[string]any)["type"])
}

func TestAnthropicChatModel_overloadedIsBusy(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, 529} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, status, map[string]any{"type": "error", "error": map[string]any{"type": "overloaded_error", "message": "busy"}})
		}))

		_, err := ai.NewAnthropicChatModel("k").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())
		server.Close()

		assert.ErrorIs(t, err, assistant.ErrModelBusy, "HTTP %d", status)
	}
}

func TestAnthropicChatModel_authErrorIsNotBusy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"type": "error", "error": map[string]any{"type": "authentication_error", "message": "invalid x-api-key"}})
	}))
	defer server.Close()

	_, err := ai.NewAnthropicChatModel("bad").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())

	require.Error(t, err)
	assert.NotErrorIs(t, err, assistant.ErrModelBusy)
}

func TestAnthropicChatModel_errorNamesTheModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"type": "error", "error": map[string]any{"type": "not_found_error", "message": "model: claude-old"}})
	}))
	defer server.Close()

	_, err := ai.NewAnthropicChatModel("k").WithBaseURL(server.URL).WithModel("claude-old").Next(context.Background(), sampleTurn())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "anthropic (claude-old)")
}
