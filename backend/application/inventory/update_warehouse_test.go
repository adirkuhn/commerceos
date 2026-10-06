package inventory_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestUpdateWarehouse_renamesWarehouse(t *testing.T) {
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "w1", Name: "Old name"}}
	uc := appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appinv.UpdateWarehouseInput{ID: "w1", Name: "New name"})

	require.NoError(t, err)
	require.Equal(t, "New name", result.Name)
	require.Len(t, repo.updated, 1)
	require.Equal(t, "New name", repo.updated[0].Name)
}

func TestUpdateWarehouse_updatesAddressFields(t *testing.T) {
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "w1", Name: "Main DC", City: "Old City"}}
	uc := appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appinv.UpdateWarehouseInput{
		ID: "w1", Name: "Main DC", Address1: "1 Main St", City: "New City", Province: "IL",
		Zip: "62701", Country: "US", Phone: "+1 555-0100", Active: false,
	})

	require.NoError(t, err)
	require.Equal(t, "1 Main St", result.Address1)
	require.Equal(t, "New City", result.City)
	require.False(t, result.Active)
	require.Len(t, repo.updated, 1)
	require.Equal(t, "New City", repo.updated[0].City)
}

func TestUpdateWarehouse_propagatesNotFound(t *testing.T) {
	repo := &stubWarehouseRepo{findErr: invDomain.ErrWarehouseNotFound}
	uc := appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.UpdateWarehouseInput{ID: "missing", Name: "New name"})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrWarehouseNotFound))
}

func TestUpdateWarehouse_propagatesUpdateError(t *testing.T) {
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "w1"}, updateErr: errors.New("db down")}
	uc := appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appinv.UpdateWarehouseInput{ID: "w1", Name: "New name"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
