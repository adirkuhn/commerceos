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

func TestOllamaChatModel_mapsConversationToolsAndToolCalls(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/chat", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{"role": "assistant", "content": "",
				"tool_calls": []map[string]any{{"function": map[string]any{"name": "get_sales_order", "arguments": map[string]any{"order": "CO-13"}}}}},
		})
	}))
	defer server.Close()

	reply, err := ai.NewOllamaChatModel(server.URL).Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	require.Len(t, reply.ToolCalls, 1)
	assert.NotEmpty(t, reply.ToolCalls[0].ID)
	assert.Equal(t, "get_sales_order", reply.ToolCalls[0].Name)
	assert.JSONEq(t, `{"order":"CO-13"}`, reply.ToolCalls[0].Arguments)

	assert.Equal(t, "llama3.2", body["model"])
	assert.Equal(t, false, body["stream"])
	msgs := body["messages"].([]any)
	require.Len(t, msgs, 6)
	assert.Equal(t, map[string]any{"role": "system", "content": "be helpful"}, msgs[0])
	call := msgs[2].(map[string]any)
	assert.Equal(t, "assistant", call["role"])
	fn := call["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)
	assert.Equal(t, "get_sales_order", fn["name"])
	assert.Equal(t, map[string]any{"order": "CO-12"}, fn["arguments"])
	result := msgs[3].(map[string]any)
	assert.Equal(t, "tool", result["role"])
	assert.Equal(t, "get_sales_order", result["tool_name"])
	assert.Equal(t, `{"status":"shipped"}`, result["content"])

	tool := body["tools"].([]any)[0].(map[string]any)
	assert.Equal(t, "function", tool["type"])
	assert.Equal(t, "get_sales_order", tool["function"].(map[string]any)["name"])
}

func TestOllamaChatModel_textAnswerAndCustomModel(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeNDJSON(w, map[string]any{"model": "qwen3", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{"role": "assistant", "content": "CO-13 is pending."}})
	}))
	defer server.Close()

	reply, err := ai.NewOllamaChatModel(server.URL).WithModel("qwen3").Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	assert.Equal(t, "CO-13 is pending.", reply.Content)
	assert.Equal(t, "qwen3", body["model"])
}

func TestOllamaChatModel_serverDownOrBusyIsTransient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"server busy"}`))
	}))
	_, err := ai.NewOllamaChatModel(server.URL).Next(context.Background(), sampleTurn())
	server.Close()
	assert.ErrorIs(t, err, assistant.ErrModelBusy)

	_, err = ai.NewOllamaChatModel(server.URL).Next(context.Background(), sampleTurn())
	assert.ErrorIs(t, err, assistant.ErrModelBusy, "a self-hosted server that is down comes back later")
}

func TestOllamaChatModel_modelErrorIsNotTransient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"llama2 does not support tools"}`))
	}))
	defer server.Close()

	_, err := ai.NewOllamaChatModel(server.URL).Next(context.Background(), sampleTurn())

	require.Error(t, err)
	assert.NotErrorIs(t, err, assistant.ErrModelBusy)
	assert.Contains(t, err.Error(), "does not support tools")
}

func TestOllamaChatModel_errorNamesTheModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	_, err := ai.NewOllamaChatModel(server.URL).WithModel("qwen9").Next(context.Background(), sampleTurn())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ollama (qwen9)")
	assert.Contains(t, err.Error(), "model not found")
}
