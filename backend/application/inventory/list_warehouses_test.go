package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListWarehouses_returnsRepoResult(t *testing.T) {
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "w1", Name: "Main DC"}}}
	uc := appinv.NewListWarehousesUseCase(newMockGormDB(t), warehouseRepo)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "Main DC", out[0].Name)
}
