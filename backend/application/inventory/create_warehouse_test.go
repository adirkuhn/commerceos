package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestCreateWarehouse_setsNameAndTenant(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), warehouseRepo)

	w, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{Name: "Main DC"})

	require.NoError(t, err)
	require.Equal(t, "Main DC", w.Name)
	require.True(t, warehouseRepo.createCalled)
}

func TestCreateWarehouse_propagatesRepoError(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{createErr: invDomain.ErrWarehouseNotFound}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), warehouseRepo)

	_, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{Name: "Main DC"})

	require.Error(t, err)
}
