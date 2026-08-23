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

const testEncryptionSecret = "test-encryption-secret-32bytes!"

func TestConnectBol_createsConnectionAndEnqueuesEvent(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-bol-1"
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
	uc := appintegrations.NewConnectBolUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	conn, err := uc.Execute(ctx, appintegrations.ConnectBolInput{
		ClientID: "client-1", ClientSecret: "secret-1", EconomicOperatorID: "op-1",
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderBol, created.Provider)
	assert.Equal(t, "client-1", created.ConsumerKey)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)
	require.NotNil(t, created.BolEconomicOperatorID)
	assert.Equal(t, "op-1", *created.BolEconomicOperatorID)

	decrypted, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "secret-1", decrypted)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)

	require.NotNil(t, conn)
	assert.Equal(t, "conn-bol-1", conn.ID)
}
