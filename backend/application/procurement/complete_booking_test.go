package procurement_test

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/errs"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func completableBookingFixtures() (*stubBookingRepo, *stubPurchaseOrderRepo) {
	return &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}},
		&stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
}

func TestCompleteBooking_replaysAStagedReceiveIntoRealStock(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	adjustment := &stubRecordAdjustmentForBooking{}
	writeOff := &stubMarkDamagedOrLostForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, adjustment, writeOff, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, receiveStock.calls, 1)
	assert.Equal(t, "sku-1", receiveStock.calls[0].SKUID)
	assert.Equal(t, "warehouse-1", receiveStock.calls[0].WarehouseID)
	assert.Equal(t, int64(10), receiveStock.calls[0].Quantity)
	assert.Empty(t, adjustment.calls)
	assert.Empty(t, writeOff.calls)
	require.Len(t, scanRepo.markReplayed, 1, "the applied row must be marked replayed so a retried Finish won't re-apply it")
	assert.Equal(t, "scan-1", scanRepo.markReplayed[0].scanID)
	assert.Empty(t, result.MissingItems)
	assert.Equal(t, procurementDomain.BookingStatusCompleted, result.Booking.Status)
	assert.Equal(t, "user-1", result.Booking.CompletedBy)
}

func TestCompleteBooking_computesLandedCostForAReceiveFromItsLineItemAndSharedShipping(t *testing.T) {
	db := newMockGormDBN(t, 3)
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", SKUID: strPtr("sku-1"), Quantity: 10, UnitCostMinorUnits: 800, LineType: procurementDomain.LineItemTypeProduct},
		{ID: "line-2", SKUID: strPtr("sku-2"), Quantity: 5, UnitCostMinorUnits: 400, LineType: procurementDomain.LineItemTypeProduct},
		{ID: "line-shipping", Quantity: 1, UnitCostMinorUnits: 1000, LineType: procurementDomain.LineItemTypeShipping},
	}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, receiveStock.calls, 1)
	require.NotNil(t, receiveStock.calls[0].PurchaseOrderLineItemID)
	assert.Equal(t, "line-1", *receiveStock.calls[0].PurchaseOrderLineItemID)
	require.NotNil(t, receiveStock.calls[0].UnitCostMinorUnits)
	assert.Equal(t, int64(880), *receiveStock.calls[0].UnitCostMinorUnits) // 800 + 800/1000*1000/10
}

func TestCompleteBooking_replaysAStagedCorrectionAsANegativeAdjustment(t *testing.T) {
	db := newMockGormDBN(t, 4) // read + mark-replayed(2 rows) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	batchID := "batch-1"
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10, BatchID: &batchID},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -3, BatchID: &batchID, Reason: "counted wrong"},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	adjustment := &stubRecordAdjustmentForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, adjustment, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, adjustment.calls, 1)
	assert.Equal(t, "sku-1", adjustment.calls[0].SKUID)
	assert.Equal(t, &batchID, adjustment.calls[0].BatchID)
	assert.Equal(t, invDomain.MovementTypeAdjustment, adjustment.calls[0].Type)
	assert.Equal(t, int64(-3), adjustment.calls[0].QuantityDelta)
	assert.Equal(t, "counted wrong", adjustment.calls[0].Reason)
	// Net received: 10 - 3 = 7, still short of the expected 10.
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, int64(7), result.MissingItems[0].ScannedQty)
}

func TestCompleteBooking_replaysADamagedWriteOffWithoutReducingReceivedProgress(t *testing.T) {
	db := newMockGormDBN(t, 4) // read + mark-replayed(2 rows) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	batchID := "batch-1"
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10, BatchID: &batchID},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, Quantity: 0, WriteOffQuantity: 2, BatchID: &batchID, Reason: "arrived crushed"},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	writeOff := &stubMarkDamagedOrLostForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, &stubRecordAdjustmentForBooking{}, writeOff, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, writeOff.calls, 1)
	assert.Equal(t, "sku-1", writeOff.calls[0].SKUID)
	assert.Equal(t, &batchID, writeOff.calls[0].BatchID)
	assert.Equal(t, invDomain.MovementTypeDamaged, writeOff.calls[0].Type)
	assert.Equal(t, int64(2), writeOff.calls[0].Quantity)
	assert.Equal(t, "arrived crushed", writeOff.calls[0].Reason)
	// Damaged doesn't reduce received progress -- 10 received still meets
	// the expected 10, no gap.
	assert.Empty(t, result.MissingItems)
}

func TestCompleteBooking_writeOffCountsTowardScannedProgressAlongsideReceives(t *testing.T) {
	// The reported case: 24 expected, 23 received, 1 damaged -- the damaged
	// unit still arrived (it just isn't sellable), so nothing should be
	// missing.
	db := newMockGormDBN(t, 4) // read + mark-replayed(2 rows) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 24, LineType: procurementDomain.LineItemTypeProduct}}}
	batchID := "batch-1"
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 23, BatchID: &batchID},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 1, BatchID: &batchID},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, result.MissingItems, "23 received + 1 damaged accounts for all 24 expected")
}

func TestCompleteBooking_marksThePurchaseOrderReceivedWhenNothingIsMissing(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, poRepo.updated, 1)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, poRepo.updated[0].Status)
}

func TestCompleteBooking_marksThePurchaseOrderPartiallyReceivedWhenSomethingIsMissing(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 7},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, poRepo.updated, 1)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, poRepo.updated[0].Status)
}

func TestCompleteBooking_replaysALostWriteOff(t *testing.T) {
	db := newMockGormDBN(t, 4) // read + mark-replayed(2 rows) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionLost, WriteOffQuantity: 1},
	}}
	writeOff := &stubMarkDamagedOrLostForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, writeOff, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, writeOff.calls, 1)
	assert.Equal(t, invDomain.MovementTypeLost, writeOff.calls[0].Type)
}

func TestCompleteBooking_resolvesAPendingBatchAcrossLaterRowsByLotNumber(t *testing.T) {
	db := newMockGormDBN(t, 5) // read + mark-replayed(3 rows) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 20, LineType: procurementDomain.LineItemTypeProduct}}}
	// Two receive rows into the same not-yet-real "LOT-NEW" batch, then a
	// damaged row against that same pending batch -- none of the three
	// carry a real BatchID (nothing was real until now).
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10, LotNumber: "LOT-NEW"},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10, LotNumber: "LOT-NEW"},
		{ID: "scan-3", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 2, LotNumber: "LOT-NEW"},
	}}
	receiveStock := &stubReceiveStockForBooking{outs: []*appinventory.ReceiveStockResult{
		{Batch: &invDomain.Batch{ID: "real-batch-1"}, Movement: &invDomain.StockMovement{ID: "m1"}},
	}}
	writeOff := &stubMarkDamagedOrLostForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, &stubRecordAdjustmentForBooking{}, writeOff, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, receiveStock.calls, 2)
	assert.Nil(t, receiveStock.calls[0].BatchID, "first receive into this lot creates the batch")
	require.NotNil(t, receiveStock.calls[1].BatchID, "second receive resolves to the batch the first one created")
	assert.Equal(t, "real-batch-1", *receiveStock.calls[1].BatchID)
	require.Len(t, writeOff.calls, 1)
	require.NotNil(t, writeOff.calls[0].BatchID, "damaged row against the same lot resolves to the same real batch")
	assert.Equal(t, "real-batch-1", *writeOff.calls[0].BatchID)
}

func TestCompleteBooking_rejectsACorrectionWhoseLotWasNeverStaged(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	// A correction referencing a lot with no earlier receive row -- nothing
	// to resolve it against.
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -1, LotNumber: "GHOST-LOT"},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	assert.ErrorIs(t, err, procurementDomain.ErrPendingBatchNotFound)
}

func TestCompleteBooking_skipsARowAlreadyReplayedByAnEarlierFinishAttempt(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 unreplayed row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	replayedAt := time.Now().Add(-time.Hour)
	batchID := "batch-1"
	// scan-1 already applied by a prior (crashed-before-completing) Finish
	// attempt -- receiveStock must NOT be called for it again. scan-2 is
	// still unreplayed and must go through as normal.
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 5, BatchID: &batchID, ReplayedAt: &replayedAt},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -1, BatchID: &batchID, Reason: "counted wrong"},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	adjustment := &stubRecordAdjustmentForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, adjustment, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, receiveStock.calls, "already-replayed receive must not be re-applied")
	require.Len(t, adjustment.calls, 1, "the not-yet-replayed correction must still be applied")
	require.Len(t, scanRepo.markReplayed, 1, "only the newly-applied row gets marked -- the already-replayed one is untouched")
	assert.Equal(t, "scan-2", scanRepo.markReplayed[0].scanID)
	assert.Equal(t, procurementDomain.BookingStatusCompleted, result.Booking.Status)
}

func TestCompleteBooking_resolvesAPendingBatchFromAnAlreadyReplayedRow(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 unreplayed row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	replayedAt := time.Now().Add(-time.Hour)
	realBatchID := "real-batch-77"
	// scan-1 already replayed and, since it created the lot, already has its
	// resolved real BatchID persisted. scan-2 is a not-yet-replayed
	// correction still referencing the lot by name (it was staged before
	// scan-1 became real) -- it must resolve to scan-1's real batch without
	// calling receiveStock again.
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10, LotNumber: "LOT-X", BatchID: &realBatchID, ReplayedAt: &replayedAt},
		{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -2, LotNumber: "LOT-X", Reason: "counted wrong"},
	}}
	receiveStock := &stubReceiveStockForBooking{}
	adjustment := &stubRecordAdjustmentForBooking{}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, adjustment, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, receiveStock.calls)
	require.Len(t, adjustment.calls, 1)
	require.NotNil(t, adjustment.calls[0].BatchID)
	assert.Equal(t, realBatchID, *adjustment.calls[0].BatchID)
}

func TestCompleteBooking_excludesNonProductLinesFromMissingItems(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	skuID := "sku-1"
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", SKUID: &skuID, RawDescription: "Widget A", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct},
		{ID: "line-2", RawDescription: "Transporte DACHSER", Quantity: 1, LineType: procurementDomain.LineItemTypeShipping},
		{ID: "line-3", RawDescription: "Desconto", Quantity: 1, LineType: procurementDomain.LineItemTypeDiscount},
	}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, result.MissingItems, "shipping/discount lines must never appear in the missing-items report")
}

func TestCompleteBooking_scalesExpectedQtyByUnitsPerPurchasedItem(t *testing.T) {
	db := newMockGormDBN(t, 3) // read + mark-replayed(1 row) + mark-completed
	bookingRepo, poRepo := completableBookingFixtures()
	skuID := "sku-unit-1"
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", SKUID: &skuID, RawDescription: "Rapadurinha 36x25gr", Quantity: 1, UnitsPerPurchasedItem: 36, LineType: procurementDomain.LineItemTypeProduct},
	}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-unit-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 30},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 1)
	assert.Equal(t, int64(36), result.MissingItems[0].ExpectedQty, "1 box * 36 units/box")
	assert.Equal(t, int64(30), result.MissingItems[0].ScannedQty)
}

func TestCompleteBooking_rejectsAlreadyCompletedBooking(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusCompleted}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, &stubLineItemRepo{}, &stubBookingScanRepo{}, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
}

func TestCompleteBooking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, &stubLineItemRepo{}, &stubBookingScanRepo{}, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing", "user-1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}

func TestCompleteBooking_propagatesReplayError(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 10},
	}}
	receiveStock := &stubReceiveStockForBooking{err: invDomain.ErrInsufficientStock}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, receiveStock, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.Error(t, err)
}

// The Finish error names the row it stopped on, so the scanner can say
// which entry to fix: the line's own description plus the lot.
func TestCompleteBooking_orphanedWriteOffErrorNamesProductAndLot(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Pão de queijo 1kg", Quantity: 47, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 1, LotNumber: "31/10/2027"},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.ErrorIs(t, err, procurementDomain.ErrPendingBatchNotFound)
	detail, ok := errs.DetailOf(err)
	require.True(t, ok)
	assert.Equal(t, "Pão de queijo 1kg · 31/10/2027", detail)
}

func TestCompleteBooking_insufficientStockErrorNamesTheProduct(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := completableBookingFixtures()
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), RawDescription: "Pão de queijo 1kg", Quantity: 47, LineType: procurementDomain.LineItemTypeProduct}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 1},
	}}
	writeOff := &stubMarkDamagedOrLostForBooking{err: fmt.Errorf("markDamagedOrLost: %w", invDomain.ErrInsufficientStock)}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, lineItemRepo, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, writeOff, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.ErrorIs(t, err, invDomain.ErrInsufficientStock)
	detail, ok := errs.DetailOf(err)
	require.True(t, ok)
	assert.Equal(t, "Pão de queijo 1kg", detail)
}

func TestCompleteBooking_rowWithoutALineItemIsNamedBySKU(t *testing.T) {
	db := newMockGormDBN(t, 1)
	bookingRepo, poRepo := completableBookingFixtures()
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionLost, WriteOffQuantity: 1, LotNumber: "L9"},
	}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, poRepo, &stubLineItemRepo{}, scanRepo, &stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	detail, ok := errs.DetailOf(err)
	require.True(t, ok)
	assert.Equal(t, "sku-1 · L9", detail)
}
