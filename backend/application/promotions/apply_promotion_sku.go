package promotions

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuPriceLookup narrows pms.SKURepository to the one method this package
// needs (ISP) -- a plain read of the SKU's current regular price to
// compute a promotion's discount from, same pattern as
// application/reports' skuCostPriceLookup.
type skuPriceLookup interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

// setPromotionSalePrice narrows application/pms.SetPromotionSalePriceUseCase
// to the two methods this package needs (ISP).
type setPromotionSalePrice interface {
	Apply(ctx context.Context, skuID string, salePriceMinorUnits int64) (previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time, err error)
	Revert(ctx context.Context, skuID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error
}

// computePromotionSalePrice applies line's discount to regularPriceMinorUnits,
// clamped to a minimum of 1 minor unit -- a promotion must never produce a
// free or negative sale price, regardless of how a fixed_amount discount
// compares to the SKU's actual price. A fixed_price never raises the price:
// it is capped at the regular price.
func computePromotionSalePrice(regularPriceMinorUnits int64, discountType promotionsDomain.PromotionType, discountValue int64) int64 {
	var discounted int64
	switch discountType {
	case promotionsDomain.PromotionTypePercentage:
		discounted = regularPriceMinorUnits - (regularPriceMinorUnits*discountValue)/10000
	case promotionsDomain.PromotionTypeFixedAmount:
		discounted = regularPriceMinorUnits - discountValue
	case promotionsDomain.PromotionTypeFixedPrice:
		discounted = min(discountValue, regularPriceMinorUnits)
	}
	if discounted < 1 {
		discounted = 1
	}
	return discounted
}

// ApplyPromotionSKUUseCase pushes one PromotionSKU line's discount onto its
// SKU's sale price -- one line at a time so
// application/scheduler.RunPromotionActivationUseCase's cross-tenant sweep
// can log-and-continue on a single line's failure without aborting the
// whole run.
type ApplyPromotionSKUUseCase struct {
	db           *gorm.DB
	skuLookup    skuPriceLookup
	setSalePrice setPromotionSalePrice
	lineRepo     promotionsDomain.PromotionSKURepository
	outbox       eventrecord.Enqueuer
}

func NewApplyPromotionSKUUseCase(db *gorm.DB, skuLookup skuPriceLookup, setSalePrice setPromotionSalePrice, lineRepo promotionsDomain.PromotionSKURepository, outbox eventrecord.Enqueuer) *ApplyPromotionSKUUseCase {
	return &ApplyPromotionSKUUseCase{db: db, skuLookup: skuLookup, setSalePrice: setSalePrice, lineRepo: lineRepo, outbox: outbox}
}

func (uc *ApplyPromotionSKUUseCase) Execute(ctx context.Context, line promotionsDomain.PromotionSKU) error {
	var priceMinorUnits int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuLookup.FindByID(ctx, line.SKUID)
		if err != nil {
			return err
		}
		priceMinorUnits = sku.PriceMinorUnits
		return nil
	})
	if err != nil {
		return fmt.Errorf("applyPromotionSKU: %w", err)
	}

	newSalePrice := computePromotionSalePrice(priceMinorUnits, line.DiscountType, line.DiscountValue)
	previousSalePrice, previousSaleEndDate, err := uc.setSalePrice.Apply(ctx, line.SKUID, newSalePrice)
	if err != nil {
		return fmt.Errorf("applyPromotionSKU: %w", err)
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lineRepo.MarkApplied(ctx, line.ID, previousSalePrice, previousSaleEndDate); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PromotionLineApplied, line.ID, eventrecord.Data{
			"promotion_id": line.PromotionID, "sku_id": line.SKUID,
			"sale_price_minor_units": newSalePrice, "previous_sale_price_minor_units": previousSalePrice,
		})
	})
	if err != nil {
		return fmt.Errorf("applyPromotionSKU: %w", err)
	}
	return nil
}

// RevertPromotionSKUUseCase restores a PromotionSKU line's SKU to whatever
// sale price it had before the promotion applied -- no SKU price lookup
// needed, everything required is already snapshotted on the line itself.
type RevertPromotionSKUUseCase struct {
	db           *gorm.DB
	setSalePrice setPromotionSalePrice
	lineRepo     promotionsDomain.PromotionSKURepository
	outbox       eventrecord.Enqueuer
}

func NewRevertPromotionSKUUseCase(db *gorm.DB, setSalePrice setPromotionSalePrice, lineRepo promotionsDomain.PromotionSKURepository, outbox eventrecord.Enqueuer) *RevertPromotionSKUUseCase {
	return &RevertPromotionSKUUseCase{db: db, setSalePrice: setSalePrice, lineRepo: lineRepo, outbox: outbox}
}

func (uc *RevertPromotionSKUUseCase) Execute(ctx context.Context, line promotionsDomain.PromotionSKU) error {
	if err := uc.setSalePrice.Revert(ctx, line.SKUID, line.PreviousSalePriceMinorUnits, line.PreviousSaleEndDate); err != nil {
		return fmt.Errorf("revertPromotionSKU: %w", err)
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lineRepo.MarkReverted(ctx, line.ID); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PromotionLineReverted, line.ID, eventrecord.Data{
			"promotion_id": line.PromotionID, "sku_id": line.SKUID,
			"restored_sale_price_minor_units": line.PreviousSalePriceMinorUnits,
		})
	})
	if err != nil {
		return fmt.Errorf("revertPromotionSKU: %w", err)
	}
	return nil
}
