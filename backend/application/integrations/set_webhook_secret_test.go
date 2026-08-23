package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const testWebhookEncryptionSecret = "test-encryption-secret"

func TestSetWebhookSecret_generatesAndEncryptsSecret(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
			require.Equal(t, "conn-1", id)
			return conn, nil
		},
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewSetWebhookSecretUseCase(newMockGormDB(t), repo, testWebhookEncryptionSecret)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Equal(t, "conn-1", result.ConnectionID)
	require.NotEmpty(t, result.PlaintextSecret)
	require.NotNil(t, updated)
	require.NotEmpty(t, updated.WebhookSecretEncrypted)

	decrypted, err := crypto.Decrypt(testWebhookEncryptionSecret, updated.WebhookSecretEncrypted)
	require.NoError(t, err)
	require.Equal(t, result.PlaintextSecret, decrypted)
}

func TestSetWebhookSecret_rotatesExistingSecret(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", WebhookSecretEncrypted: "old-encrypted-value"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewSetWebhookSecretUseCase(newMockGormDB(t), repo, testWebhookEncryptionSecret)

	_, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.NotEqual(t, "old-encrypted-value", updated.WebhookSecretEncrypted)
}

func TestSetWebhookSecret_propagatesNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewSetWebhookSecretUseCase(newMockGormDB(t), repo, testWebhookEncryptionSecret)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
