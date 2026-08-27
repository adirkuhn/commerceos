package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateBatchInput struct {
	ID          string
	WarehouseID string
	LotNumber   string
	ExpiryDate  *time.Time
}

type UpdateBatchUseCase struct {
	db         *gorm.DB
	batchRepo  invDomain.BatchRepository
	outboxRepo outbox.Repository
}

func NewUpdateBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, outboxRepo outbox.Repository) *UpdateBatchUseCase {
	return &UpdateBatchUseCase{db: db, batchRepo: batchRepo, outboxRepo: outboxRepo}
}

// Execute updates a batch's warehouse, lot number, and expiry date. SKUID
// and ReceivedAt are deliberately not editable here — see
// BatchRepository.Update's doc comment; moving a batch to a different SKU
// is effectively a different batch (delete + create).
func (uc *UpdateBatchUseCase) Execute(ctx context.Context, in UpdateBatchInput) (*invDomain.Batch, error) {
	var batch invDomain.Batch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.batchRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.WarehouseID = in.WarehouseID
		existing.LotNumber = in.LotNumber
		existing.ExpiryDate = in.ExpiryDate
		existing.UpdatedAt = time.Now()
		if err := uc.batchRepo.Update(ctx, existing); err != nil {
			return err
		}
		batch = *existing

		// LotNumber/ExpiryDate are exactly what SyncStockToChannelsUseCase
		// pushes to Woo as lot_number/expiry_date (toWooMetaData) -- a
		// correction here (no quantity change) must still reach channels,
		// same as any other stock-affecting edit, or the pushed lot/expiry
		// silently goes stale until an unrelated stock movement happens to
		// touch the same batch.
		return enqueueStockChanged(ctx, uc.outboxRepo, batch.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateBatch: %w", err)
	}
	return &batch, nil
}
