package integrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestSetWhatsAppAIParsing_enablesWhenAIConnectionConfigured(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp}
	var updated *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
			require.Equal(t, "conn-1", id)
			return conn, nil
		},
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	aiConnRepo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProviderGemini}, nil
	}}
	uc := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), connRepo, aiConnRepo)

	result, err := uc.Execute(context.Background(), "conn-1", true)

	require.NoError(t, err)
	require.True(t, result.WhatsAppAIParsingEnabled)
	require.True(t, updated.WhatsAppAIParsingEnabled)
}

func TestSetWhatsAppAIParsing_refusesToEnableWithNoAIConnection(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	aiConnRepo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil }}
	uc := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), connRepo, aiConnRepo)

	_, err := uc.Execute(context.Background(), "conn-1", true)

	require.ErrorIs(t, err, appintegrations.ErrAIConnectionRequiredForWhatsAppParsing)
}

func TestSetWhatsAppAIParsing_disablingNeverChecksAIConnection(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWhatsApp, WhatsAppAIParsingEnabled: true}
	var updated *integrationsDomain.Connection
	connRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	aiConnRepoCalled := false
	aiConnRepo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		aiConnRepoCalled = true
		return nil, nil
	}}
	uc := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), connRepo, aiConnRepo)

	_, err := uc.Execute(context.Background(), "conn-1", false)

	require.NoError(t, err)
	require.False(t, updated.WhatsAppAIParsingEnabled)
	require.False(t, aiConnRepoCalled, "turning it off should never require an AI connection to exist")
}

func TestSetWhatsAppAIParsing_rejectsNonWhatsAppConnection(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	uc := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), connRepo, &mocks.AIConnectionRepo{})

	_, err := uc.Execute(context.Background(), "conn-1", true)

	require.ErrorIs(t, err, integrationsDomain.ErrNotWhatsAppConnection)
}

func TestSetWhatsAppAIParsing_propagatesNotFound(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
		return nil, integrationsDomain.ErrConnectionNotFound
	}}
	uc := appintegrations.NewSetWhatsAppAIParsingUseCase(newMockGormDB(t), connRepo, &mocks.AIConnectionRepo{})

	_, err := uc.Execute(context.Background(), "missing", true)

	require.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)
}
