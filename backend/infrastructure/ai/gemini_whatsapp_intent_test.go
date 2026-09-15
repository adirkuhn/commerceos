package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adirkuhn/commerceos/infrastructure/ai"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiWhatsAppIntentParser_ParseIntent_success(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
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

	genCfg, _ := capturedBody["generationConfig"].(map[string]any)
	require.NotNil(t, genCfg, "generationConfig missing from request: %v", capturedBody)
	thinkingCfg, _ := genCfg["thinkingConfig"].(map[string]any)
	require.NotNil(t, thinkingCfg, "thinking must be explicitly disabled for this classification-only call")
	assert.EqualValues(t, 0, thinkingCfg["thinkingBudget"])
}

func TestGeminiWhatsAppIntentParser_ParseIntent_fallsBackToNextModelOn429(t *testing.T) {
	var calledModels []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledModels = append(calledModels, r.URL.Path)
		if strings.Contains(r.URL.Path, "primary-model") {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 429, "message": "quota exceeded", "status": "RESOURCE_EXHAUSTED"},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role":  "model",
						"parts": []map[string]any{{"text": `{"command":"STOCK","product_query":"flocao"}`}},
					},
					"finishReason": "STOP",
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL).
		WithModel("primary-model").WithFallbackModels("fallback-model-1,fallback-model-2")
	intent, err := parser.ParseIntent(context.Background(), "tem flocao?")

	require.NoError(t, err)
	assert.Equal(t, whatsappDomain.IntentCommandStock, intent.Command)
	assert.Equal(t, "flocao", intent.ProductQuery)
	require.Len(t, calledModels, 2, "primary rate-limited, first fallback succeeds -- second fallback never called")
	assert.Contains(t, calledModels[0], "primary-model")
	assert.Contains(t, calledModels[1], "fallback-model-1")
}

func TestGeminiWhatsAppIntentParser_ParseIntent_allModelsRateLimited_returnsLastError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 429, "message": "quota exceeded", "status": "RESOURCE_EXHAUSTED"},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL).
		WithModel("primary-model").WithFallbackModels("fallback-model")
	_, err := parser.ParseIntent(context.Background(), "tem flocao?")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
	assert.Equal(t, 2, callCount, "tries primary + the one fallback, then gives up")
}

func TestGeminiWhatsAppIntentParser_ParseIntent_nonRateLimitError_neverRetries(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []map[string]any{}})
	}))
	defer server.Close()

	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL).
		WithModel("primary-model").WithFallbackModels("fallback-model")
	_, err := parser.ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no text found")
	assert.Equal(t, 1, callCount, "a non-429 failure would fail identically against every model -- never worth retrying")
}

func TestGeminiWhatsAppIntentParser_WithFallbackModels_dropsBlankAndDuplicatePrimaryEntries(t *testing.T) {
	var calledModels []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledModels = append(calledModels, r.URL.Path)
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 429}})
	}))
	defer server.Close()

	// "primary-model" (== the model itself) and the blank entry must both
	// be dropped -- only "real-fallback" should ever be called.
	parser := ai.NewGeminiWhatsAppIntentParser("test-key").WithBaseURL(server.URL).
		WithModel("primary-model").WithFallbackModels("primary-model,,real-fallback")
	_, err := parser.ParseIntent(context.Background(), "oi")

	require.Error(t, err)
	require.Len(t, calledModels, 2)
	assert.Contains(t, calledModels[0], "primary-model")
	assert.Contains(t, calledModels[1], "real-fallback")
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
