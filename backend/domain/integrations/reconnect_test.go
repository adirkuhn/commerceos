package integrations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

func TestConnection_Disconnect_keepsTheRowButDropsEveryCredential(t *testing.T) {
	now := time.Date(2026, 11, 3, 7, 43, 0, 0, time.UTC)
	c := integrations.Connection{
		ID: "conn-1", Provider: integrations.ProviderWooCommerce, StoreURL: "https://shop.example",
		ConsumerKey: "ck_1", ConsumerSecretEncrypted: "enc", WebhookSecretEncrypted: "whenc",
		Status: integrations.StatusConnected,
	}

	c.Disconnect(now)

	assert.True(t, c.IsDisconnected())
	assert.Equal(t, "conn-1", c.ID, "same row -- nothing referencing it is touched")
	assert.Equal(t, integrations.StoreURL("https://shop.example"), c.StoreURL, "kept so a reconnect finds it")
	assert.Empty(t, c.ConsumerKey)
	assert.Empty(t, c.ConsumerSecretEncrypted)
	assert.Empty(t, c.WebhookSecretEncrypted)
	assert.Equal(t, now, c.UpdatedAt)
}

type fakeConnectionSaver struct {
	existing *integrations.Connection
	findErr  error
	created  []integrations.Connection
	updated  []integrations.Connection
	asked    []string
}

func (f *fakeConnectionSaver) FindByStore(_ context.Context, p integrations.Provider, u integrations.StoreURL) (*integrations.Connection, error) {
	f.asked = append(f.asked, string(p)+"|"+string(u))
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.existing == nil {
		return nil, integrations.ErrConnectionNotFound
	}
	c := *f.existing
	return &c, nil
}

func (f *fakeConnectionSaver) Create(_ context.Context, c *integrations.Connection) error {
	c.ID = "new-id"
	f.created = append(f.created, *c)
	return nil
}

func (f *fakeConnectionSaver) Update(_ context.Context, c *integrations.Connection) error {
	f.updated = append(f.updated, *c)
	return nil
}

func freshWoo() integrations.Connection {
	return integrations.Connection{
		TenantID: "t1", Provider: integrations.ProviderWooCommerce, StoreURL: "https://shop.example",
		ConsumerKey: "ck_new", ConsumerSecretEncrypted: "enc_new", Status: integrations.StatusConnected,
		CreatedAt: time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC),
	}
}

func TestSaveConnection_reconnectingTheSameStoreReusesTheDisconnectedRow(t *testing.T) {
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeConnectionSaver{existing: &integrations.Connection{
		ID: "07c49b89", Provider: integrations.ProviderWooCommerce, StoreURL: "https://shop.example",
		Status: integrations.StatusDisconnected, CreatedAt: created,
	}}
	c := freshWoo()

	reconnected, err := integrations.SaveConnection(context.Background(), repo, &c)

	require.NoError(t, err)
	assert.True(t, reconnected)
	assert.Empty(t, repo.created, "no new id -- orders, listings and schedules keep pointing at it")
	require.Len(t, repo.updated, 1)
	assert.Equal(t, "07c49b89", repo.updated[0].ID)
	assert.Equal(t, created, repo.updated[0].CreatedAt)
	assert.Equal(t, "ck_new", repo.updated[0].ConsumerKey)
	assert.Equal(t, integrations.StatusConnected, repo.updated[0].Status)
	assert.Equal(t, "07c49b89", c.ID, "caller sees the reused id")
	assert.Equal(t, []string{"woocommerce|https://shop.example"}, repo.asked)
}

func TestSaveConnection_nothingToReuseCreatesANewRow(t *testing.T) {
	repo := &fakeConnectionSaver{}
	c := freshWoo()

	reconnected, err := integrations.SaveConnection(context.Background(), repo, &c)

	require.NoError(t, err)
	assert.False(t, reconnected)
	require.Len(t, repo.created, 1)
	assert.Empty(t, repo.updated)
	assert.Equal(t, "new-id", c.ID)
}

func TestSaveConnection_aLookupFailureIsReturned(t *testing.T) {
	boom := errors.New("db down")
	repo := &fakeConnectionSaver{findErr: boom}
	c := freshWoo()

	_, err := integrations.SaveConnection(context.Background(), repo, &c)

	require.ErrorIs(t, err, boom)
	assert.Empty(t, repo.created)
}

// Woo rotated its keys, or the keys stopped working: connecting the store
// again must replace them on the same row. A second row is impossible
// (UNIQUE tenant+provider+store) and the row is what everything links to.
func TestSaveConnection_aLiveConnectionToTheSameStoreGetsTheNewKeys(t *testing.T) {
	for _, status := range []integrations.Status{integrations.StatusConnected, integrations.StatusError} {
		t.Run(string(status), func(t *testing.T) {
			repo := &fakeConnectionSaver{existing: &integrations.Connection{
				ID: "63dd429a", Provider: integrations.ProviderWooCommerce, StoreURL: "https://shop.example",
				ConsumerKey: "ck_old", ConsumerSecretEncrypted: "enc_old", Status: status,
			}}
			c := freshWoo()

			reconnected, err := integrations.SaveConnection(context.Background(), repo, &c)

			require.NoError(t, err)
			assert.True(t, reconnected)
			assert.Empty(t, repo.created)
			require.Len(t, repo.updated, 1)
			assert.Equal(t, "63dd429a", repo.updated[0].ID)
			assert.Equal(t, "ck_new", repo.updated[0].ConsumerKey)
			assert.Equal(t, "enc_new", repo.updated[0].ConsumerSecretEncrypted)
			assert.Equal(t, integrations.StatusConnected, repo.updated[0].Status)
		})
	}
}

// A connect form only carries credentials. Everything configured on the
// row afterwards must survive, or a key change silently resets it.
func TestSaveConnection_keepsWhatTheConnectFormDoesNotCarry(t *testing.T) {
	warehouse, operator, delivery, port, from, phone := "wh-1", "op-1", "24uurs-21", "587", "a@b.example", "+31600000000"
	repo := &fakeConnectionSaver{existing: &integrations.Connection{
		ID: "c1", Provider: integrations.ProviderWooCommerce, StoreURL: "https://shop.example",
		Status: integrations.StatusConnected, WebhookSecretEncrypted: "wh_enc",
		DefaultCurrency: "EUR", DefaultWarehouseID: &warehouse,
		BolEconomicOperatorID: &operator, BolDeliveryCode: &delivery, BolShippingCostMinorUnits: 495,
		SMTPPort: &port, EmailFrom: &from,
		WhatsAppTestModeEnabled: true, WhatsAppTestPhoneE164: &phone, WhatsAppAIParsingEnabled: true,
	}}
	c := freshWoo()

	_, err := integrations.SaveConnection(context.Background(), repo, &c)

	require.NoError(t, err)
	got := repo.updated[0]
	assert.Equal(t, "wh_enc", got.WebhookSecretEncrypted)
	assert.Equal(t, "EUR", got.DefaultCurrency)
	assert.Equal(t, &warehouse, got.DefaultWarehouseID)
	assert.Equal(t, &operator, got.BolEconomicOperatorID)
	assert.Equal(t, &delivery, got.BolDeliveryCode)
	assert.Equal(t, int64(495), got.BolShippingCostMinorUnits)
	assert.Equal(t, &port, got.SMTPPort)
	assert.Equal(t, &from, got.EmailFrom)
	assert.True(t, got.WhatsAppTestModeEnabled)
	assert.Equal(t, &phone, got.WhatsAppTestPhoneE164)
	assert.True(t, got.WhatsAppAIParsingEnabled)
}

func TestSaveConnection_whatTheConnectFormDoesCarryWins(t *testing.T) {
	oldOperator, newOperator, oldPort, newPort := "op-old", "op-new", "25", "587"
	repo := &fakeConnectionSaver{existing: &integrations.Connection{
		ID: "c1", Provider: integrations.ProviderBol, Status: integrations.StatusConnected,
		WebhookSecretEncrypted: "old", BolEconomicOperatorID: &oldOperator, SMTPPort: &oldPort,
		PostNLCustomerCode: "OLD", PostNLCustomerNumber: "1", PostNLSandbox: true,
	}}
	c := integrations.Connection{
		Provider: integrations.ProviderBol, Status: integrations.StatusConnected, ConsumerKey: "id",
		WebhookSecretEncrypted: "new", BolEconomicOperatorID: &newOperator, SMTPPort: &newPort,
		PostNLCustomerCode: "NEW", PostNLCustomerNumber: "2", PostNLSandbox: false,
	}

	_, err := integrations.SaveConnection(context.Background(), repo, &c)

	require.NoError(t, err)
	got := repo.updated[0]
	assert.Equal(t, "new", got.WebhookSecretEncrypted)
	assert.Equal(t, &newOperator, got.BolEconomicOperatorID)
	assert.Equal(t, &newPort, got.SMTPPort)
	assert.Equal(t, "NEW", got.PostNLCustomerCode)
	assert.Equal(t, "2", got.PostNLCustomerNumber)
	assert.False(t, got.PostNLSandbox, "the form carries sandbox, so false is a choice")
}
