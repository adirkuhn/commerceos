package inventory

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetStockLevelsUseCase is GetSKUStockLevelUseCase for a known set of SKUs
// (e.g. one storefront listing page): two bulk queries for all of them,
// each computed with the same shared computeStockLevel rule.
type GetStockLevelsUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
	settingsRepo settingsGetter
}

func NewGetStockLevelsUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository, settingsRepo settingsGetter) *GetStockLevelsUseCase {
	return &GetStockLevelsUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo, settingsRepo: settingsRepo}
}

// Execute returns a level for every requested SKU id (zero when it has no
// stock history).
func (uc *GetStockLevelsUseCase) Execute(ctx context.Context, skuIDs []string) (map[string]StockLevel, error) {
	levels := make(map[string]StockLevel, len(skuIDs))
	if len(skuIDs) == 0 {
		return levels, nil
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		movementsBySKU := make(map[string][]invDomain.StockMovement, len(skuIDs))
		for _, m := range movements {
			movementsBySKU[m.SKUID] = append(movementsBySKU[m.SKUID], m)
		}
		batchesBySKU := make(map[string][]invDomain.Batch, len(skuIDs))
		for _, b := range batches {
			batchesBySKU[b.SKUID] = append(batchesBySKU[b.SKUID], b)
		}
		for _, id := range skuIDs {
			levels[id] = computeStockLevel(id, movementsBySKU[id], batchesBySKU[id], settings.InventorySyncRulesEnabled)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getStockLevels: %w", err)
	}
	return levels, nil
}
