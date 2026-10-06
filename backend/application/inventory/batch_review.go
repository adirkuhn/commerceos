package inventory

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

// How an expired batch left the review list -- data.outcome of
// inventory.batch.reviewed.
const (
	BatchReviewOutcomeDismissed  = "dismissed"
	BatchReviewOutcomeWrittenOff = "written_off"
	BatchReviewOutcomeDiscarded  = "discarded"
)

// markBatchReviewed takes a batch off the expiry review list and records
// how. Must run inside tenant.WithTenant.
func markBatchReviewed(ctx context.Context, batchRepo invDomain.BatchRepository, outbox eventrecord.Enqueuer, batchID, outcome string) error {
	batch, err := batchRepo.FindByID(ctx, batchID)
	if err != nil {
		return err
	}
	if err := batchRepo.MarkReviewed(ctx, batchID); err != nil {
		return err
	}
	return eventrecord.Record(ctx, outbox, events.BatchReviewed, batchID, eventrecord.Data{
		"sku_id": batch.SKUID, "warehouse_id": batch.WarehouseID, "outcome": outcome,
	})
}
