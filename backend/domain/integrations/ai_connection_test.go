package integrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestAIProvider_Valid(t *testing.T) {
	for _, p := range []integrationsDomain.AIProvider{
		integrationsDomain.AIProviderOpenAI,
		integrationsDomain.AIProviderAnthropic,
		integrationsDomain.AIProviderGemini,
		integrationsDomain.AIProviderOllama,
		integrationsDomain.AIProviderNVIDIA,
	} {
		assert.True(t, p.Valid(), "%s should be valid", p)
	}
	assert.False(t, integrationsDomain.AIProvider("mistral").Valid())
	assert.False(t, integrationsDomain.AIProvider("").Valid())
}

func TestAIProvider_RequiresAPIKey(t *testing.T) {
	assert.True(t, integrationsDomain.AIProviderNVIDIA.RequiresAPIKey(), "NVIDIA's hosted API needs an nvapi- key")
	assert.False(t, integrationsDomain.AIProviderOllama.RequiresAPIKey())
}
