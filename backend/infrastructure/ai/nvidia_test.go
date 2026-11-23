package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
)

// chatCompletion is the minimal OpenAI-compatible Chat Completions response
// NVIDIA's API (integrate.api.nvidia.com/v1) returns.
func chatCompletion(content string) map[string]any {
	return map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion", "created": 1, "model": "m",
		"choices": []map[string]any{{
			"index": 0, "finish_reason": "stop",
			"message": map[string]any{"role": "assistant", "content": content},
		}},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer nvapi-test", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, chatCompletion(`{"command":"PRICE","product_query":"doce de leite"}`))
	}))
	defer server.Close()

	parser := ai.NewNVIDIAWhatsAppIntentParser("nvapi-test").WithBaseURL(server.URL)
	intent, err := parser.ParseIntent(context.Background(), "quanto custa doce de leite?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandPrice, intent.Command)
	assert.Equal(t, "doce de leite", intent.ProductQuery)
	assert.Equal(t, "openai/gpt-oss-120b", body["model"], "default model")
	assert.EqualValues(t, 0, body["temperature"])
	assert.EqualValues(t, 200, body["max_tokens"])

	nvext, ok := body["nvext"].(map[string]any)
	require.True(t, ok, "structured output is forced through NVIDIA's nvext.guided_json")
	assert.NotNil(t, nvext["guided_json"])

	msgs := body["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	assert.Contains(t, last["content"], "quanto custa doce de leite?")
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_toleratesMarkdownFence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, chatCompletion("```json\n{\"command\":\"STOCK\",\"product_query\":\"farofa\"}\n```"))
	}))
	defer server.Close()

	intent, err := ai.NewNVIDIAWhatsAppIntentParser("k").WithBaseURL(server.URL).ParseIntent(context.Background(), "tem farofa?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandStock, intent.Command)
	assert.Equal(t, "farofa", intent.ProductQuery)
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_customModel(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, chatCompletion(`{"command":"UNKNOWN","product_query":""}`))
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAWhatsAppIntentParser("k").WithBaseURL(server.URL).
		WithModel("nvidia/llama-3.3-nemotron-super-49b-v1").ParseIntent(context.Background(), "oi")

	require.NoError(t, err)
	assert.Equal(t, "nvidia/llama-3.3-nemotron-super-49b-v1", body["model"])
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_fallsBackOnRateLimit(t *testing.T) {
	var models []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		model := body["model"].(string)
		models = append(models, model)
		if model == "openai/gpt-oss-120b" {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"message": "rate limited"}})
			return
		}
		writeJSON(w, http.StatusOK, chatCompletion(`{"command":"INFO","product_query":"cafe"}`))
	}))
	defer server.Close()

	intent, err := ai.NewNVIDIAWhatsAppIntentParser("k").WithBaseURL(server.URL).
		WithFallbackModels(" , openai/gpt-oss-120b, meta/llama-3.1-8b-instruct").
		ParseIntent(context.Background(), "me fala do cafe")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandInfo, intent.Command)
	assert.Equal(t, []string{"openai/gpt-oss-120b", "meta/llama-3.1-8b-instruct"}, models,
		"blank and primary-duplicate fallback entries are dropped")
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_doesNotFallBackOnOtherErrors(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAWhatsAppIntentParser("bad").WithBaseURL(server.URL).
		WithFallbackModels("meta/llama-3.1-8b-instruct").ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
	assert.Equal(t, 1, calls, "an auth error fails identically on every model, so no fallback")
}

func TestNVIDIAWhatsAppIntentParser_ParseIntent_invalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, chatCompletion("Sure! The command is STOCK."))
	}))
	defer server.Close()

	_, err := ai.NewNVIDIAWhatsAppIntentParser("k").WithBaseURL(server.URL).ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nvidia")
}

func TestNVIDIADocumentParser_ExtractLineItems_success(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		writeJSON(w, http.StatusOK, chatCompletion(`{"line_items":[{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500,"line_type":"product","batch_code":"","expiry_date":"2027-03-01"}],"supplier_name":"Acme Foods"}`))
	}))
	defer server.Close()

	result, err := ai.NewNVIDIADocumentParser("nvapi-test").WithBaseURL(server.URL).ExtractLineItems(context.Background(),
		procurementDomain.DocumentInput{FileName: "po.pdf", ContentType: "application/pdf", ExtractedText: "WID-001 Widget A qty 10 $5.00"})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, "WID-001", result.LineItems[0].RawSKU)
	assert.EqualValues(t, 10, result.LineItems[0].Quantity)
	assert.EqualValues(t, 500, result.LineItems[0].UnitCostMinorUnits)
	require.NotNil(t, result.LineItems[0].ExpiryDate)
	assert.Equal(t, "2027-03-01", result.LineItems[0].ExpiryDate.Format("2006-01-02"))
	assert.Equal(t, "Acme Foods", result.SupplierName)

	msgs := body["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	assert.Contains(t, last["content"], "WID-001 Widget A qty 10 $5.00", "prompts on the PDF's extracted text")
}

func TestNVIDIADocumentParser_ExtractLineItems_requiresExtractedText(t *testing.T) {
	_, err := ai.NewNVIDIADocumentParser("k").ExtractLineItems(context.Background(),
		procurementDomain.DocumentInput{FileName: "scan.pdf", ContentType: "application/pdf", Data: []byte("%PDF")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no extracted text")
}
