package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// TestPortionFromBulk_consumesDefaultWeightFromBulkPool is the "book 20kg,
// peel off a 378g pack" case: the target SKU's own WeightGrams is used when
// no explicit weight is given.
func TestPortionFromBulk_consumesDefaultWeightFromBulkPool(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, outboxRepo)

	result, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-bulk-1", result.BulkOutMovement.SKUID)
	assert.Equal(t, invDomain.MovementTypeSplitOut, result.BulkOutMovement.Type)
	assert.Equal(t, int64(-378), result.BulkOutMovement.QuantityDelta)
	assert.Equal(t, "sku-378g", result.UnitInMovement.SKUID)
	assert.Equal(t, invDomain.MovementTypeSplitIn, result.UnitInMovement.Type)
	assert.Equal(t, int64(1), result.UnitInMovement.QuantityDelta)
	require.Len(t, movementRepo.created, 2)
	assert.ElementsMatch(t, outboxRepo.calls, []string{"inventory.stock.changed", "inventory.stock.changed"})
}

// TestPortionFromBulk_overridesWithActualScaleReading covers the case
// where the physical item's real weight differs from the SKU's nominal
// WeightGrams (e.g. a "~400g" catalog item that actually weighed 412g).
func TestPortionFromBulk_overridesWithActualScaleReading(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-400g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(400),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-400g", WarehouseID: "warehouse-1", WeightGrams: int64Ptr(412),
	})

	require.NoError(t, err)
	assert.Equal(t, int64(-412), result.BulkOutMovement.QuantityDelta, "actual scale reading wins over the nominal 400g")
}

func TestPortionFromBulk_rejectsWhenSKUIsNotPortioned(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-1", WarehouseID: "warehouse-1",
	})

	require.ErrorIs(t, err, invDomain.ErrSKUIsNotPortioned)
}

func TestPortionFromBulk_rejectsInsufficientBulkStock(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 100},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1",
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestPortionFromBulk_scopesStockCheckToWarehouseAndBatch(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	// 20kg exist, but at a different warehouse -- must not count toward
	// this warehouse's available pool.
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-other", Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1",
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestPortionFromBulk_rejectsZeroWeightWithNoNominalWeightSet(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", PortionedFromSKUID: strPtr("sku-bulk-1")}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-1", WarehouseID: "warehouse-1",
	})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestPortionFromBulk_rejectsNonPositiveExplicitWeight(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1", WeightGrams: int64Ptr(0),
	})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestPortionFromBulk_propagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "missing", WarehouseID: "warehouse-1",
	})

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

// TestPortionFromBulk_unitStockIsUnBatchTrackedByDefault mirrors
// SplitCaseIntoUnits' same default: the portioned item lands in general
// (un-lotted) stock unless the caller explicitly opts in via
// Input.TargetBatchID (see the test below), even when the bulk stock it
// was peeled from came from a specific batch/lot.
func TestPortionFromBulk_unitStockIsUnBatchTrackedByDefault(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	batchID := "batch-1"
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", BatchID: &batchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1", BatchID: &batchID,
	})

	require.NoError(t, err)
	assert.Equal(t, &batchID, result.BulkOutMovement.BatchID)
	assert.Nil(t, result.UnitInMovement.BatchID)
}

// TestPortionFromBulk_linksUnitStockToTargetBatchWhenSet is the regression
// test for the new opt-in: BulkAddWeightPortionsUseCase sets TargetBatchID
// so a portioned SKU's stock can carry the bulk lot's own expiry forward
// (see that use case's own doc comment) instead of landing un-lotted.
func TestPortionFromBulk_linksUnitStockToTargetBatchWhenSet(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-378g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(378),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})
	targetBatchID := "batch-new-sku-1"

	result, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-378g", WarehouseID: "warehouse-1", TargetBatchID: &targetBatchID,
	})

	require.NoError(t, err)
	require.NotNil(t, result.UnitInMovement.BatchID)
	assert.Equal(t, targetBatchID, *result.UnitInMovement.BatchID)
}

// TestPortionFromBulk_portionsQuantityPackagesOfTheSameWeightAtOnce covers
// "10 packages of 380g": one call takes Quantity x WeightGrams off the bulk
// pool and adds Quantity units, instead of ten separate round trips.
func TestPortionFromBulk_portionsQuantityPackagesOfTheSameWeightAtOnce(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-380g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(380),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 20000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-380g", WarehouseID: "warehouse-1", Quantity: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(-3800), result.BulkOutMovement.QuantityDelta)
	assert.Equal(t, int64(10), result.UnitInMovement.QuantityDelta)
}

func TestPortionFromBulk_checksStockForTheWholeQuantity(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-380g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(380),
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-bulk-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 1000},
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-380g", WarehouseID: "warehouse-1", Quantity: 3,
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock, "3 x 380g = 1140g, only 1000g in the pool")
	assert.Empty(t, movementRepo.created)
}

func TestPortionFromBulk_rejectsNegativeQuantity(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-380g", PortionedFromSKUID: strPtr("sku-bulk-1"), WeightGrams: int64Ptr(380),
	}}
	uc := appinventory.NewPortionFromBulkUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.PortionFromBulkInput{
		TargetSKUID: "sku-380g", WarehouseID: "warehouse-1", Quantity: -1,
	})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}
