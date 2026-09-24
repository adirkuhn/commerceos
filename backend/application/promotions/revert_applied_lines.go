package promotions

import (
	"context"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

// revertAppliedLines restores the pre-promotion sale price of every line
// currently live on its SKU. It must run before those lines are deleted
// (UpdatePromotionUseCase replaces them, DeletePromotionUseCase cascades):
// the snapshot of the SKU's old price lives only on the line, so deleting an
// applied line first would leave the SKU discounted for good. Stops at the
// first failure so the caller can abort before touching the lines; a line
// already reverted by then is simply re-applied by the promotion_activation
// sweep if the promotion is still running. Must run outside
// tenant.WithTenant -- each revert opens its own.
func revertAppliedLines(ctx context.Context, reverter lineReverter, lines []promotionsDomain.PromotionSKU) error {
	for _, line := range lines {
		if !line.Applied {
			continue
		}
		if err := reverter.Execute(ctx, line); err != nil {
			return err
		}
	}
	return nil
}

// clearanceBatchIDs is the set of batches lines already sell off -- see
// validateClearanceBatches' alreadyOnPromotion.
func clearanceBatchIDs(lines []promotionsDomain.PromotionSKU) map[string]bool {
	ids := map[string]bool{}
	for _, l := range lines {
		if l.BatchID != nil {
			ids[*l.BatchID] = true
		}
	}
	return ids
}
