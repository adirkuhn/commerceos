package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListStockMovementsUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
}

func NewListStockMovementsUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository) *ListStockMovementsUseCase {
	return &ListStockMovementsUseCase{db: db, movementRepo: movementRepo}
}

func (uc *ListStockMovementsUseCase) Execute(ctx context.Context, skuID string) ([]invDomain.StockMovement, error) {
	var movements []invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		movements, err = uc.movementRepo.ListBySKU(ctx, skuID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listStockMovements: %w", err)
	}
	return movements, nil
}
