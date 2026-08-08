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

func int64Ptr(v int64) *int64 { return &v }

// TestSplitCaseIntoUnits_convertsCaseStockIntoUnitStock is the "box of a
// product" edge case: splitting 2 cases of a 36-unit case SKU must remove 2
// from the case SKU's ledger and add 72 to its unit SKU's ledger, atomically.
func TestSplitCaseIntoUnits_convertsCaseStockIntoUnitStock(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-case-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 5},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, movementRepo, outboxRepo)

	result, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, "sku-case-1", result.CaseOutMovement.SKUID)
	assert.Equal(t, invDomain.MovementTypeSplitOut, result.CaseOutMovement.Type)
	assert.Equal(t, int64(-2), result.CaseOutMovement.QuantityDelta)
	assert.Equal(t, "sku-unit-1", result.UnitInMovement.SKUID)
	assert.Equal(t, invDomain.MovementTypeSplitIn, result.UnitInMovement.Type)
	assert.Equal(t, int64(72), result.UnitInMovement.QuantityDelta, "2 cases * 36 units/case")
	require.Len(t, movementRepo.created, 2)
	assert.ElementsMatch(t, outboxRepo.calls, []string{"inventory.stock.changed", "inventory.stock.changed"})
}

func TestSplitCaseIntoUnits_rejectsWhenSKUIsNotACase(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-1", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrSKUIsNotACase)
}

func TestSplitCaseIntoUnits_rejectsInsufficientCaseStock(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-case-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 1},
	}}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 2,
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestSplitCaseIntoUnits_scopesStockCheckToWarehouseAndBatch(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-case-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
	// 5 cases exist, but at a different warehouse -- must not count toward
	// this warehouse's available stock.
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-other", Type: invDomain.MovementTypeReceipt, QuantityDelta: 5},
	}}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestSplitCaseIntoUnits_rejectsNonPositiveCasesToSplit(t *testing.T) {
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), &stubSKURepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 0,
	})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestSplitCaseIntoUnits_propagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "missing", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

// TestSplitCaseIntoUnits_unitStockIsNeverBatchTracked documents the
// deliberate limitation: the unit side of a split always lands in general
// (un-lotted) stock, even when the case stock being split came from a
// specific batch/lot.
func TestSplitCaseIntoUnits_unitStockIsNeverBatchTracked(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-case-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
	batchID := "batch-1"
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 5},
	}}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, &batchID, result.CaseOutMovement.BatchID)
	assert.Nil(t, result.UnitInMovement.BatchID)
}
