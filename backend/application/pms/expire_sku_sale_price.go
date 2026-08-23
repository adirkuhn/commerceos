package pms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ExpireSKUSalePriceUseCase reverts a single SKU's sale back to its
// regular price. Called by cmd/scheduler once per SKU returned by
// SKURepository.ListWithExpiredSale, under that SKU's own tenant context
// (see runOne's comment in cmd/scheduler for why cross-tenant polling
// still dispatches per-item work under the real tenant identity).
type ExpireSKUSalePriceUseCase struct {
	db               *gorm.DB
	variantRepo      pmsDomain.VariantRepository
	skuRepo          pmsDomain.SKURepository
	outboxRepo       outbox.Repository
	priceHistoryRepo pmsDomain.SKUPriceHistoryRepository
}

func NewExpireSKUSalePriceUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository, outboxRepo outbox.Repository, priceHistoryRepo pmsDomain.SKUPriceHistoryRepository) *ExpireSKUSalePriceUseCase {
	return &ExpireSKUSalePriceUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo, outboxRepo: outboxRepo, priceHistoryRepo: priceHistoryRepo}
}

func (uc *ExpireSKUSalePriceUseCase) Execute(ctx context.Context, skuID string) (*pmsDomain.SKU, error) {
	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		oldPriceMinorUnits := existing.PriceMinorUnits
		oldSalePriceMinorUnits := existing.SalePriceMinorUnits

		existing.SalePriceMinorUnits = nil
		existing.SaleEndDate = nil
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}
		sku = *existing

		if err := recordSKUPriceChangeIfNeeded(ctx, uc.priceHistoryRepo, &sku, oldPriceMinorUnits, oldSalePriceMinorUnits, pmsDomain.PriceChangeReasonSaleExpired); err != nil {
			return err
		}

		variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("expireSKUSalePrice: %w", err)
	}
	return &sku, nil
}
