package ai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	"github.com/adirkuhn/commerceos/domain/assistant"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
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

func TestAssistantModelResolver_otherProvidersAreNotSupportedYet(t *testing.T) {
	for _, p := range []integrationsDomain.AIProvider{
		integrationsDomain.AIProviderOpenAI, integrationsDomain.AIProviderAnthropic,
		integrationsDomain.AIProviderGemini, integrationsDomain.AIProviderOllama,
	} {
		_, err := assistantResolverFor(t, &integrationsDomain.AIConnection{Provider: p, BaseURL: "http://x"}).Resolve(context.Background())

		assert.ErrorIs(t, err, assistant.ErrProviderUnsupported, string(p))
	}
}
