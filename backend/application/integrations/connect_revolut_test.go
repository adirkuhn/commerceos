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

func TestConnectRevolut_createsConnectionAndEnqueuesEvent(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-revolut-1"
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
	uc := appintegrations.NewConnectRevolutUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret, "https://api.example.com")

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appintegrations.ConnectRevolutInput{APIKey: "revolut-secret"})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderRevolut, created.Provider)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)

	decrypted, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "revolut-secret", decrypted)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)
	require.NotNil(t, result)
	assert.Equal(t, "conn-revolut-1", result.Connection.ID)
	assert.Equal(t, "https://api.example.com/api/v1/webhooks/v1/tenant-1/revolut", result.WebhookURL)
}

func TestConnectRevolut_noTenantInContext_rejects(t *testing.T) {
	uc := appintegrations.NewConnectRevolutUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret, "https://api.example.com")

	_, err := uc.Execute(context.Background(), appintegrations.ConnectRevolutInput{APIKey: "revolut-secret"})

	require.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}
