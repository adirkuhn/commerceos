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
)

func TestSetWhatsAppTestMode_enablesWithTestPhone(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp}
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
	uc := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", true, "31612345678")

	require.NoError(t, err)
	require.True(t, result.WhatsAppTestModeEnabled)
	require.NotNil(t, result.WhatsAppTestPhoneE164)
	require.Equal(t, "31612345678", *result.WhatsAppTestPhoneE164)
	require.True(t, updated.WhatsAppTestModeEnabled)
	require.Equal(t, "31612345678", *updated.WhatsAppTestPhoneE164)
}

func TestSetWhatsAppTestMode_disablingClearsTheStoredNumber(t *testing.T) {
	existingPhone := "31612345678"
	conn := &integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp,
		WhatsAppTestModeEnabled: true, WhatsAppTestPhoneE164: &existingPhone,
	}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "conn-1", false, "")

	require.NoError(t, err)
	require.False(t, result.WhatsAppTestModeEnabled)
	require.Nil(t, result.WhatsAppTestPhoneE164)
	require.Nil(t, updated.WhatsAppTestPhoneE164)
}

func TestSetWhatsAppTestMode_enablingWithNoPhoneIsRejected(t *testing.T) {
	repo := &mocks.ConnectionRepo{}
	uc := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", true, "  ")

	require.ErrorIs(t, err, integrationsDomain.ErrWhatsAppTestPhoneRequired)
}

func TestSetWhatsAppTestMode_rejectsNonWhatsAppConnection(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
	}
	uc := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "conn-1", true, "31612345678")

	require.ErrorIs(t, err, integrationsDomain.ErrNotWhatsAppConnection)
}

func TestSetWhatsAppTestMode_propagatesNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewSetWhatsAppTestModeUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", true, "31612345678")

	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
