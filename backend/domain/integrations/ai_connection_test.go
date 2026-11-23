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

func conns(ids ...string) []integrationsDomain.AIConnection {
	out := make([]integrationsDomain.AIConnection, len(ids))
	for i, id := range ids {
		out[i] = integrationsDomain.AIConnection{ID: id, Position: i}
	}
	return out
}

func ids(cs []integrationsDomain.AIConnection) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

func TestNextAIPosition_appendsAfterTheLast(t *testing.T) {
	assert.Equal(t, 0, integrationsDomain.NextAIPosition(nil))
	assert.Equal(t, 8, integrationsDomain.NextAIPosition([]integrationsDomain.AIConnection{{Position: 2}, {Position: 7}}))
}

func TestReorderAIConnections_setsPositionsInTheGivenOrder(t *testing.T) {
	out, err := integrationsDomain.ReorderAIConnections(conns("a", "b", "c"), []string{"c", "a", "b"})

	assert.NoError(t, err)
	assert.Equal(t, []string{"c", "a", "b"}, ids(out))
	for i, c := range out {
		assert.Equal(t, i, c.Position)
	}
}

func TestReorderAIConnections_mustNameEveryConnectionOnce(t *testing.T) {
	for name, order := range map[string][]string{
		"missing one": {"a", "b"},
		"unknown id":  {"a", "b", "x"},
		"duplicate":   {"a", "a", "b"},
		"extra":       {"a", "b", "c", "a"},
	} {
		_, err := integrationsDomain.ReorderAIConnections(conns("a", "b", "c"), order)
		assert.ErrorIs(t, err, integrationsDomain.ErrInvalidAIOrder, name)
	}
}

func TestCompactAIPositions_closesGapsKeepingOrder(t *testing.T) {
	in := []integrationsDomain.AIConnection{{ID: "b", Position: 4}, {ID: "a", Position: 1}, {ID: "c", Position: 9}}

	out := integrationsDomain.CompactAIPositions(in)

	assert.Equal(t, []string{"a", "b", "c"}, ids(out))
	assert.Equal(t, []int{0, 1, 2}, []int{out[0].Position, out[1].Position, out[2].Position})
}

func TestAIConnection_validate(t *testing.T) {
	ok := integrationsDomain.AIConnection{Provider: integrationsDomain.AIProviderNVIDIA, APIKeyEncrypted: "enc"}
	assert.NoError(t, ok.Validate())

	bad := ok
	bad.Provider = "mistral"
	assert.ErrorIs(t, bad.Validate(), integrationsDomain.ErrProviderNotSupported)

	noKey := ok
	noKey.APIKeyEncrypted = ""
	assert.ErrorIs(t, noKey.Validate(), integrationsDomain.ErrAPIKeyRequired)

	ollama := integrationsDomain.AIConnection{Provider: integrationsDomain.AIProviderOllama}
	assert.ErrorIs(t, ollama.Validate(), integrationsDomain.ErrBaseURLRequired)
	ollama.BaseURL = "http://localhost:11434"
	assert.NoError(t, ollama.Validate())

	long := ok
	long.Label = string(make([]rune, integrationsDomain.MaxAILabelLength+1))
	assert.ErrorIs(t, long.Validate(), integrationsDomain.ErrAILabelTooLong)
}
