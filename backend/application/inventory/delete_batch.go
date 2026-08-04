package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteBatchUseCase struct {
	db           *gorm.DB
	batchRepo    invDomain.BatchRepository
	movementRepo invDomain.StockMovementRepository
}

func NewDeleteBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, movementRepo invDomain.StockMovementRepository) *DeleteBatchUseCase {
	return &DeleteBatchUseCase{db: db, batchRepo: batchRepo, movementRepo: movementRepo}
}

// Execute deletes a batch that has never had any stock recorded against it
// (e.g. created by mistake, wrong SKU/warehouse). A batch with any
// movement history is rejected — the ledger is append-only and a batch
// with movements is load-bearing for past stock numbers, even if its
// current balance is zero.
func (uc *DeleteBatchUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.batchRepo.FindByID(ctx, id); err != nil {
			return err
		}
		movements, err := uc.movementRepo.ListByBatch(ctx, id)
		if err != nil {
			return err
		}
		if len(movements) > 0 {
			return invDomain.ErrBatchHasMovements
		}
		return uc.batchRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteBatch: %w", err)
	}
	return nil
}
