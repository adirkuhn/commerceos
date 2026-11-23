package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func TestWhatsAppIntentParserResolver_returnsNilWhenNotConfigured(t *testing.T) {
	repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil })
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	parser, err := resolver.Resolve(context.Background())

	require.NoError(t, err)
	assert.Nil(t, parser)
}

func TestWhatsAppIntentParserResolver_buildsCorrectAdapterPerProvider(t *testing.T) {
	cases := []struct {
		provider integrationsDomain.AIProvider
		baseURL  string
		wantType any
	}{
		{integrationsDomain.AIProviderOpenAI, "", &ai.OpenAIWhatsAppIntentParser{}},
		{integrationsDomain.AIProviderAnthropic, "", &ai.AnthropicWhatsAppIntentParser{}},
		{integrationsDomain.AIProviderGemini, "", &ai.GeminiWhatsAppIntentParser{}},
		{integrationsDomain.AIProviderOllama, "http://localhost:11434", &ai.OllamaWhatsAppIntentParser{}},
		{integrationsDomain.AIProviderNVIDIA, "", &ai.NVIDIAWhatsAppIntentParser{}},
	}
	for _, tc := range cases {
		t.Run(string(tc.provider), func(t *testing.T) {
			repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) {
				return &integrationsDomain.AIConnection{Provider: tc.provider, BaseURL: tc.baseURL}, nil
			})
			resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

			parser, err := resolver.Resolve(context.Background())

			require.NoError(t, err)
			require.NotNil(t, parser)
			assert.IsType(t, tc.wantType, parser)
		})
	}
}

func TestWhatsAppIntentParserResolver_geminiWiresFallbackModelsFromConnection(t *testing.T) {
	var calledModels []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledModels = append(calledModels, r.URL.Path)
		if strings.Contains(r.URL.Path, "primary-model") {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": 429}})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"role": "model", "parts": []map[string]any{{"text": `{"command":"STOCK","product_query":"flocao"}`}}}, "finishReason": "STOP"},
			},
		})
	}))
	defer server.Close()

	encryptedKey, err := crypto.Encrypt("test-secret", "test-api-key")
	require.NoError(t, err)
	repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{
			Provider: integrationsDomain.AIProviderGemini, BaseURL: server.URL, APIKeyEncrypted: encryptedKey,
			Model: "primary-model", FallbackModels: "fallback-model",
		}, nil
	})
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	parser, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	intent, err := parser.ParseIntent(context.Background(), "tem flocao?")
	require.NoError(t, err)
	assert.Equal(t, "flocao", intent.ProductQuery)
	require.Len(t, calledModels, 2)
	assert.Contains(t, calledModels[1], "fallback-model")
}

func TestWhatsAppIntentParserResolver_propagatesRepoError(t *testing.T) {
	repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return nil, errors.New("db down")
	})
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}

func TestWhatsAppIntentParserResolver_propagatesUnsupportedProvider(t *testing.T) {
	repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProvider("deepseek")}, nil
	})
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}

func TestWhatsAppIntentParserResolver_wiresNVIDIAFallbackModels(t *testing.T) {
	var models []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		models = append(models, body["model"].(string))
		if len(models) == 1 {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"message": "rate limited"}})
			return
		}
		writeJSON(w, http.StatusOK, chatCompletion(`{"command":"STOCK","product_query":"x"}`))
	}))
	defer server.Close()

	repo := repoOf(func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{
			Provider: integrationsDomain.AIProviderNVIDIA, BaseURL: server.URL, FallbackModels: "meta/llama-3.1-8b-instruct",
		}, nil
	})

	parser, err := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret").Resolve(context.Background())
	require.NoError(t, err)
	_, err = parser.ParseIntent(context.Background(), "tem x?")

	require.NoError(t, err)
	assert.Equal(t, []string{"meta/llama-3.3-70b-instruct", "meta/llama-3.1-8b-instruct"}, models)
}
