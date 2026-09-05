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

const testPublicAPIBaseURL = "https://api.example.com"

func TestConnectWhatsApp_createsConnectionWithWebhookSecretAndReturnsDeliveryURL(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-whatsapp-1"
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
	uc := appintegrations.NewConnectWhatsAppUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret, testPublicAPIBaseURL)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appintegrations.ConnectWhatsAppInput{
		BaseURL: "https://evolution.example.com", InstanceName: "tenant-1-instance", APIKey: "api-key-1",
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderWhatsApp, created.Provider)
	assert.Equal(t, "https://evolution.example.com", created.StoreURL.String())
	assert.Equal(t, "tenant-1-instance", created.ConsumerKey)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)
	assert.NotEmpty(t, created.WebhookSecretEncrypted, "a webhook secret must be generated up front -- there is no separate SetWebhookSecret step for this provider")

	decryptedAPIKey, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "api-key-1", decryptedAPIKey)

	decryptedWebhookSecret, err := crypto.Decrypt(testEncryptionSecret, created.WebhookSecretEncrypted)
	require.NoError(t, err)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)

	require.NotNil(t, result)
	assert.Equal(t, "conn-whatsapp-1", result.Connection.ID)
	expectedURL := testPublicAPIBaseURL + "/api/v1/webhooks/v1/tenant-1/whatsapp/" + decryptedWebhookSecret
	assert.Equal(t, expectedURL, result.WebhookURL)
}

func TestConnectWhatsApp_invalidBaseURL_rejectsBeforeCreatingAnything(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			t.Fatal("must not create a connection for an invalid base URL")
			return nil
		},
	}
	uc := appintegrations.NewConnectWhatsAppUseCase(newMockGormDB(t), connRepo, &mocks.OutboxRepo{}, testEncryptionSecret, testPublicAPIBaseURL)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appintegrations.ConnectWhatsAppInput{BaseURL: "not-a-url", InstanceName: "x", APIKey: "y"})

	require.Error(t, err)
}

func TestConnectWhatsApp_noTenantInContext_rejects(t *testing.T) {
	uc := appintegrations.NewConnectWhatsAppUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret, testPublicAPIBaseURL)

	_, err := uc.Execute(context.Background(), appintegrations.ConnectWhatsAppInput{
		BaseURL: "https://evolution.example.com", InstanceName: "x", APIKey: "y",
	})

	require.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}
