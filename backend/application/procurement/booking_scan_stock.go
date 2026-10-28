package procurement

import (
	"context"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// scopedStockReader is *inventory.GetScopedStockUseCase: one SKU's real
// on-hand stock in a warehouse and batch scope (nil batchID = general
// ledger). It opens its own transaction, so call it outside
// tenant.WithTenant.
type scopedStockReader interface {
	Execute(ctx context.Context, skuID, warehouseID string, batchID *string) (int64, error)
}

// checkBookingScanSource reports whether the source scan stands for -- a
// pending lot, an existing batch or the general ledger -- still has enough
// stock once every staged row in scans is replayed. Same rules
// CompleteBookingUseCase meets at Finish, checked early so a bad row is
// rejected while the scanner can still fix it.
func checkBookingScanSource(ctx context.Context, stock scopedStockReader, warehouseID string, scans []procurementDomain.BookingScan, scan procurementDomain.BookingScan) error {
	switch scan.Source() {
	case procurementDomain.BookingScanSourcePendingLot:
		return procurementDomain.CheckPendingLot(scans, scan.SKUID, scan.LotNumber)
	case procurementDomain.BookingScanSourceRealBatch, procurementDomain.BookingScanSourceLedger:
		onHand, err := stock.Execute(ctx, scan.SKUID, warehouseID, scan.BatchID)
		if err != nil {
			return err
		}
		if onHand+procurementDomain.StagedDelta(scans, scan.SKUID, scan.BatchID) < 0 {
			return invDomain.ErrInsufficientStock
		}
	}
	return nil
}
