package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetScopedStockUseCase returns one SKU's on-hand quantity in a single
// warehouse and batch scope (nil batchID = general/un-lotted stock) -- the
// same figure a removal is checked against (see requireSufficientStock),
// exposed so a caller composing several removals (e.g.
// application/pms.PackWeightPortionsUseCase) can check the total up front
// instead of failing halfway through.
type GetScopedStockUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
}

func NewGetScopedStockUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository) *GetScopedStockUseCase {
	return &GetScopedStockUseCase{db: db, movementRepo: movementRepo}
}

func (uc *GetScopedStockUseCase) Execute(ctx context.Context, skuID, warehouseID string, batchID *string) (int64, error) {
	var total int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		total = currentStock(movements, warehouseID, batchID)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("getScopedStock: %w", err)
	}
	return total, nil
}
