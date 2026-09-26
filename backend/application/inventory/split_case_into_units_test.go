package inventory_test

import (
	"context"
	"testing"
	"time"

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
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubBatchRepo{}, movementRepo, outboxRepo)

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
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{})

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
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubBatchRepo{}, movementRepo, &stubOutboxRepo{})

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
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubBatchRepo{}, movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestSplitCaseIntoUnits_rejectsNonPositiveCasesToSplit(t *testing.T) {
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), &stubSKURepo{}, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 0,
	})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestSplitCaseIntoUnits_propagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), skuRepo, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "missing", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func caseSKURepo() *stubSKURepo {
	return &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-case-1", CaseOfSKUID: strPtr("sku-unit-1"), UnitsPerCase: int64Ptr(36)}}
}

func caseStockInBatch(batchID string, qty int64) *stubStockMovementRepo {
	return &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: qty},
	}}
}

// TestSplitCaseIntoUnits_fromBatchCreatesMatchingUnitBatch: splitting a
// case out of a lot must not strip the lot -- the units land in a batch on
// the unit SKU carrying the same lot number, expiry and receipt date, so
// FEFO, expiry alerts and batch-tied promotions still see them.
func TestSplitCaseIntoUnits_fromBatchCreatesMatchingUnitBatch(t *testing.T) {
	expiry := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	received := time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{
		ID: "batch-case-1", SKUID: "sku-case-1", WarehouseID: "warehouse-1",
		LotNumber: "300326", ExpiryDate: &expiry, ReceivedAt: received, Status: invDomain.BatchStatusActive,
	}}
	batchID := "batch-case-1"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, batchRepo.created, "a unit-side batch must be created")
	assert.Equal(t, "sku-unit-1", batchRepo.created.SKUID)
	assert.Equal(t, "warehouse-1", batchRepo.created.WarehouseID)
	assert.Equal(t, "300326", batchRepo.created.LotNumber)
	assert.Equal(t, &expiry, batchRepo.created.ExpiryDate)
	assert.Equal(t, received, batchRepo.created.ReceivedAt)
	assert.Equal(t, invDomain.BatchStatusActive, batchRepo.created.Status)
	assert.Equal(t, []string{"sku-unit-1"}, batchRepo.listedSKUIDs)
	assert.Equal(t, &batchID, result.CaseOutMovement.BatchID)
	require.NotNil(t, result.UnitInMovement.BatchID)
	assert.Equal(t, batchRepo.created.ID, *result.UnitInMovement.BatchID)
}

// TestSplitCaseIntoUnits_fromBatchReusesExistingMatchingUnitBatch: a
// second case split from the same lot tops up the unit batch the first
// split created instead of spawning a duplicate lot.
func TestSplitCaseIntoUnits_fromBatchReusesExistingMatchingUnitBatch(t *testing.T) {
	expiry := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	sameExpiry := expiry
	batchRepo := &stubBatchRepo{
		findResult: &invDomain.Batch{ID: "batch-case-1", SKUID: "sku-case-1", WarehouseID: "warehouse-1", LotNumber: "300326", ExpiryDate: &expiry},
		batches: []invDomain.Batch{
			{ID: "unit-other-wh", SKUID: "sku-unit-1", WarehouseID: "warehouse-2", LotNumber: "300326", ExpiryDate: &sameExpiry, Status: invDomain.BatchStatusActive},
			{ID: "unit-other-expiry", SKUID: "sku-unit-1", WarehouseID: "warehouse-1", LotNumber: "300326", Status: invDomain.BatchStatusActive},
			{ID: "unit-expired", SKUID: "sku-unit-1", WarehouseID: "warehouse-1", LotNumber: "300326", ExpiryDate: &sameExpiry, Status: invDomain.BatchStatusExpired},
			{ID: "unit-match", SKUID: "sku-unit-1", WarehouseID: "warehouse-1", LotNumber: "300326", ExpiryDate: &sameExpiry, Status: invDomain.BatchStatusActive},
		},
	}
	batchID := "batch-case-1"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.NoError(t, err)
	assert.False(t, batchRepo.createCalled, "an existing matching unit batch must be reused")
	require.NotNil(t, result.UnitInMovement.BatchID)
	assert.Equal(t, "unit-match", *result.UnitInMovement.BatchID)
}

func TestSplitCaseIntoUnits_fromGeneralLedgerKeepsUnitsInGeneralLedger(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-case-1", WarehouseID: "warehouse-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 5},
	}}
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", CasesToSplit: 1,
	})

	require.NoError(t, err)
	assert.Nil(t, result.UnitInMovement.BatchID)
	assert.False(t, batchRepo.createCalled)
}

func TestSplitCaseIntoUnits_rejectsBatchOfAnotherSKU(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-x", SKUID: "sku-other", WarehouseID: "warehouse-1"}}
	batchID := "batch-x"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrBatchDoesNotBelongToSKU)
}

func TestSplitCaseIntoUnits_rejectsBatchInAnotherWarehouse(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-case-1", SKUID: "sku-case-1", WarehouseID: "warehouse-2"}}
	batchID := "batch-case-1"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrBatchNotInWarehouse)
}

func TestSplitCaseIntoUnits_propagatesBatchNotFound(t *testing.T) {
	batchRepo := &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}
	batchID := "missing"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.ErrorIs(t, err, invDomain.ErrBatchNotFound)
}

func TestSplitCaseIntoUnits_propagatesUnitBatchCreateError(t *testing.T) {
	createErr := assert.AnError
	batchRepo := &stubBatchRepo{
		findResult: &invDomain.Batch{ID: "batch-case-1", SKUID: "sku-case-1", WarehouseID: "warehouse-1", LotNumber: "L"},
		createErr:  createErr,
	}
	batchID := "batch-case-1"
	uc := appinventory.NewSplitCaseIntoUnitsUseCase(newMockGormDB(t), caseSKURepo(), batchRepo, caseStockInBatch(batchID, 5), &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: "sku-case-1", WarehouseID: "warehouse-1", BatchID: &batchID, CasesToSplit: 1,
	})

	require.ErrorIs(t, err, createErr)
}
