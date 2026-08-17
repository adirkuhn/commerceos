package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListBatches_returnsBatchesForSKU(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}}
	movementRepo := &stubStockMovementRepo{}
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "wh-1"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo, movementRepo, warehouseRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	// General-ledger row for the batch's warehouse comes first (at 0, since
	// no un-lotted movements exist), then the real batch.
	require.Len(t, batches, 2)
	require.True(t, batches[0].IsGeneralLedger)
	require.False(t, batches[1].IsGeneralLedger)
	require.Equal(t, "batch-1", batches[1].ID)
}

func TestListBatches_populatesQuantityPerBatchFromLedger(t *testing.T) {
	batchID1, batchID2 := "batch-1", "batch-2"
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: batchID1, SKUID: "sku-1", WarehouseID: "wh-1"},
		{ID: batchID2, SKUID: "sku-1", WarehouseID: "wh-1"},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID1, QuantityDelta: 100},
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID1, QuantityDelta: -30},
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID2, QuantityDelta: 5},
	}}
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "wh-1"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo, movementRepo, warehouseRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	byID := map[string]invDomain.Batch{}
	for _, b := range batches {
		byID[b.ID] = b
	}
	require.Equal(t, int64(70), byID[batchID1].Quantity)
	require.Equal(t, int64(5), byID[batchID2].Quantity)
}

func TestListBatches_addsGeneralLedgerRowPerWarehouseFirst(t *testing.T) {
	batchID := "batch-1"
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: batchID, SKUID: "sku-1", WarehouseID: "wh-1"},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		// batch-scoped: booked into batch-1, general ledger for wh-1 never touched.
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID, QuantityDelta: 100},
		// un-lotted stock in a second warehouse the SKU has no batch in.
		{SKUID: "sku-1", WarehouseID: "wh-2", BatchID: nil, QuantityDelta: 12},
	}}
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "wh-1"}, {ID: "wh-2"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo, movementRepo, warehouseRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	// General-ledger rows come first, one per warehouse the SKU has any
	// activity in -- wh-1 at 0 (never touched, but must still be visible so
	// a reviewer can see there's nothing to remove) and wh-2 at 12.
	require.Len(t, batches, 3)
	require.True(t, batches[0].IsGeneralLedger)
	require.Equal(t, "wh-1", batches[0].WarehouseID)
	require.Equal(t, int64(0), batches[0].Quantity)
	require.True(t, batches[1].IsGeneralLedger)
	require.Equal(t, "wh-2", batches[1].WarehouseID)
	require.Equal(t, int64(12), batches[1].Quantity)
	require.False(t, batches[2].IsGeneralLedger)
	require.Equal(t, batchID, batches[2].ID)
}

// A brand-new SKU that has never had a batch or movement recorded in any
// warehouse must still surface a qty:0 general-ledger row per tenant
// warehouse -- otherwise there's no row to tap in the UI at all, and stock
// can never be entered for it (see CameraScanner/InventoryApp bug report:
// "Total stock: 0" with no way to adjust or add a batch).
func TestListBatches_seedsGeneralLedgerRowForEveryWarehouseEvenWithNoActivity(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	movementRepo := &stubStockMovementRepo{}
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "wh-1"}, {ID: "wh-2"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo, movementRepo, warehouseRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, batches, 2)
	require.True(t, batches[0].IsGeneralLedger)
	require.Equal(t, "wh-1", batches[0].WarehouseID)
	require.Equal(t, int64(0), batches[0].Quantity)
	require.True(t, batches[1].IsGeneralLedger)
	require.Equal(t, "wh-2", batches[1].WarehouseID)
	require.Equal(t, int64(0), batches[1].Quantity)
}

// A warehouse that already got a general-ledger row from real activity
// (seeded earlier in the loop) must not be duplicated when the "every
// warehouse" seed pass runs afterwards.
func TestListBatches_doesNotDuplicateGeneralLedgerRowForWarehouseWithActivity(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: nil, QuantityDelta: 7},
	}}
	batchRepo := &stubBatchRepo{}
	warehouseRepo := &stubWarehouseRepo{warehouses: []invDomain.Warehouse{{ID: "wh-1"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo, movementRepo, warehouseRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Equal(t, int64(7), batches[0].Quantity)
}
