package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scenarios from a real booking (2026-10-28): a damaged write-off lost its
// receive row to a delete, and a damaged write-off re-added against the
// general ledger while every received unit sat in the booking's new lot
// batch. Both used to surface only at Finish, after part of the booking was
// already applied.

var stockT0 = time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)

func stagedReceive(id, lot string, qty int64, minute int) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{ID: id, BookingID: "booking-1", SKUID: "sku-1", LotNumber: lot,
		ActionType: procurementDomain.BookingScanActionReceive, Quantity: qty, ScannedAt: stockT0.Add(time.Duration(minute) * time.Minute)}
}

func stagedDamaged(id, lot string, qty int64, minute int) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{ID: id, BookingID: "booking-1", SKUID: "sku-1", LotNumber: lot,
		ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: qty, ScannedAt: stockT0.Add(time.Duration(minute) * time.Minute)}
}

func newScanUC(t *testing.T, scans *stubBookingScanRepo, stock *stubScopedStock) *appprocurement.ScanBookingItemUseCase {
	bookingRepo, poRepo := activeBookingFixtures()
	return appprocurement.NewScanBookingItemUseCase(newMockGormDBN(t, 2), bookingRepo, poRepo, scans, &stubSKURepo{},
		&stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, &eventrecordtest.Outbox{}, stock)
}

func newDeleteUC(t *testing.T, scans *stubBookingScanRepo, stock *stubScopedStock) *appprocurement.DeleteBookingScanUseCase {
	bookingRepo, poRepo := activeBookingFixtures()
	return appprocurement.NewDeleteBookingScanUseCase(newMockGormDBN(t, 2), bookingRepo, poRepo, scans, &eventrecordtest.Outbox{}, stock)
}

func damagedInput(lot string, batchID *string, qty int64) appprocurement.ScanBookingItemInput {
	return appprocurement.ScanBookingItemInput{BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged,
		Quantity: qty, LotNumber: lot, BatchID: batchID, ScannedBy: "user-1"}
}

// ── scanning a correction / damaged / lost row ──────────────────────────────

func TestScanBookingItem_damagedFromPendingLotWithinItsReceiveIsStaged(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{stagedReceive("r1", "31/10/2027", 47, 0)}}
	stock := &stubScopedStock{}

	_, err := newScanUC(t, scans, stock).Execute(context.Background(), damagedInput("31/10/2027", nil, 1))

	require.NoError(t, err)
	assert.Len(t, scans.created, 1)
	assert.Zero(t, stock.calls, "a pending lot is checked against this booking's receives, not real stock")
}

func TestScanBookingItem_damagedFromPendingLotWithNoReceiveIsRejected(t *testing.T) {
	scans := &stubBookingScanRepo{}

	_, err := newScanUC(t, scans, &stubScopedStock{}).Execute(context.Background(), damagedInput("31/10/2027", nil, 1))

	assert.ErrorIs(t, err, procurementDomain.ErrPendingBatchNotFound)
	assert.Empty(t, scans.created)
}

func TestScanBookingItem_damagedBeyondWhatThePendingLotReceivesIsRejected(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "L1", 3, 0), stagedDamaged("d1", "L1", 2, 1),
	}}

	_, err := newScanUC(t, scans, &stubScopedStock{}).Execute(context.Background(), damagedInput("L1", nil, 2))

	assert.ErrorIs(t, err, procurementDomain.ErrPendingLotInsufficient)
	assert.Empty(t, scans.created)
}

// The real booking: 47 received into lot 31/10/2027, then damaged 1 from the
// general ledger, which held nothing for this product.
func TestScanBookingItem_damagedFromEmptyGeneralLedgerIsRejectedWhenScanned(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{stagedReceive("r1", "31/10/2027", 47, 0)}}

	_, err := newScanUC(t, scans, &stubScopedStock{onHand: map[string]int64{"": 0}}).Execute(context.Background(), damagedInput("", nil, 1))

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
	assert.Empty(t, scans.created)
}

func TestScanBookingItem_damagedFromGeneralLedgerWithStockIsStaged(t *testing.T) {
	scans := &stubBookingScanRepo{}

	_, err := newScanUC(t, scans, &stubScopedStock{onHand: map[string]int64{"": 5}}).Execute(context.Background(), damagedInput("", nil, 5))

	require.NoError(t, err)
	assert.Len(t, scans.created, 1)
}

func TestScanBookingItem_realBatchCountsWriteOffsAlreadyStagedOnIt(t *testing.T) {
	batchID := "batch-9"
	earlier := stagedDamaged("d1", "", 1, 0)
	earlier.BatchID = &batchID
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{earlier}}

	_, err := newScanUC(t, scans, &stubScopedStock{onHand: map[string]int64{batchID: 1}}).Execute(context.Background(), damagedInput("", &batchID, 1))

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
	assert.Empty(t, scans.created)
}

func TestScanBookingItem_correctionIsCheckedLikeAWriteOff(t *testing.T) {
	scans := &stubBookingScanRepo{}
	in := damagedInput("L1", nil, 1)
	in.ActionType = procurementDomain.BookingScanActionCorrection

	_, err := newScanUC(t, scans, &stubScopedStock{}).Execute(context.Background(), in)

	assert.ErrorIs(t, err, procurementDomain.ErrPendingBatchNotFound)
	assert.Empty(t, scans.created)
}

func TestScanBookingItem_receiveNeedsNoStockCheck(t *testing.T) {
	scans := &stubBookingScanRepo{}
	stock := &stubScopedStock{err: errors.New("must not be called")}

	_, err := newScanUC(t, scans, stock).Execute(context.Background(), appprocurement.ScanBookingItemInput{
		BookingID: "booking-1", SKUID: "sku-1", Quantity: 3, LotNumber: "L1",
	})

	require.NoError(t, err)
	assert.Zero(t, stock.calls)
}

func TestScanBookingItem_stockLookupFailureIsReturned(t *testing.T) {
	scans := &stubBookingScanRepo{}
	boom := errors.New("db down")

	_, err := newScanUC(t, scans, &stubScopedStock{err: boom}).Execute(context.Background(), damagedInput("", nil, 1))

	assert.ErrorIs(t, err, boom)
	assert.Empty(t, scans.created)
}

// ── deleting a receive row ──────────────────────────────────────────────────

// The real booking: the lot's receive rows were deleted while a damaged row
// still took from that lot.
func TestDeleteBookingScan_refusesToDeleteTheReceiveADamagedRowTakesFrom(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "31/10/2027", 47, 0), stagedDamaged("d1", "31/10/2027", 1, 1),
	}}

	err := newDeleteUC(t, scans, &stubScopedStock{}).Execute(context.Background(), "booking-1", "r1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingScanHasDependents)
	assert.Empty(t, scans.deleted)
}

func TestDeleteBookingScan_allowsDeletingAReceiveWhenAnEarlierOneStillCoversTheLot(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "L1", 10, 0), stagedReceive("r2", "L1", 10, 1), stagedDamaged("d1", "L1", 1, 2),
	}}

	err := newDeleteUC(t, scans, &stubScopedStock{}).Execute(context.Background(), "booking-1", "r2")

	require.NoError(t, err)
	assert.Equal(t, []string{"r2"}, scans.deleted)
}

// Finish replays in scan order: a receive scanned after the damaged row
// would not exist yet when the damaged row is replayed.
func TestDeleteBookingScan_refusesWhenTheRemainingReceiveIsScannedAfterTheDamagedRow(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "L1", 10, 0), stagedDamaged("d1", "L1", 1, 1), stagedReceive("r2", "L1", 10, 2),
	}}

	err := newDeleteUC(t, scans, &stubScopedStock{}).Execute(context.Background(), "booking-1", "r1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingScanHasDependents)
	assert.Empty(t, scans.deleted)
}

func TestDeleteBookingScan_refusesWhenTheRemainingReceiveIsTooSmall(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "L1", 1, 0), stagedReceive("r2", "L1", 10, 1), stagedDamaged("d1", "L1", 5, 2),
	}}

	err := newDeleteUC(t, scans, &stubScopedStock{}).Execute(context.Background(), "booking-1", "r2")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingScanHasDependents)
	assert.Empty(t, scans.deleted)
}

func TestDeleteBookingScan_refusesToDeleteAReceiveIntoARealBatchItsWriteOffNeeds(t *testing.T) {
	batchID := "batch-9"
	into := stagedReceive("r1", "", 3, 0)
	into.BatchID = &batchID
	take := stagedDamaged("d1", "", 2, 1)
	take.BatchID = &batchID
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{into, take}}

	err := newDeleteUC(t, scans, &stubScopedStock{onHand: map[string]int64{batchID: 0}}).Execute(context.Background(), "booking-1", "r1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingScanHasDependents)
	assert.Empty(t, scans.deleted)
}

func TestDeleteBookingScan_deletingAWriteOffNeedsNoStockCheck(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "L1", 10, 0), stagedDamaged("d1", "L1", 1, 1),
	}}
	stock := &stubScopedStock{err: errors.New("must not be called")}

	err := newDeleteUC(t, scans, stock).Execute(context.Background(), "booking-1", "d1")

	require.NoError(t, err)
	assert.Equal(t, []string{"d1"}, scans.deleted)
	assert.Zero(t, stock.calls)
}

func TestDeleteBookingScan_receiveIntoItsOwnUnnamedBatchCanAlwaysBeDeleted(t *testing.T) {
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		stagedReceive("r1", "", 10, 0), stagedDamaged("d1", "", 1, 1),
	}}

	err := newDeleteUC(t, scans, &stubScopedStock{onHand: map[string]int64{"": 1}}).Execute(context.Background(), "booking-1", "r1")

	require.NoError(t, err)
	assert.Equal(t, []string{"r1"}, scans.deleted)
}

func TestDeleteBookingScan_stockLookupFailureIsReturned(t *testing.T) {
	batchID := "batch-9"
	into := stagedReceive("r1", "", 3, 0)
	into.BatchID = &batchID
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{into}}
	boom := errors.New("db down")

	err := newDeleteUC(t, scans, &stubScopedStock{err: boom}).Execute(context.Background(), "booking-1", "r1")

	assert.ErrorIs(t, err, boom)
	assert.Empty(t, scans.deleted)
}
