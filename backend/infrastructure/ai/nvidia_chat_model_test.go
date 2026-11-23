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

func toolCallCompletion(id, name, args string) map[string]any {
	return map[string]any{
		"id": "chatcmpl-2", "object": "chat.completion", "created": 1, "model": "m",
		"choices": []map[string]any{{
			"index": 0, "finish_reason": "tool_calls",
			"message": map[string]any{
				"role": "assistant", "content": nil,
				"tool_calls": []map[string]any{{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": args}}},
			},
		}},
	}
}

var orderTool = assistant.ToolDefinition{
	Name: "get_sales_order", Description: "Look up one order.",
	Parameters: map[string]any{"type": "object", "properties": map[string]any{"number": map[string]any{"type": "string"}}},
}

func TestNVIDIAChatModel_sendsConversationAndToolsAndReturnsToolCalls(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer nvapi-test", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, toolCallCompletion("call_9", "get_sales_order", `{"number":"CO-13"}`))
	}))
	defer server.Close()

	model := ai.NewNVIDIAChatModel("nvapi-test").WithBaseURL(server.URL)
	reply, err := model.Next(context.Background(), assistant.ChatTurn{
		System: "be helpful",
		Messages: []assistant.Message{
			{Role: assistant.RoleUser, Content: "status CO-12 and CO-13?"},
			{Role: assistant.RoleAssistant, ToolCalls: []assistant.ToolCall{{ID: "call_1", Name: "get_sales_order", Arguments: `{"number":"CO-12"}`}}},
			{Role: assistant.RoleTool, ToolCallID: "call_1", ToolName: "get_sales_order", Content: `{"status":"shipped"}`},
			{Role: assistant.RoleAssistant, Question: true, Notice: assistant.NoticeTooManySteps},
			{Role: assistant.RoleUser, Content: "try again"},
		},
		Tools: []assistant.ToolDefinition{orderTool},
	})

	require.NoError(t, err)
	assert.Empty(t, reply.Content)
	assert.Equal(t, []assistant.ToolCall{{ID: "call_9", Name: "get_sales_order", Arguments: `{"number":"CO-13"}`}}, reply.ToolCalls)

	assert.Equal(t, "mistralai/mistral-large-4-1t-a52b", body["model"])
	assert.EqualValues(t, 0, body["temperature"])
	msgs := body["messages"].([]any)
	require.Len(t, msgs, 6)
	assert.Equal(t, map[string]any{"role": "system", "content": "be helpful"}, msgs[0])
	assert.Equal(t, "user", msgs[1].(map[string]any)["role"])
	call := msgs[2].(map[string]any)
	assert.Equal(t, "assistant", call["role"])
	calls := call["tool_calls"].([]any)
	assert.Equal(t, "call_1", calls[0].(map[string]any)["id"])
	assert.Equal(t, map[string]any{"name": "get_sales_order", "arguments": `{"number":"CO-12"}`}, calls[0].(map[string]any)["function"])
	assert.Equal(t, map[string]any{"role": "tool", "content": `{"status":"shipped"}`, "tool_call_id": "call_1"}, msgs[3])
	notice := msgs[4].(map[string]any)
	assert.Equal(t, "assistant", notice["role"])
	assert.NotEmpty(t, notice["content"], "an app-written notice still reaches the model as text")

	tools := body["tools"].([]any)
	require.Len(t, tools, 1)
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	assert.Equal(t, "get_sales_order", fn["name"])
	assert.Equal(t, "Look up one order.", fn["description"])
	assert.Equal(t, "object", fn["parameters"].(map[string]any)["type"])
}

func TestNVIDIAChatModel_returnsFinalText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, chatCompletion("CO-12 has shipped."))
	}))
	defer server.Close()

	reply, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).Next(context.Background(),
		assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "status?"}}})

	require.NoError(t, err)
	assert.Equal(t, "CO-12 has shipped.", reply.Content)
	assert.Empty(t, reply.ToolCalls)
}

func TestNVIDIAChatModel_busyErrorsAreTransient(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, status, map[string]any{"error": map[string]any{"message": "busy"}})
		}))

		_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).Next(context.Background(),
			assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})
		server.Close()

		require.Error(t, err)
		assert.ErrorIs(t, err, assistant.ErrModelBusy, "HTTP %d", status)
	}
}

func TestNVIDIAChatModel_authErrorIsNotTransient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAChatModel("bad").WithBaseURL(server.URL).Next(context.Background(),
		assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.Error(t, err)
	assert.NotErrorIs(t, err, assistant.ErrModelBusy)
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestNVIDIAChatModel_fallsBackOnRateLimit(t *testing.T) {
	var models []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		models = append(models, body["model"].(string))
		if len(models) == 1 {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"message": "rate limited"}})
			return
		}
		writeJSON(w, http.StatusOK, chatCompletion("ok"))
	}))
	defer server.Close()

	reply, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).WithFallbackModels("meta/llama-3.1-8b-instruct").
		Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.NoError(t, err)
	assert.Equal(t, "ok", reply.Content)
	assert.Equal(t, []string{"mistralai/mistral-large-4-1t-a52b", "meta/llama-3.1-8b-instruct"}, models)
}

func TestNVIDIAChatModel_errorNamesTheModelAndTheProvidersReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusGone, map[string]any{"detail": "Model has reached its end of life"})
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).Next(context.Background(),
		assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mistralai/mistral-large-4-1t-a52b")
	assert.Contains(t, err.Error(), "end of life")
	assert.NotErrorIs(t, err, assistant.ErrModelBusy, "a retired model won't come back by waiting")
}

func nvidiaBodies(t *testing.T, status func(model string) int) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		bodies = append(bodies, body)
		if code := status(body["model"].(string)); code != http.StatusOK {
			writeJSON(w, code, map[string]any{"error": map[string]any{"message": "rate limited"}})
			return
		}
		writeJSON(w, http.StatusOK, chatCompletion("ok"))
	}))
	t.Cleanup(server.Close)
	return server, &bodies
}

func TestNVIDIAChatModel_mistralLargeSkipsReasoning(t *testing.T) {
	server, bodies := nvidiaBodies(t, func(string) int { return http.StatusOK })

	_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).Next(context.Background(),
		assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.NoError(t, err)
	assert.Equal(t, "mistralai/mistral-large-4-1t-a52b", (*bodies)[0]["model"])
	assert.Equal(t, "none", (*bodies)[0]["reasoning_effort"], "no thinking: faster, and the answer fits the token budget")
}

func TestNVIDIAChatModel_otherModelsGetNoReasoningEffort(t *testing.T) {
	server, bodies := nvidiaBodies(t, func(string) int { return http.StatusOK })

	_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).WithModel("minimaxai/minimax-m3.1-flash").
		Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.NoError(t, err)
	assert.NotContains(t, (*bodies)[0], "reasoning_effort", "models that don't know it may reject it")
}

func TestNVIDIAChatModel_defaultFallbackWhenNoneIsSet(t *testing.T) {
	server, bodies := nvidiaBodies(t, func(model string) int {
		if model == "mistralai/mistral-large-4-1t-a52b" {
			return http.StatusTooManyRequests
		}
		return http.StatusOK
	})

	_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).WithFallbackModels("").
		Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

	require.NoError(t, err)
	require.Len(t, *bodies, 2)
	assert.Equal(t, "minimaxai/minimax-m3.1-flash", (*bodies)[1]["model"])
}

func TestNVIDIAWhatsAppIntentParser_mistralLargeSkipsReasoning(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, chatCompletion(`{"command":"UNKNOWN","product_query":""}`))
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAWhatsAppIntentParser("k").WithBaseURL(server.URL).ParseIntent(context.Background(), "oi")

	require.NoError(t, err)
	assert.Equal(t, "none", body["reasoning_effort"])
}

func TestNVIDIAChatModel_aRetiredOrUnknownModelMovesToTheFallback(t *testing.T) {
	for _, status := range []int{http.StatusGone, http.StatusNotFound} {
		server, bodies := nvidiaBodies(t, func(model string) int {
			if model == "mistralai/mistral-large-4-1t-a52b" {
				return status
			}
			return http.StatusOK
		})

		_, err := ai.NewNVIDIAChatModel("k").WithBaseURL(server.URL).WithFallbackModels("minimaxai/minimax-m3.1-flash").
			Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "x"}}})

		require.NoError(t, err, "HTTP %d", status)
		assert.Len(t, *bodies, 2)
	}
}
