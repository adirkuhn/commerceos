package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// settingsGetter is the one method this use case needs from
// settings.Repository (ISP) — whether the tenant has Inventory Rules
// enabled decides which computeStockLevel branch runs.
type settingsGetter interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type GetSKUStockLevelUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
	settingsRepo settingsGetter
}

func NewGetSKUStockLevelUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository, settingsRepo settingsGetter) *GetSKUStockLevelUseCase {
	return &GetSKUStockLevelUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo, settingsRepo: settingsRepo}
}

func (uc *GetSKUStockLevelUseCase) Execute(ctx context.Context, skuID string) (*StockLevel, error) {
	var level StockLevel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		level = computeStockLevel(skuID, movements, batches, settings.InventorySyncRulesEnabled)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSKUStockLevel: %w", err)
	}
	return &level, nil
}
