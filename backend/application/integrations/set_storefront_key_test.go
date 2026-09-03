package integrations_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestSetStorefrontKey_generatesAndHashesKey(t *testing.T) {
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
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Equal(t, "conn-1", result.ConnectionID)
	require.NotEmpty(t, result.PlaintextKey)
	require.NotNil(t, updated)

	sum := sha256.Sum256([]byte(result.PlaintextKey))
	require.Equal(t, hex.EncodeToString(sum[:]), updated.StorefrontKeyHash)
}

func TestSetStorefrontKey_rotatesExistingKey(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", StorefrontKeyHash: "old-hash"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.NotEqual(t, "old-hash", updated.StorefrontKeyHash)
}

func TestSetStorefrontKey_propagatesNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewSetStorefrontKeyUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
