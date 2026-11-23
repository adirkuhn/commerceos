package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
)

func geminiReply(parts ...map[string]any) map[string]any {
	return map[string]any{"candidates": []map[string]any{{"content": map[string]any{"role": "model", "parts": parts}, "finishReason": "STOP"}}}
}

func TestGeminiChatModel_mapsConversationToolsAndFunctionCalls(t *testing.T) {
	var body map[string]any
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		require.Equal(t, "gm-key", r.Header.Get("X-Goog-Api-Key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, geminiReply(
			map[string]any{"text": "Checking CO-13."},
			map[string]any{"functionCall": map[string]any{"name": "get_sales_order", "args": map[string]any{"order": "CO-13"}}},
		))
	}))
	defer server.Close()

	turn := sampleTurn()
	turn.Messages[1].ToolCalls[0].Signature = "c2lnLTA=" // Gemini made this call itself
	reply, err := ai.NewGeminiChatModel("gm-key").WithBaseURL(server.URL).Next(context.Background(), turn)

	require.NoError(t, err)
	assert.Contains(t, path, "gemini-3.6-flash:generateContent")
	assert.Equal(t, "Checking CO-13.", reply.Content)
	require.Len(t, reply.ToolCalls, 1)
	assert.NotEmpty(t, reply.ToolCalls[0].ID, "Gemini sends no call id, so one is generated")
	assert.Equal(t, "get_sales_order", reply.ToolCalls[0].Name)
	assert.JSONEq(t, `{"order":"CO-13"}`, reply.ToolCalls[0].Arguments)

	sys := body["systemInstruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)
	assert.Equal(t, "be helpful", sys["text"])

	contents := body["contents"].([]any)
	roles := make([]string, len(contents))
	for i, c := range contents {
		roles[i] = c.(map[string]any)["role"].(string)
	}
	assert.Equal(t, []string{"user", "model", "user", "model", "user"}, roles)

	modelParts := contents[1].(map[string]any)["parts"].([]any)
	call := modelParts[1].(map[string]any)["functionCall"].(map[string]any)
	assert.Equal(t, "get_sales_order", call["name"])
	assert.Equal(t, map[string]any{"order": "CO-12"}, call["args"])

	resp := contents[2].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
	assert.Equal(t, "get_sales_order", resp["name"])
	assert.Equal(t, map[string]any{"status": "shipped"}, resp["response"])

	decl := body["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)[0].(map[string]any)
	assert.Equal(t, "get_sales_order", decl["name"])
	assert.Equal(t, "object", decl["parametersJsonSchema"].(map[string]any)["type"])
}

func TestGeminiChatModel_nonObjectToolResultIsWrapped(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, geminiReply(map[string]any{"text": "ok"}))
	}))
	defer server.Close()
	turn := assistant.ChatTurn{Messages: []assistant.Message{
		{Role: assistant.RoleUser, Content: "x"},
		{Role: assistant.RoleAssistant, ToolCalls: []assistant.ToolCall{{ID: "c", Name: "search", Arguments: `{}`, Signature: "c2ln"}}},
		{Role: assistant.RoleTool, ToolCallID: "c", ToolName: "search", Content: `[1,2]`},
	}}

	_, err := ai.NewGeminiChatModel("k").WithBaseURL(server.URL).Next(context.Background(), turn)

	require.NoError(t, err)
	resp := body["contents"].([]any)[2].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
	assert.Equal(t, map[string]any{"output": "[1,2]"}, resp["response"])
}

func TestGeminiChatModel_fallsBackOnRateLimitThenBusy(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"code": 429, "message": "quota", "status": "RESOURCE_EXHAUSTED"}})
	}))
	defer server.Close()

	_, err := ai.NewGeminiChatModel("k").WithBaseURL(server.URL).WithFallbackModels("gemini-3.5-flash-lite").
		Next(context.Background(), sampleTurn())

	assert.ErrorIs(t, err, assistant.ErrModelBusy)
	require.Len(t, paths, 2)
	assert.True(t, strings.Contains(paths[1], "gemini-3.5-flash-lite"))
}

func TestGeminiChatModel_keepsTheThoughtSignatureOfItsCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, geminiReply(map[string]any{
			"functionCall":     map[string]any{"name": "search_customers", "args": map[string]any{"query": "Ana"}},
			"thoughtSignature": "c2lnLTE=",
		}))
	}))
	defer server.Close()

	reply, err := ai.NewGeminiChatModel("k").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	require.Len(t, reply.ToolCalls, 1)
	assert.Equal(t, "c2lnLTE=", reply.ToolCalls[0].Signature)
}

func TestGeminiChatModel_sendsItsOwnSignedCallsBackWithTheirSignature(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, geminiReply(map[string]any{"text": "ok"}))
	}))
	defer server.Close()
	turn := assistant.ChatTurn{Messages: []assistant.Message{
		{Role: assistant.RoleUser, Content: "find Ana"},
		{Role: assistant.RoleAssistant, ToolCalls: []assistant.ToolCall{
			{ID: "c1", Name: "search_customers", Arguments: `{"query":"Ana"}`, Signature: "c2lnLTE="},
			{ID: "c2", Name: "search_suppliers", Arguments: `{"query":"Ana"}`},
		}},
		{Role: assistant.RoleTool, ToolCallID: "c1", ToolName: "search_customers", Content: `{"hits":1}`},
		{Role: assistant.RoleTool, ToolCallID: "c2", ToolName: "search_suppliers", Content: `{"hits":0}`},
	}}

	_, err := ai.NewGeminiChatModel("k").WithBaseURL(server.URL).Next(context.Background(), turn)

	require.NoError(t, err)
	contents := body["contents"].([]any)
	parts := contents[1].(map[string]any)["parts"].([]any)
	require.Len(t, parts, 2, "a signed step keeps its function calls -- parallel calls after the first carry none")
	assert.Equal(t, "c2lnLTE=", parts[0].(map[string]any)["thoughtSignature"])
	assert.NotNil(t, parts[0].(map[string]any)["functionCall"])
	assert.Nil(t, parts[1].(map[string]any)["thoughtSignature"])
	results := contents[2].(map[string]any)["parts"].([]any)
	assert.NotNil(t, results[0].(map[string]any)["functionResponse"])
}

func TestGeminiChatModel_callsAnotherModelMadeGoAsText(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, geminiReply(map[string]any{"text": "ok"}))
	}))
	defer server.Close()

	// sampleTurn's call came from another provider: no signature.
	_, err := ai.NewGeminiChatModel("k").WithBaseURL(server.URL).Next(context.Background(), sampleTurn())

	require.NoError(t, err)
	raw, _ := json.Marshal(body["contents"])
	assert.NotContains(t, string(raw), "functionCall", "Gemini refuses calls without its signature")
	assert.NotContains(t, string(raw), "functionResponse")
	contents := body["contents"].([]any)
	step := contents[1].(map[string]any)
	assert.Equal(t, "model", step["role"])
	stepText, _ := json.Marshal(step["parts"])
	assert.Contains(t, string(stepText), "get_sales_order")
	assert.Contains(t, string(stepText), `CO-12`)
	result := contents[2].(map[string]any)
	assert.Equal(t, "user", result["role"])
	resultText, _ := json.Marshal(result["parts"])
	assert.Contains(t, string(resultText), "shipped", "the facts the other model looked up are kept")
}
