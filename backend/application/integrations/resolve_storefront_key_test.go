package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestResolveStorefrontKey_returnsTenantIDForAMatchingHash(t *testing.T) {
	// Generate a real key + hash through SetStorefrontKeyUseCase so this
	// test exercises the exact same hashing both use cases share, rather
	// than assuming they agree.
	conn := &integrationsDomain.Connection{ID: "conn-1", TenantID: "tenant-1"}
	setRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn:   func(_ context.Context, c *integrationsDomain.Connection) error { conn = c; return nil },
	}
	setUC := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), setRepo)
	setResult, err := setUC.Execute(context.Background(), "conn-1")
	require.NoError(t, err)

	resolveRepo := &mocks.ConnectionRepo{
		FindByStorefrontKeyHashFn: func(_ context.Context, hash string) (*integrationsDomain.Connection, error) {
			require.Equal(t, conn.StorefrontKeyHash, hash)
			return conn, nil
		},
	}
	resolveUC := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), resolveRepo)

	tenantID, err := resolveUC.Execute(context.Background(), setResult.PlaintextKey)

	require.NoError(t, err)
	require.Equal(t, "tenant-1", tenantID)
}

func TestResolveStorefrontKey_rejectsUnknownKey(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByStorefrontKeyHashFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "wrong-key")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}

func TestResolveStorefrontKey_rejectsEmptyKey(t *testing.T) {
	uc := appintegrations.NewResolveStorefrontKeyUseCase(newMockGormDB(t), &mocks.ConnectionRepo{})

	_, err := uc.Execute(context.Background(), "")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
