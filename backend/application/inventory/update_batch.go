package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
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
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewUpdateBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *UpdateBatchUseCase {
	return &UpdateBatchUseCase{db: db, batchRepo: batchRepo}
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateBatch: %w", err)
	}
	return &batch, nil
}
