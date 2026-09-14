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

func TestConnectEmail_createsConnectionAndEnqueuesEvent(t *testing.T) {
	var created *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			c.ID = "conn-email-1"
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
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appintegrations.ConnectEmailInput{
		Host: "smtp.example.com", Port: "587", Username: "apikey", Password: "s3cr3t", From: "orders@mystore.com",
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, integrationsDomain.ProviderEmail, created.Provider)
	assert.Equal(t, integrationsDomain.StatusConnected, created.Status)
	assert.Equal(t, "smtp.example.com", created.StoreURL.String())
	assert.Equal(t, "apikey", created.ConsumerKey)
	require.NotNil(t, created.SMTPPort)
	assert.Equal(t, "587", *created.SMTPPort)
	require.NotNil(t, created.EmailFrom)
	assert.Equal(t, "orders@mystore.com", *created.EmailFrom)

	decrypted, err := crypto.Decrypt(testEncryptionSecret, created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", decrypted)

	assert.Equal(t, "integrations.connection.connected", enqueuedSubject)
	require.NotNil(t, result)
	assert.Equal(t, "conn-email-1", result.ID)
}

func TestConnectEmail_allowsBlankUsernamePassword(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error { c.ID = "conn-email-2"; return nil }}
	outboxRepo := &mocks.OutboxRepo{EnqueueFn: func(context.Context, string, string, []byte, time.Time) error { return nil }}
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), connRepo, outboxRepo, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appintegrations.ConnectEmailInput{Host: "smtp.example.com", Port: "25", From: "orders@mystore.com"})

	require.NoError(t, err)
}

func TestConnectEmail_rejectsMissingHost(t *testing.T) {
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appintegrations.ConnectEmailInput{Port: "587", From: "orders@mystore.com"})

	require.ErrorIs(t, err, integrationsDomain.ErrEmailHostRequired)
}

func TestConnectEmail_rejectsMissingPort(t *testing.T) {
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appintegrations.ConnectEmailInput{Host: "smtp.example.com", From: "orders@mystore.com"})

	require.ErrorIs(t, err, integrationsDomain.ErrEmailPortRequired)
}

func TestConnectEmail_rejectsMissingFrom(t *testing.T) {
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appintegrations.ConnectEmailInput{Host: "smtp.example.com", Port: "587"})

	require.ErrorIs(t, err, integrationsDomain.ErrEmailFromRequired)
}

func TestConnectEmail_noTenantInContext_rejects(t *testing.T) {
	uc := appintegrations.NewConnectEmailUseCase(newMockGormDB(t), &mocks.ConnectionRepo{}, &mocks.OutboxRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), appintegrations.ConnectEmailInput{Host: "smtp.example.com", Port: "587", From: "orders@mystore.com"})

	require.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}
