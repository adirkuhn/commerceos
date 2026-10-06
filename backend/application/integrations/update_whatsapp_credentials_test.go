package integrations_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func TestUpdateWhatsAppCredentials_replacesBaseURLInstanceNameAndKey(t *testing.T) {
	conn := &integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		StoreURL: "https://old.example.com", ConsumerKey: "old-instance",
		WhatsAppTestModeEnabled: true, WhatsAppAIParsingEnabled: true,
	}
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
	uc := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, "test-encryption-secret", &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: "https://new.example.com", InstanceName: "new-instance", APIKey: "new-key",
	})

	require.NoError(t, err)
	require.Equal(t, integrationsDomain.StoreURL("https://new.example.com"), result.StoreURL)
	require.Equal(t, "new-instance", result.ConsumerKey)
	plaintext, err := crypto.Decrypt("test-encryption-secret", result.ConsumerSecretEncrypted)
	require.NoError(t, err)
	require.Equal(t, "new-key", plaintext)
	// untouched by this use case
	require.True(t, updated.WhatsAppTestModeEnabled)
	require.True(t, updated.WhatsAppAIParsingEnabled)
}

func TestUpdateWhatsAppCredentials_emptyAPIKeyKeepsTheExistingOne(t *testing.T) {
	existingEncrypted, err := crypto.Encrypt("test-encryption-secret", "still-the-old-key")
	require.NoError(t, err)
	conn := &integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		ConsumerSecretEncrypted: existingEncrypted,
	}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn:   func(_ context.Context, _ *integrationsDomain.Connection) error { return nil },
	}
	uc := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, "test-encryption-secret", &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: "https://new.example.com", InstanceName: "new-instance", APIKey: "",
	})

	require.NoError(t, err)
	plaintext, err := crypto.Decrypt("test-encryption-secret", result.ConsumerSecretEncrypted)
	require.NoError(t, err)
	require.Equal(t, "still-the-old-key", plaintext)
}

func TestUpdateWhatsAppCredentials_invalidBaseURL(t *testing.T) {
	repo := &mocks.ConnectionRepo{}
	uc := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, "test-encryption-secret", &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: "not-a-url", InstanceName: "instance", APIKey: "key",
	})

	require.ErrorIs(t, err, integrationsDomain.ErrStoreURLInvalid)
}

func TestUpdateWhatsAppCredentials_rejectsNonWhatsAppConnection(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
	}
	uc := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, "test-encryption-secret", &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: "https://new.example.com", InstanceName: "instance", APIKey: "key",
	})

	require.ErrorIs(t, err, integrationsDomain.ErrNotWhatsAppConnection)
}

func TestUpdateWhatsAppCredentials_connectionNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewUpdateWhatsAppCredentialsUseCase(newMockGormDB(t), repo, "test-encryption-secret", &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", appintegrations.UpdateWhatsAppCredentialsInput{
		BaseURL: "https://new.example.com", InstanceName: "instance", APIKey: "key",
	})

	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
