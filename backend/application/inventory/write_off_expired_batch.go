package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// damagedOrLostRecorder is the one method this use case needs from
// MarkDamagedOrLostUseCase (ISP). Called as a plain sequential operation
// between the two tenant.WithTenant blocks below -- it opens its own
// transaction via the same shared *gorm.DB, so it can't be nested inside
// this use case's own transaction closure (same reasoning as
// application/channels's stockLevelGetter).
type damagedOrLostRecorder interface {
	Execute(ctx context.Context, in MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error)
}

// WriteOffExpiredBatchUseCase is the review queue's "Write-off" action --
// the one place an expired-batch review action actually moves stock: it
// records a Lost movement for whatever quantity the batch still has, then
// marks the batch reviewed. If the batch is already at zero (e.g. sold
// through before anyone got to the queue), no movement is created -- there
// is nothing to write off -- but the batch is still marked reviewed.
type WriteOffExpiredBatchUseCase struct {
	db                *gorm.DB
	batchRepo         invDomain.BatchRepository
	movementRepo      invDomain.StockMovementRepository
	markDamagedOrLost damagedOrLostRecorder
}

func NewWriteOffExpiredBatchUseCase(
	db *gorm.DB,
	batchRepo invDomain.BatchRepository,
	movementRepo invDomain.StockMovementRepository,
	markDamagedOrLost damagedOrLostRecorder,
) *WriteOffExpiredBatchUseCase {
	return &WriteOffExpiredBatchUseCase{db: db, batchRepo: batchRepo, movementRepo: movementRepo, markDamagedOrLost: markDamagedOrLost}
}

func (uc *WriteOffExpiredBatchUseCase) Execute(ctx context.Context, batchID string, performedBy string) error {
	var batch invDomain.Batch
	var quantity int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		b, err := uc.batchRepo.FindByID(ctx, batchID)
		if err != nil {
			return err
		}
		batch = *b

		movements, err := uc.movementRepo.ListByBatch(ctx, batchID)
		if err != nil {
			return err
		}
		for _, m := range movements {
			quantity += m.QuantityDelta
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("writeOffExpiredBatch: %w", err)
	}

	if quantity > 0 {
		if _, _, err := uc.markDamagedOrLost.Execute(ctx, MarkDamagedOrLostInput{
			SKUID: batch.SKUID, WarehouseID: batch.WarehouseID, BatchID: &batchID,
			Quantity: quantity, Type: invDomain.MovementTypeLost,
			Reason: "expired batch write-off", OccurredAt: time.Now(), DiscardedBy: performedBy,
		}); err != nil {
			return fmt.Errorf("writeOffExpiredBatch: %w", err)
		}
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.batchRepo.MarkReviewed(ctx, batchID)
	})
	if err != nil {
		return fmt.Errorf("writeOffExpiredBatch: %w", err)
	}
	return nil
}
