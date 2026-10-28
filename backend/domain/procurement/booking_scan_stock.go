package procurement

import (
	"sort"
)

// BookingScanSource is where a staged row's stock lives once
// CompleteBookingUseCase replays it.
type BookingScanSource int

const (
	// BookingScanSourceRealBatch: BatchID names a batch that already exists.
	BookingScanSourceRealBatch BookingScanSource = iota
	// BookingScanSourcePendingLot: the batch a receive row in this booking
	// creates for SKUID+LotNumber.
	BookingScanSourcePendingLot
	// BookingScanSourceLedger: a correction/damaged/lost row with neither
	// batch nor lot takes from the general ledger.
	BookingScanSourceLedger
	// BookingScanSourceNewBatch: a receive with neither batch nor lot still
	// creates a batch of its own (see inventory.ReceiveStockUseCase) that no
	// other row can name -- it never adds to the general ledger.
	BookingScanSourceNewBatch
)

func (s BookingScan) Source() BookingScanSource {
	switch {
	case s.BatchID != nil:
		return BookingScanSourceRealBatch
	case s.LotNumber != "":
		return BookingScanSourcePendingLot
	case s.ActionType == BookingScanActionReceive:
		return BookingScanSourceNewBatch
	default:
		return BookingScanSourceLedger
	}
}

// StockDelta is the row's effect on its source's stock: + for a receive,
// - for a correction or a damaged/lost write-off.
func (s BookingScan) StockDelta() int64 {
	switch s.ActionType {
	case BookingScanActionDamaged, BookingScanActionLost:
		return -s.WriteOffQuantity
	default:
		return s.Quantity
	}
}

// CheckPendingLot replays one pending lot's rows in scan order, the same
// order CompleteBookingUseCase does, and fails when a row takes from the lot
// before a receive has created it (ErrPendingBatchNotFound) or takes more
// than the lot holds at that point (ErrPendingLotInsufficient). A lot whose
// receive was already replayed is a real batch now -- real stock, not this
// check, decides what can still be taken from it.
func CheckPendingLot(scans []BookingScan, skuID, lotNumber string) error {
	var lotRows []BookingScan
	for _, s := range scans {
		if s.SKUID != skuID || s.LotNumber != lotNumber {
			continue
		}
		if s.ReplayedAt != nil && s.ActionType == BookingScanActionReceive {
			return nil
		}
		if s.Source() == BookingScanSourcePendingLot {
			lotRows = append(lotRows, s)
		}
	}
	sort.SliceStable(lotRows, func(i, j int) bool { return lotRows[i].ScannedAt.Before(lotRows[j].ScannedAt) })

	created := false
	var balance int64
	for _, s := range lotRows {
		if s.ActionType == BookingScanActionReceive {
			created = true
		} else if !created {
			return ErrPendingBatchNotFound
		}
		balance += s.StockDelta()
		if balance < 0 {
			return ErrPendingLotInsufficient
		}
	}
	return nil
}

// StagedDelta sums the not-yet-replayed effect of this booking's rows on one
// real source: an existing batch, or the general ledger when batchID is nil.
// Added to that source's real stock it is what the source holds once the
// booking finishes.
func StagedDelta(scans []BookingScan, skuID string, batchID *string) int64 {
	var total int64
	for _, s := range scans {
		if s.SKUID != skuID || s.ReplayedAt != nil {
			continue
		}
		switch s.Source() {
		case BookingScanSourceRealBatch:
			if batchID != nil && *s.BatchID == *batchID {
				total += s.StockDelta()
			}
		case BookingScanSourceLedger:
			if batchID == nil {
				total += s.StockDelta()
			}
		}
	}
	return total
}
