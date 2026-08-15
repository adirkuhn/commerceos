package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListBatchMovementsUseCase is ListStockMovementsUseCase's per-batch
// counterpart -- the SKU-wide ledger view mixes every batch's movements
// together, this scopes to just one lot.
type ListBatchMovementsUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
}

func NewListBatchMovementsUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository) *ListBatchMovementsUseCase {
	return &ListBatchMovementsUseCase{db: db, movementRepo: movementRepo}
}

func (uc *ListBatchMovementsUseCase) Execute(ctx context.Context, batchID string) ([]invDomain.StockMovement, error) {
	var movements []invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		movements, err = uc.movementRepo.ListByBatch(ctx, batchID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listBatchMovements: %w", err)
	}
	return movements, nil
}
