package integrations_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)


func connRepoWith(conn *integrationsDomain.Connection) *mocks.ConnectionRepo {
	return &mocks.ConnectionRepo{
		FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn:   func(context.Context, *integrationsDomain.Connection) error { return nil },
	}
}

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func noSecrets(t *testing.T, e eventrecordtest.Event, secrets ...string) {
	t.Helper()
	raw, err := json.Marshal(e.Payload)
	require.NoError(t, err)
	for _, s := range secrets {
		assert.NotContains(t, string(raw), s)
	}
}

func TestDisconnectConnection_RecordsDisconnected(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce})

	require.NoError(t, appintegrations.NewDisconnectConnectionUseCase(newMockGormDB(t), repo, outbox).Execute(context.Background(), "conn-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.connection.disconnected", e.Subject)
	assert.Equal(t, "conn-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "woocommerce"}, e.Data())
}

// COM-15: disconnect keeps the row (deleting it cascaded into sales
// channels, listings and schedules) and only wipes the credentials.
func TestDisconnectConnection_KeepsTheRowAndWipesCredentials(t *testing.T) {
	var saved *integrationsDomain.Connection
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce,
		StoreURL: "https://shop.example", ConsumerKey: "ck", ConsumerSecretEncrypted: "enc", Status: integrationsDomain.StatusConnected})
	repo.UpdateFn = func(_ context.Context, c *integrationsDomain.Connection) error { saved = c; return nil }

	require.NoError(t, appintegrations.NewDisconnectConnectionUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).Execute(context.Background(), "conn-1"))

	require.NotNil(t, saved)
	assert.Equal(t, "conn-1", saved.ID)
	assert.Equal(t, integrationsDomain.StatusDisconnected, saved.Status)
	assert.Empty(t, saved.ConsumerSecretEncrypted)
	assert.Equal(t, integrationsDomain.StoreURL("https://shop.example"), saved.StoreURL)
}

func TestUpdateConnectionSettings_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderBol, DefaultCurrency: "EUR"})
	code := "24uurs-23"

	_, err := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, outbox).Execute(context.Background(),
		appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "eur", BolDeliveryCode: &code})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.connection.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"bol_delivery_code": {nil, "24uurs-23"}}, e.Changes())
	assert.Equal(t, map[string]any{"provider": "bol"}, e.Data())
}

func TestSetWhatsAppTestMode_RecordsChanges(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp})

	_, err := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, outbox).Execute(context.Background(), "conn-1", true, "31612345678")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.connection.updated", e.Subject)
	assert.Equal(t, map[string][2]any{
		"whatsapp_test_mode_enabled": {false, true}, "whatsapp_test_phone": {nil, "31612345678"},
	}, e.Changes())
}

func TestSetWhatsAppAIParsing_RecordsChanges(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp})
	ai := &mocks.AIConnectionRepo{GetFn: func(context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{ID: "ai-1"}, nil
	}}

	_, err := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), repo, ai, outbox).Execute(context.Background(), "conn-1", true)

	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"whatsapp_ai_parsing_enabled": {false, true}}, onlyEvent(t, outbox).Changes())
}

func TestUpdateWhatsAppCredentials_RecordsThatTheKeyChangedNotTheKey(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		StoreURL: "https://old.example.com", ConsumerKey: "shop"})

	_, err := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, testEncryptionSecret, outbox).Execute(context.Background(), "conn-1",
		appintegrations.UpdateWhatsAppCredentialsInput{BaseURL: "https://evo.example.com", InstanceName: "shop", APIKey: "super-secret-key"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, map[string][2]any{
		"base_url": {"https://old.example.com", "https://evo.example.com"}, "api_key": {nil, "changed"},
	}, e.Changes())
	noSecrets(t, e, "super-secret-key")
}

func TestSetWebhookSecret_RecordsWithoutTheSecret(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce})

	out, err := appintegrations.NewSetWebhookSecretUseCase(newMockGormDB(t), repo, testEncryptionSecret, outbox).Execute(context.Background(), "conn-1", "")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.connection.webhook_secret_set", e.Subject)
	assert.Equal(t, map[string]any{"provider": "woocommerce"}, e.Data())
	noSecrets(t, e, out.PlaintextSecret)
}

func TestTestConnection_RecordsResult(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce,
		Status: integrationsDomain.StatusConnected, ConsumerSecretEncrypted: "not-decryptable"})

	_, err := appintegrations.NewTestConnectionUseCase(newMockGormDB(t), repo, &mocks.WooClientFactory{}, testEncryptionSecret, outbox).Execute(context.Background(), "conn-1")

	require.Error(t, err, "undecryptable secret fails the test")
	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.connection.tested", e.Subject)
	assert.Equal(t, map[string]any{"provider": "woocommerce", "succeeded": false}, e.Data())
}

func TestConnectAIProvider_RecordsWithoutTheKey(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	ai := &mocks.AIConnectionRepo{UpsertFn: func(_ context.Context, c *integrationsDomain.AIConnection) error {
		c.ID = "ai-1"
		return nil
	}}

	_, err := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), ai, testEncryptionSecret, outbox).Execute(withTenantCtx("t-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderOpenAI, APIKey: "sk-secret", Model: "gpt-x",
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.ai_connection.connected", e.Subject)
	assert.Equal(t, "ai-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "openai", "model": "gpt-x", "base_url": "", "fallback_models": ""}, e.Data())
	noSecrets(t, e, "sk-secret")
}

func TestDisconnectAIConnection_RecordsDisconnected(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	ai := &mocks.AIConnectionRepo{
		GetFn:    func(context.Context) (*integrationsDomain.AIConnection, error) { return &integrationsDomain.AIConnection{ID: "ai-1"}, nil },
		DeleteFn: func(context.Context) error { return nil },
	}

	require.NoError(t, appintegrations.NewDisconnectAIConnectionUseCase(newMockGormDB(t), ai, outbox).Execute(context.Background()))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "integrations.ai_connection.disconnected", e.Subject)
	assert.Equal(t, "ai-1", e.Payload["entity_id"])
}

// Already disconnected: nothing changed, nothing recorded.
func TestDisconnectAIConnection_NothingToDisconnectRecordsNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	ai := &mocks.AIConnectionRepo{
		GetFn:    func(context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil },
		DeleteFn: func(context.Context) error { return nil },
	}

	require.NoError(t, appintegrations.NewDisconnectAIConnectionUseCase(newMockGormDB(t), ai, outbox).Execute(context.Background()))

	assert.Empty(t, outbox.Events)
}

func TestIntegrationUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	conn := func() *mocks.ConnectionRepo {
		return connRepoWith(&integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp})
	}

	assert.ErrorIs(t, appintegrations.NewDisconnectConnectionUseCase(newMockGormDB(t), conn(), failing).Execute(context.Background(), "conn-1"), boom)
	_, err := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), conn(), failing).Execute(context.Background(),
		appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "USD"})
	assert.ErrorIs(t, err, boom)
	_, err = appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), conn(), failing).Execute(context.Background(), "conn-1", false, "")
	assert.ErrorIs(t, err, boom)
	_, err = appintegrations.NewSetWebhookSecretUseCase(newMockGormDB(t), conn(), testEncryptionSecret, failing).Execute(context.Background(), "conn-1", "s")
	assert.ErrorIs(t, err, boom)
}
