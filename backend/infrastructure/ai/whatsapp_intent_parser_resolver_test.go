package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
)

func TestWhatsAppIntentParserResolver_returnsNilWhenNotConfigured(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil }}
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
	}
	for _, tc := range cases {
		t.Run(string(tc.provider), func(t *testing.T) {
			repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
				return &integrationsDomain.AIConnection{Provider: tc.provider, BaseURL: tc.baseURL}, nil
			}}
			resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

			parser, err := resolver.Resolve(context.Background())

			require.NoError(t, err)
			require.NotNil(t, parser)
			assert.IsType(t, tc.wantType, parser)
		})
	}
}

func TestWhatsAppIntentParserResolver_propagatesRepoError(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return nil, errors.New("db down")
	}}
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}

func TestWhatsAppIntentParserResolver_propagatesUnsupportedProvider(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProvider("deepseek")}, nil
	}}
	resolver := ai.NewWhatsAppIntentParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}
