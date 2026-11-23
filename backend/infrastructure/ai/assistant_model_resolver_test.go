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
	repo := repoOf(func(context.Context) (*integrationsDomain.AIConnection, error) { return conn, nil })
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

func listRepo(conns ...integrationsDomain.AIConnection) *mocks.AIConnectionRepo {
	return &mocks.AIConnectionRepo{ListFn: func(context.Context) ([]integrationsDomain.AIConnection, error) { return conns, nil }}
}

func TestAssistantModelResolver_routesToTheNextConnectionOnAnyFailure(t *testing.T) {
	var hits []string
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits = append(hits, "first")
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer failing.Close()
	working := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits = append(hits, "second")
		writeJSON(w, http.StatusOK, chatCompletion("from the second"))
	}))
	defer working.Close()
	disabled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits = append(hits, "disabled")
		writeJSON(w, http.StatusOK, chatCompletion("never"))
	}))
	defer disabled.Close()

	repo := listRepo(
		integrationsDomain.AIConnection{Position: 0, Enabled: false, Provider: integrationsDomain.AIProviderNVIDIA, BaseURL: disabled.URL},
		integrationsDomain.AIConnection{Position: 1, Enabled: true, Provider: integrationsDomain.AIProviderNVIDIA, BaseURL: failing.URL, Label: "key 1"},
		integrationsDomain.AIConnection{Position: 2, Enabled: true, Provider: integrationsDomain.AIProviderOpenAI, BaseURL: working.URL},
	)
	model, err := ai.NewAssistantModelResolver(newMockGormDB(t), repo, "test-secret").Resolve(context.Background())
	require.NoError(t, err)

	reply, err := model.Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "hi"}}})

	require.NoError(t, err)
	assert.Equal(t, "from the second", reply.Content)
	assert.Equal(t, []string{"first", "second"}, hits, "a disabled connection is skipped; a bad key moves on")
}

func TestAssistantModelResolver_allFailingWithOneBusyKeepsTheRequestQueued(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer bad.Close()
	busy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]any{"message": "quota"}})
	}))
	defer busy.Close()
	repo := listRepo(
		integrationsDomain.AIConnection{Position: 0, Enabled: true, Provider: integrationsDomain.AIProviderOpenAI, BaseURL: bad.URL},
		integrationsDomain.AIConnection{Position: 1, Enabled: true, Provider: integrationsDomain.AIProviderNVIDIA, BaseURL: busy.URL},
	)
	model, err := ai.NewAssistantModelResolver(newMockGormDB(t), repo, "test-secret").Resolve(context.Background())
	require.NoError(t, err)

	_, err = model.Next(context.Background(), assistant.ChatTurn{Messages: []assistant.Message{{Role: assistant.RoleUser, Content: "hi"}}})

	require.Error(t, err)
	assert.ErrorIs(t, err, assistant.ErrModelBusy)
	assert.Contains(t, err.Error(), "invalid api key")
	assert.Contains(t, err.Error(), "quota")
}

func TestAssistantModelResolver_onlyDisabledConnectionsMeansNotConfigured(t *testing.T) {
	repo := listRepo(integrationsDomain.AIConnection{Enabled: false, Provider: integrationsDomain.AIProviderNVIDIA})

	model, err := ai.NewAssistantModelResolver(newMockGormDB(t), repo, "test-secret").Resolve(context.Background())

	require.NoError(t, err)
	assert.Nil(t, model)
}
