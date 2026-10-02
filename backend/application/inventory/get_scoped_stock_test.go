package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestGetScopedStock_sumsOnlyTheGivenWarehouseAndBatch(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BatchID: strPtr("batch-1"), QuantityDelta: 50000},
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BatchID: strPtr("batch-1"), QuantityDelta: -7130},
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BatchID: strPtr("batch-2"), QuantityDelta: 9000},
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", QuantityDelta: 4000},
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-2", BatchID: strPtr("batch-1"), QuantityDelta: 1000},
	}}
	uc := appinventory.NewGetScopedStockUseCase(newMockGormDB(t), movementRepo)

	got, err := uc.Execute(context.Background(), "sku-bulk-1", "warehouse-1", strPtr("batch-1"))

	require.NoError(t, err)
	assert.Equal(t, int64(42870), got)
}

func TestGetScopedStock_nilBatchMeansGeneralLedger(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BatchID: strPtr("batch-1"), QuantityDelta: 50000},
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", QuantityDelta: 4000},
	}}
	uc := appinventory.NewGetScopedStockUseCase(newMockGormDB(t), movementRepo)

	got, err := uc.Execute(context.Background(), "sku-bulk-1", "warehouse-1", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(4000), got)
}
