package integrations_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestConnectMyParcel_createsConnectionDefaultSettingsAndEnqueuesEvent(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-myparcel-1"
			created = c
			return nil
		},
	}
	var upsertedSettings *shippingDomain.ShippingConnectionSettings
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		UpsertFn: func(_ context.Context, s *shippingDomain.ShippingConnectionSettings) error {
			upsertedSettings = s
			return nil
		},
	}
	var enqueuedSubject string
	outboxRepo := &mocks.OutboxRepo{
		EnqueueFn: func(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
			enqueuedSubject = subject
			return nil
		},
	}
	uc := appintegrations.NewConnectMyParcelUseCase(newMockGormDB(t), connRepo, settingsRepo, outboxRepo, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	conn, err := uc.Execute(ctx, appintegrations.ConnectMyParcelInput{APIKey: "api-key-1"})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderMyParcel, created.Provider)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)
	assert.Empty(t, created.ConsumerKey, "MyParcel has one credential -- ConsumerKey stays unused, the API key lives in ConsumerSecretEncrypted")

	decrypted, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "api-key-1", decrypted)

	require.NotNil(t, upsertedSettings, "connecting must seed a default settings row so a later carrier-override upsert never hits the FK unexpectedly")
	assert.Equal(t, "conn-myparcel-1", upsertedSettings.ConnectionID)
	assert.Equal(t, shippingDomain.CarrierPostNL, upsertedSettings.DefaultCarrier)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)
	require.NotNil(t, conn)
	assert.Equal(t, "conn-myparcel-1", conn.ID)
}

func TestConnectMyParcel_noTenantContext_returnsError(t *testing.T) {
	uc := appintegrations.NewConnectMyParcelUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), appintegrations.ConnectMyParcelInput{APIKey: "k"})

	require.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}

func TestConnectMyParcel_reconnectingLeavesTheCarrierDefaults(t *testing.T) {
	var updated *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		FindByStoreFn: func(context.Context, integrationsDomain.Provider, integrationsDomain.StoreURL) (*integrationsDomain.Connection, error) {
			return &integrationsDomain.Connection{ID: "conn-myparcel-1", Provider: integrationsDomain.ProviderMyParcel, Status: integrationsDomain.StatusConnected}, nil
		},
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	var upsertedSettings *shippingDomain.ShippingConnectionSettings
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		UpsertFn: func(_ context.Context, s *shippingDomain.ShippingConnectionSettings) error {
			upsertedSettings = s
			return nil
		},
	}
	var enqueuedSubject string
	outboxRepo := &mocks.OutboxRepo{
		EnqueueFn: func(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
			enqueuedSubject = subject
			return nil
		},
	}
	uc := appintegrations.NewConnectMyParcelUseCase(newMockGormDB(t), connRepo, settingsRepo, outboxRepo, testEncryptionSecret)

	conn, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appintegrations.ConnectMyParcelInput{APIKey: "api-key-2"})

	require.NoError(t, err)
	assert.Equal(t, "conn-myparcel-1", conn.ID)
	require.NotNil(t, updated)
	assert.Nil(t, upsertedSettings, "carrier defaults kept")
	assert.NotEmpty(t, enqueuedSubject, "the connect is still recorded")
}
