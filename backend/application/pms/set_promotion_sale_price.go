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

// SetPromotionSalePriceUseCase drives a SKU's SalePriceMinorUnits from a
// promotions.Promotion's activation/reversal sweep (see
// application/scheduler.RunPromotionActivationUseCase) -- deliberately
// separate from UpdateSKUUseCase, which is a full-replace of every SKU
// field and would require the caller to round-trip fields it has no
// business touching (see ExpireSKUSalePriceUseCase, the same narrow-mutation
// pattern already used for the sale-price-expiry cron).
type SetPromotionSalePriceUseCase struct {
	db               *gorm.DB
	variantRepo      pmsDomain.VariantRepository
	skuRepo          pmsDomain.SKURepository
	outboxRepo       outbox.Repository
	priceHistoryRepo pmsDomain.SKUPriceHistoryRepository
}

func NewSetPromotionSalePriceUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, skuRepo pmsDomain.SKURepository, outboxRepo outbox.Repository, priceHistoryRepo pmsDomain.SKUPriceHistoryRepository) *SetPromotionSalePriceUseCase {
	return &SetPromotionSalePriceUseCase{db: db, variantRepo: variantRepo, skuRepo: skuRepo, outboxRepo: outboxRepo, priceHistoryRepo: priceHistoryRepo}
}

// Apply overwrites skuID's sale price (a promotion always wins over
// whatever was there, manual or none -- see domain/promotions.PromotionSKU's
// doc) and clears SaleEndDate, since a promotion controls its own duration
// via Promotion.EndsAt, not the SKU's date field. Returns the sale
// price/end date that were in effect immediately before, for the caller
// (application/promotions.ApplyPromotionSKUUseCase) to snapshot on the
// PromotionSKU line so Revert can restore them later.
func (uc *SetPromotionSalePriceUseCase) Apply(ctx context.Context, skuID string, salePriceMinorUnits int64) (previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time, err error) {
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		oldPriceMinorUnits := existing.PriceMinorUnits
		oldSalePriceMinorUnits := existing.SalePriceMinorUnits
		previousSalePriceMinorUnits = existing.SalePriceMinorUnits
		previousSaleEndDate = existing.SaleEndDate

		newSalePrice := salePriceMinorUnits
		existing.SalePriceMinorUnits = &newSalePrice
		existing.SaleEndDate = nil
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}

		if err := recordSKUPriceChangeIfNeeded(ctx, uc.priceHistoryRepo, existing, oldPriceMinorUnits, oldSalePriceMinorUnits, pmsDomain.PriceChangeReasonPromotionApplied); err != nil {
			return err
		}

		variant, err := uc.variantRepo.FindByID(ctx, existing.VariantID)
		if err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("setPromotionSalePrice.Apply: %w", err)
	}
	return previousSalePriceMinorUnits, previousSaleEndDate, nil
}

// Revert restores skuID's sale price/end date to whatever they were before
// a promotion took over (nil/nil if it had no sale at all) -- the mirror
// image of Apply, called once the parent promotion leaves its active
// window.
func (uc *SetPromotionSalePriceUseCase) Revert(ctx context.Context, skuID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		oldPriceMinorUnits := existing.PriceMinorUnits
		oldSalePriceMinorUnits := existing.SalePriceMinorUnits

		existing.SalePriceMinorUnits = previousSalePriceMinorUnits
		existing.SaleEndDate = previousSaleEndDate
		existing.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, existing); err != nil {
			return err
		}

		if err := recordSKUPriceChangeIfNeeded(ctx, uc.priceHistoryRepo, existing, oldPriceMinorUnits, oldSalePriceMinorUnits, pmsDomain.PriceChangeReasonPromotionEnded); err != nil {
			return err
		}

		variant, err := uc.variantRepo.FindByID(ctx, existing.VariantID)
		if err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return fmt.Errorf("setPromotionSalePrice.Revert: %w", err)
	}
	return nil
}
