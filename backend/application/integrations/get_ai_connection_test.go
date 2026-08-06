package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestGetAIConnection_returnsConfiguredConnection(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
			return &integrationsDomain.AIConnection{TenantID: "tenant-1", Provider: integrationsDomain.AIProviderGemini}, nil
		},
	}
	uc := appintegrations.NewGetAIConnectionUseCase(newMockGormDB(t), repo)

	conn, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.Equal(t, integrationsDomain.AIProviderGemini, conn.Provider)
}

func TestGetAIConnection_returnsNilWhenNotConfigured(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil },
	}
	uc := appintegrations.NewGetAIConnectionUseCase(newMockGormDB(t), repo)

	conn, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Nil(t, conn)
}

func TestGetAIConnection_propagatesRepoError(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, errors.New("db down") },
	}
	uc := appintegrations.NewGetAIConnectionUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}
