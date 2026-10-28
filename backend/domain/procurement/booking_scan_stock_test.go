package procurement_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

var scanT0 = time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)

func at(minute int) time.Time { return scanT0.Add(time.Duration(minute) * time.Minute) }

func receive(lot string, qty int64, minute int) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{SKUID: "sku-1", LotNumber: lot, ActionType: procurementDomain.BookingScanActionReceive, Quantity: qty, ScannedAt: at(minute)}
}

func damaged(lot string, qty int64, minute int) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{SKUID: "sku-1", LotNumber: lot, ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: qty, ScannedAt: at(minute)}
}

func correction(lot string, qty int64, minute int) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{SKUID: "sku-1", LotNumber: lot, ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -qty, ScannedAt: at(minute)}
}

func TestBookingScanStockDelta(t *testing.T) {
	assert.Equal(t, int64(5), receive("L1", 5, 0).StockDelta())
	assert.Equal(t, int64(-2), correction("L1", 2, 0).StockDelta())
	assert.Equal(t, int64(-3), damaged("L1", 3, 0).StockDelta())
	lost := damaged("L1", 4, 0)
	lost.ActionType = procurementDomain.BookingScanActionLost
	assert.Equal(t, int64(-4), lost.StockDelta())
}

func TestCheckPendingLot_takeAfterReceiveWithinQuantityIsFine(t *testing.T) {
	scans := []procurementDomain.BookingScan{receive("31/10/2027", 47, 0), damaged("31/10/2027", 1, 1), correction("31/10/2027", 2, 2)}

	assert.NoError(t, procurementDomain.CheckPendingLot(scans, "sku-1", "31/10/2027"))
}

// Prod 2026-10-28: the lot's receive rows were deleted, the damaged row
// stayed behind and Finish failed on every retry.
func TestCheckPendingLot_takeWithNoReceiveIsPendingBatchNotFound(t *testing.T) {
	scans := []procurementDomain.BookingScan{damaged("31/10/2027", 1, 1)}

	assert.ErrorIs(t, procurementDomain.CheckPendingLot(scans, "sku-1", "31/10/2027"), procurementDomain.ErrPendingBatchNotFound)
}

func TestCheckPendingLot_receiveScannedAfterTheTakeDoesNotCount(t *testing.T) {
	scans := []procurementDomain.BookingScan{damaged("L1", 1, 1), receive("L1", 10, 5)}

	assert.ErrorIs(t, procurementDomain.CheckPendingLot(scans, "sku-1", "L1"), procurementDomain.ErrPendingBatchNotFound)
}

func TestCheckPendingLot_unorderedInputIsCheckedInScanOrder(t *testing.T) {
	scans := []procurementDomain.BookingScan{damaged("L1", 1, 5), receive("L1", 10, 1)}

	assert.NoError(t, procurementDomain.CheckPendingLot(scans, "sku-1", "L1"))
}

func TestCheckPendingLot_takingMoreThanReceivedIsInsufficient(t *testing.T) {
	scans := []procurementDomain.BookingScan{receive("L1", 3, 0), damaged("L1", 2, 1), correction("L1", 2, 2)}

	assert.ErrorIs(t, procurementDomain.CheckPendingLot(scans, "sku-1", "L1"), procurementDomain.ErrPendingLotInsufficient)
}

func TestCheckPendingLot_ignoresOtherLotsAndSKUsAndRealBatches(t *testing.T) {
	otherSKU := receive("L1", 10, 0)
	otherSKU.SKUID = "sku-2"
	realBatch := "batch-9"
	sameLotRealBatch := damaged("L1", 5, 2)
	sameLotRealBatch.BatchID = &realBatch
	scans := []procurementDomain.BookingScan{otherSKU, receive("L2", 10, 0), receive("L1", 1, 1), sameLotRealBatch, damaged("L1", 1, 3)}

	assert.NoError(t, procurementDomain.CheckPendingLot(scans, "sku-1", "L1"))
}

// After a failed Finish the lot's receive is already a real batch: what
// remains is checked against real stock, not here.
func TestCheckPendingLot_lotAlreadyMadeRealByAReplayedReceiveIsNotCheckedHere(t *testing.T) {
	replayedAt := at(10)
	batchID := "batch-real"
	done := receive("L1", 1, 0)
	done.ReplayedAt, done.BatchID = &replayedAt, &batchID
	scans := []procurementDomain.BookingScan{done, damaged("L1", 5, 1)}

	assert.NoError(t, procurementDomain.CheckPendingLot(scans, "sku-1", "L1"))
}

func TestStagedDelta_sumsUnreplayedRowsOnOneRealSource(t *testing.T) {
	batchA, batchB := "batch-a", "batch-b"
	replayedAt := at(10)
	intoA := receive("", 4, 0)
	intoA.BatchID = &batchA
	takeA := damaged("", 1, 1)
	takeA.BatchID = &batchA
	doneTakeA := damaged("", 7, 2)
	doneTakeA.BatchID, doneTakeA.ReplayedAt = &batchA, &replayedAt
	takeB := damaged("", 2, 3)
	takeB.BatchID = &batchB
	ledgerTake := damaged("", 1, 4)
	pendingLotTake := damaged("L1", 1, 5)
	newBatchReceive := receive("", 9, 6)
	scans := []procurementDomain.BookingScan{intoA, takeA, doneTakeA, takeB, ledgerTake, pendingLotTake, newBatchReceive}

	assert.Equal(t, int64(3), procurementDomain.StagedDelta(scans, "sku-1", &batchA))
	assert.Equal(t, int64(-2), procurementDomain.StagedDelta(scans, "sku-1", &batchB))
	assert.Equal(t, int64(-1), procurementDomain.StagedDelta(scans, "sku-1", nil), "a receive without a batch makes a new batch, never general ledger stock")
	assert.Equal(t, int64(0), procurementDomain.StagedDelta(scans, "sku-2", nil))
}

func TestBookingScanSource(t *testing.T) {
	batchID := "batch-a"
	onBatch := damaged("L1", 1, 0)
	onBatch.BatchID = &batchID

	assert.Equal(t, procurementDomain.BookingScanSourceRealBatch, onBatch.Source())
	assert.Equal(t, procurementDomain.BookingScanSourcePendingLot, damaged("L1", 1, 0).Source())
	assert.Equal(t, procurementDomain.BookingScanSourceLedger, damaged("", 1, 0).Source())
	assert.Equal(t, procurementDomain.BookingScanSourcePendingLot, receive("L1", 1, 0).Source())
	assert.Equal(t, procurementDomain.BookingScanSourceNewBatch, receive("", 1, 0).Source())
}
