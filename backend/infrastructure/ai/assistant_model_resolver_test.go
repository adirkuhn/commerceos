package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	"github.com/adirkuhn/commerceos/domain/assistant"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func assistantResolverFor(t *testing.T, conn *integrationsDomain.AIConnection) *ai.AssistantModelResolver {
	t.Helper()
	repo := &mocks.AIConnectionRepo{GetFn: func(context.Context) (*integrationsDomain.AIConnection, error) { return conn, nil }}
	return ai.NewAssistantModelResolver(newMockGormDB(t), repo, "test-secret")
}

func TestAssistantModelResolver_nilWhenNotConfigured(t *testing.T) {
	model, err := assistantResolverFor(t, nil).Resolve(context.Background())

	require.NoError(t, err)
	assert.Nil(t, model)
}

func TestAssistantModelResolver_buildsNVIDIA(t *testing.T) {
	model, err := assistantResolverFor(t, &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProviderNVIDIA}).Resolve(context.Background())

	require.NoError(t, err)
	assert.IsType(t, &ai.NVIDIAChatModel{}, model)
}

func TestAssistantModelResolver_buildsEveryProvider(t *testing.T) {
	cases := []struct {
		provider integrationsDomain.AIProvider
		want     any
	}{
		{integrationsDomain.AIProviderNVIDIA, &ai.NVIDIAChatModel{}},
		{integrationsDomain.AIProviderOpenAI, &ai.OpenAIChatModel{}},
		{integrationsDomain.AIProviderAnthropic, &ai.AnthropicChatModel{}},
		{integrationsDomain.AIProviderGemini, &ai.GeminiChatModel{}},
		{integrationsDomain.AIProviderOllama, &ai.OllamaChatModel{}},
	}
	for _, tc := range cases {
		t.Run(string(tc.provider), func(t *testing.T) {
			model, err := assistantResolverFor(t, &integrationsDomain.AIConnection{Provider: tc.provider, BaseURL: "http://localhost:11434"}).
				Resolve(context.Background())

			require.NoError(t, err)
			assert.IsType(t, tc.want, model)
		})
	}
}

func TestAssistantModelResolver_unknownProviderIsUnsupported(t *testing.T) {
	_, err := assistantResolverFor(t, &integrationsDomain.AIConnection{Provider: "mistral"}).Resolve(context.Background())

	assert.ErrorIs(t, err, assistant.ErrProviderUnsupported)
}

func TestAssistantModelResolver_passesTheDecryptedKeyAndModel(t *testing.T) {
	var auth, model string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		auth, model = r.Header.Get("Authorization"), body["model"].(string)
		writeJSON(w, http.StatusOK, chatCompletion("ok"))
	}))
	defer server.Close()
	key, err := crypto.Encrypt("test-secret", "sk-real")
	require.NoError(t, err)

	m, err := assistantResolverFor(t, &integrationsDomain.AIConnection{
		Provider: integrationsDomain.AIProviderOpenAI, APIKeyEncrypted: key, BaseURL: server.URL, Model: "gpt-5-mini",
	}).Resolve(context.Background())
	require.NoError(t, err)
	_, err = m.Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "hi"}}})

	require.NoError(t, err)
	assert.Equal(t, "Bearer sk-real", auth)
	assert.Equal(t, "gpt-5-mini", model)
}
