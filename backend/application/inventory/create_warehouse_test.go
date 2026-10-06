package inventory_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestCreateWarehouse_setsNameAndTenant(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), warehouseRepo, &eventrecordtest.Outbox{})

	w, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{Name: "Main DC"})

	require.NoError(t, err)
	require.Equal(t, "Main DC", w.Name)
	require.True(t, warehouseRepo.createCalled)
}

func TestCreateWarehouse_setsAddressFields(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), warehouseRepo, &eventrecordtest.Outbox{})

	w, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{
		Name: "Main DC", Address1: "1 Main St", Address2: "Suite 100", City: "Springfield",
		Province: "IL", Zip: "62701", Country: "US", Phone: "+1 555-0100", Active: true,
	})

	require.NoError(t, err)
	require.Equal(t, "1 Main St", w.Address1)
	require.Equal(t, "Suite 100", w.Address2)
	require.Equal(t, "Springfield", w.City)
	require.Equal(t, "IL", w.Province)
	require.Equal(t, "62701", w.Zip)
	require.Equal(t, "US", w.Country)
	require.Equal(t, "+1 555-0100", w.Phone)
	require.True(t, w.Active)
}

func TestCreateWarehouse_propagatesRepoError(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{createErr: invDomain.ErrWarehouseNotFound}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), warehouseRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{Name: "Main DC"})

	require.Error(t, err)
}
