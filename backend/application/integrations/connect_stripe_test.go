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
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestConnectStripe_createsConnectionAndEnqueuesEvent(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-stripe-1"
			created = c
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
	uc := appintegrations.NewConnectStripeUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	conn, err := uc.Execute(ctx, appintegrations.ConnectStripeInput{SecretKey: "sk_test_secret"})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderStripe, created.Provider)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)

	decrypted, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "sk_test_secret", decrypted)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)
	require.NotNil(t, conn)
	assert.Equal(t, "conn-stripe-1", conn.ID)
}

func TestConnectStripe_noTenantInContext_rejects(t *testing.T) {
	uc := appintegrations.NewConnectStripeUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), appintegrations.ConnectStripeInput{SecretKey: "sk_test_secret"})

	require.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}
