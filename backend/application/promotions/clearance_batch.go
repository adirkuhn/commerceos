package promotions

import (
	"context"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

// skuBatchLister narrows application/inventory.ListBatchesUseCase to the one
// method this package needs (ISP) -- the cross-domain read goes through
// inventory's own use case, never its repository, because a batch's
// quantity only exists as a sum over the stock-movement ledger that use case
// already computes. It opens its own tenant.WithTenant, so callers must
// invoke it outside of theirs.
type skuBatchLister interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}

// clearanceBatchFor maps batchID's entry in batches (one SKU's full batch
// list) into promotions' own ClearanceBatch view -- Found is false when the
// batch is no longer listed for that SKU (deleted, or never belonged to it).
func clearanceBatchFor(batches []invDomain.Batch, batchID string) promotionsDomain.ClearanceBatch {
	for _, b := range batches {
		if b.IsGeneralLedger || b.ID != batchID {
			continue
		}
		return promotionsDomain.ClearanceBatch{
			Found: true, Quantity: b.Quantity, ExpiryDate: b.ExpiryDate,
			MarkedExpired: b.Status == invDomain.BatchStatusExpired,
		}
	}
	return promotionsDomain.ClearanceBatch{}
}
