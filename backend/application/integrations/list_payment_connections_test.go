package integrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestListPaymentConnections_returnsOnlyConnectedPaymentProviders(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{
		ListByTenantFn: func(_ context.Context) ([]integrationsDomain.Connection, error) {
			return []integrationsDomain.Connection{
				{ID: "conn-woo", Provider: integrationsDomain.ProviderWooCommerce, Status: integrationsDomain.StatusConnected},
				{ID: "conn-stripe", Provider: integrationsDomain.ProviderStripe, Status: integrationsDomain.StatusConnected},
				{ID: "conn-revolut-error", Provider: integrationsDomain.ProviderRevolut, Status: integrationsDomain.StatusError},
				{ID: "conn-revolut-ok", Provider: integrationsDomain.ProviderRevolut, Status: integrationsDomain.StatusConnected},
			}, nil
		},
	}
	uc := appintegrations.NewListPaymentConnectionsUseCase(newMockGormDB(t), connRepo)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"))

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "conn-stripe", result[0].ID)
	assert.Equal(t, "conn-revolut-ok", result[1].ID)
}

func TestListPaymentConnections_none_returnsEmpty(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{
		ListByTenantFn: func(_ context.Context) ([]integrationsDomain.Connection, error) { return nil, nil },
	}
	uc := appintegrations.NewListPaymentConnectionsUseCase(newMockGormDB(t), connRepo)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"))

	require.NoError(t, err)
	assert.Empty(t, result)
}
