package integrations_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func withTenantCtx(tenantID string) context.Context {
	return tenant.WithTenantID(context.Background(), tenantID)
}

// memAIConns is an in-memory AIConnectionRepository.
type memAIConns struct {
	rows []integrationsDomain.AIConnection
	err  error
}

func (m *memAIConns) List(context.Context) ([]integrationsDomain.AIConnection, error) {
	out := append([]integrationsDomain.AIConnection(nil), m.rows...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, m.err
}

func (m *memAIConns) Get(_ context.Context, id string) (*integrationsDomain.AIConnection, error) {
	for _, c := range m.rows {
		if c.ID == id {
			cp := c
			return &cp, nil
		}
	}
	return nil, integrationsDomain.ErrAIConnectionNotFound
}

func (m *memAIConns) Create(_ context.Context, c *integrationsDomain.AIConnection) error {
	if m.err != nil {
		return m.err
	}
	c.ID = "ai-new"
	m.rows = append(m.rows, *c)
	return nil
}

func (m *memAIConns) Update(_ context.Context, c *integrationsDomain.AIConnection) error {
	for i := range m.rows {
		if m.rows[i].ID == c.ID {
			m.rows[i] = *c
			return nil
		}
	}
	return integrationsDomain.ErrAIConnectionNotFound
}

func (m *memAIConns) Delete(_ context.Context, id string) error {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return nil
		}
	}
	return integrationsDomain.ErrAIConnectionNotFound
}

func (m *memAIConns) byID(id string) integrationsDomain.AIConnection {
	c, _ := m.Get(context.Background(), id)
	return *c
}

func threeConns() *memAIConns {
	return &memAIConns{rows: []integrationsDomain.AIConnection{
		{ID: "a", Position: 0, Provider: integrationsDomain.AIProviderNVIDIA, APIKeyEncrypted: "enc-a", Enabled: true},
		{ID: "b", Position: 1, Provider: integrationsDomain.AIProviderGemini, APIKeyEncrypted: "enc-b", Enabled: true},
		{ID: "c", Position: 2, Provider: integrationsDomain.AIProviderOllama, BaseURL: "http://ollama:11434", Enabled: true},
	}}
}

func TestCreateAIConnection_appendsEncryptedAndRecordsWithoutTheKey(t *testing.T) {
	repo, outbox := threeConns(), &eventrecordtest.Outbox{}

	c, err := appintegrations.NewCreateAIConnectionUseCase(newMockGormDB(t), repo, testEncryptionSecret, outbox).
		Execute(withTenantCtx("t-1"), appintegrations.AIConnectionInput{
			Provider: integrationsDomain.AIProviderNVIDIA, APIKey: "nvapi-secret", Model: "meta/llama-3.3-70b-instruct", Label: "NVIDIA key 2",
		})

	require.NoError(t, err)
	assert.Equal(t, 3, c.Position, "added at the end of the list")
	assert.True(t, c.Enabled)
	assert.Equal(t, "t-1", c.TenantID)
	assert.Equal(t, "NVIDIA key 2", c.Label)
	assert.Equal(t, integrationsDomain.StatusConnected, c.Status)
	plain, err := crypto.Decrypt(testEncryptionSecret, repo.byID("ai-new").APIKeyEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "nvapi-secret", plain)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "integrations.ai_connection.connected", e.Subject)
	assert.NotContains(t, string(mustJSON(t, e.Payload)), "nvapi-secret")
	assert.Equal(t, float64(3), e.Data()["position"])
}

func TestCreateAIConnection_validates(t *testing.T) {
	cases := map[string]struct {
		in   appintegrations.AIConnectionInput
		want error
	}{
		"unknown provider": {appintegrations.AIConnectionInput{Provider: "mistral", APIKey: "k"}, integrationsDomain.ErrProviderNotSupported},
		"missing key":      {appintegrations.AIConnectionInput{Provider: integrationsDomain.AIProviderOpenAI}, integrationsDomain.ErrAPIKeyRequired},
		"ollama no url":    {appintegrations.AIConnectionInput{Provider: integrationsDomain.AIProviderOllama}, integrationsDomain.ErrBaseURLRequired},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &memAIConns{}
			_, err := appintegrations.NewCreateAIConnectionUseCase(newMockGormDB(t), repo, testEncryptionSecret, &eventrecordtest.Outbox{}).
				Execute(withTenantCtx("t-1"), tc.in)

			assert.ErrorIs(t, err, tc.want)
			assert.Empty(t, repo.rows)
		})
	}
}

func TestCreateAIConnection_needsTenant(t *testing.T) {
	_, err := appintegrations.NewCreateAIConnectionUseCase(newMockGormDB(t), &memAIConns{}, testEncryptionSecret, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appintegrations.AIConnectionInput{Provider: integrationsDomain.AIProviderOpenAI, APIKey: "k"})

	assert.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}

func TestListAIConnections_inRoutingOrder(t *testing.T) {
	repo := threeConns()

	out, err := appintegrations.NewListAIConnectionsUseCase(newMockGormDB(t), repo).Execute(withTenantCtx("t-1"))

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, []string{out[0].ID, out[1].ID, out[2].ID})
}

func ptr[T any](v T) *T { return &v }

func TestUpdateAIConnection_changesOnlyWhatIsGivenAndKeepsTheKey(t *testing.T) {
	repo, outbox := threeConns(), &eventrecordtest.Outbox{}

	c, err := appintegrations.NewUpdateAIConnectionUseCase(newMockGormDB(t), repo, testEncryptionSecret, outbox).
		Execute(withTenantCtx("t-1"), "b", appintegrations.AIConnectionPatch{Model: ptr("gemini-3.5-flash-lite"), Enabled: ptr(false), APIKey: ptr("")})

	require.NoError(t, err)
	assert.Equal(t, "gemini-3.5-flash-lite", c.Model)
	assert.False(t, c.Enabled)
	assert.Equal(t, "enc-b", repo.byID("b").APIKeyEncrypted, "an empty key keeps the stored one")
	assert.Equal(t, 1, repo.byID("b").Position)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "integrations.ai_connection.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"model": {"", "gemini-3.5-flash-lite"}, "enabled": {true, false}}, e.Changes())
}

func TestUpdateAIConnection_newKeyIsEncryptedAndRecordedOnlyAsChanged(t *testing.T) {
	repo, outbox := threeConns(), &eventrecordtest.Outbox{}

	_, err := appintegrations.NewUpdateAIConnectionUseCase(newMockGormDB(t), repo, testEncryptionSecret, outbox).
		Execute(withTenantCtx("t-1"), "a", appintegrations.AIConnectionPatch{APIKey: ptr("nvapi-new")})

	require.NoError(t, err)
	plain, err := crypto.Decrypt(testEncryptionSecret, repo.byID("a").APIKeyEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "nvapi-new", plain)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.NotContains(t, string(mustJSON(t, e.Payload)), "nvapi-new")
	assert.Equal(t, true, e.Data()["api_key_changed"])
}

func TestUpdateAIConnection_switchingToAProviderThatNeedsAKeyWithoutOne(t *testing.T) {
	repo := threeConns()

	_, err := appintegrations.NewUpdateAIConnectionUseCase(newMockGormDB(t), repo, testEncryptionSecret, &eventrecordtest.Outbox{}).
		Execute(withTenantCtx("t-1"), "c", appintegrations.AIConnectionPatch{Provider: ptr(integrationsDomain.AIProviderOpenAI)})

	assert.ErrorIs(t, err, integrationsDomain.ErrAPIKeyRequired)
	assert.Equal(t, integrationsDomain.AIProviderOllama, repo.byID("c").Provider)
}

func TestUpdateAIConnection_notFound(t *testing.T) {
	_, err := appintegrations.NewUpdateAIConnectionUseCase(newMockGormDB(t), threeConns(), testEncryptionSecret, &eventrecordtest.Outbox{}).
		Execute(withTenantCtx("t-1"), "zzz", appintegrations.AIConnectionPatch{Label: ptr("x")})

	assert.ErrorIs(t, err, integrationsDomain.ErrAIConnectionNotFound)
}

func TestDeleteAIConnection_closesTheGap(t *testing.T) {
	repo, outbox := threeConns(), &eventrecordtest.Outbox{}

	require.NoError(t, appintegrations.NewDeleteAIConnectionUseCase(newMockGormDB(t), repo, outbox).Execute(withTenantCtx("t-1"), "a"))

	require.Len(t, repo.rows, 2)
	assert.Equal(t, 0, repo.byID("b").Position)
	assert.Equal(t, 1, repo.byID("c").Position)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "integrations.ai_connection.disconnected", e.Subject)
	assert.Equal(t, map[string]any{"provider": "nvidia"}, e.Data())
}

func TestDeleteAIConnection_notFound(t *testing.T) {
	err := appintegrations.NewDeleteAIConnectionUseCase(newMockGormDB(t), threeConns(), &eventrecordtest.Outbox{}).Execute(withTenantCtx("t-1"), "zzz")

	assert.ErrorIs(t, err, integrationsDomain.ErrAIConnectionNotFound)
}

func TestReorderAIConnections(t *testing.T) {
	repo, outbox := threeConns(), &eventrecordtest.Outbox{}

	out, err := appintegrations.NewReorderAIConnectionsUseCase(newMockGormDB(t), repo, outbox).Execute(withTenantCtx("t-1"), []string{"c", "a", "b"})

	require.NoError(t, err)
	assert.Equal(t, []string{"c", "a", "b"}, []string{out[0].ID, out[1].ID, out[2].ID})
	assert.Equal(t, 0, repo.byID("c").Position)
	assert.Equal(t, 2, repo.byID("b").Position)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "integrations.ai_connection.reordered", e.Subject)
	assert.Equal(t, []any{"c", "a", "b"}, e.Data()["order"])
}

func TestReorderAIConnections_mustListEveryOneOnce(t *testing.T) {
	repo := threeConns()

	_, err := appintegrations.NewReorderAIConnectionsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).Execute(withTenantCtx("t-1"), []string{"c", "a"})

	assert.ErrorIs(t, err, integrationsDomain.ErrInvalidAIOrder)
	assert.Equal(t, 0, repo.byID("a").Position)
}

func TestAIConnectionUseCases_repoErrorsAreWrapped(t *testing.T) {
	repo := &memAIConns{err: errors.New("db down")}

	_, err := appintegrations.NewListAIConnectionsUseCase(newMockGormDB(t), repo).Execute(withTenantCtx("t-1"))

	assert.ErrorContains(t, err, "db down")
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
