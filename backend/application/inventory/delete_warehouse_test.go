package inventory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestDeleteWarehouse_deletesExisting(t *testing.T) {
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "w1"}}
	uc := appinv.NewDeleteWarehouseUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "w1")

	require.NoError(t, err)
	require.Equal(t, []string{"w1"}, repo.deleted)
}

func TestDeleteWarehouse_propagatesNotFound(t *testing.T) {
	repo := &stubWarehouseRepo{findErr: invDomain.ErrWarehouseNotFound}
	uc := appinv.NewDeleteWarehouseUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrWarehouseNotFound))
	require.Empty(t, repo.deleted)
}

func TestDeleteWarehouse_propagatesDeleteError(t *testing.T) {
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "w1"}, deleteErr: errors.New("in use")}
	uc := appinv.NewDeleteWarehouseUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "w1")

	require.Error(t, err)
	require.ErrorContains(t, err, "in use")
}
