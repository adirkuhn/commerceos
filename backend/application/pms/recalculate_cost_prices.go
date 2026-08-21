package pms

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// batchListerForCostPrice is the narrow slice of *inventory.ListBatchesUseCase
// this use case needs (ISP) -- going through inventory's own use case, not
// its BatchRepository directly, is what keeps this pms use case from taking
// a cross-domain repository dependency (see CLAUDE.md's "cross-domain calls
// go through use cases" rule).
type batchListerForCostPrice interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}

// PurchaseCostLine mirrors procurement.SKUPurchaseCostLine -- a local copy,
// not an import of application/procurement, because that package already
// imports application/pms (create_product_and_link_line_item.go), so the
// reverse import here would be a cycle. cmd/scheduler's wiring adapts
// *procurement.ListReceivedCostLinesForSKUUseCase's real result into this
// shape when constructing RecalculateCostPricesUseCase.
type PurchaseCostLine struct {
	PurchaseOrderLineItemID  string
	Quantity                 int64
	LandedUnitCostMinorUnits int64
}

// purchaseCostLineListerForCostPrice is the narrow slice of
// *procurement.ListReceivedCostLinesForSKUUseCase this use case needs (ISP)
// -- same cross-domain-via-use-case rationale as batchListerForCostPrice.
// Its results cover received/partially_received POs that never created a
// Batch (see that use case's doc comment); Execute below excludes any line
// item a real Batch already accounts for, so the two sources never
// double-count the same receipt.
type purchaseCostLineListerForCostPrice interface {
	Execute(ctx context.Context, skuID string) ([]PurchaseCostLine, error)
}

// RecalculateCostPricesResult reports what happened for one SKU.
type RecalculateCostPricesResult struct {
	// Skipped is true when the SKU has no on-hand batch with a cost basis
	// (LandedCostMinorUnits set) to average -- nothing was written.
	Skipped                    bool
	CostPriceMinorUnits        int64
	RecommendedPriceMinorUnits int64
}

// RecalculateCostPricesUseCase is the daily cost-price cron's per-SKU unit
// of work (see cmd/scheduler.runRecalculateCostPrices for the cross-tenant
// dispatch loop) -- one call, one transaction, same shape as
// ExpireSKUSalePriceUseCase/ExpireBatchUseCase so a single SKU's failure
// can't affect any other's.
//
// Cost price is the weighted average of inventory.Batch.LandedCostMinorUnits
// across every batch of this SKU still holding remaining quantity (weight =
// that batch's remaining quantity) -- batches with no captured cost (nil
// LandedCostMinorUnits, e.g. anything received before landed-cost capture
// existed, or a manual/POS stock-in with no PO behind it) are excluded from
// both the numerator and the weight, not treated as zero. A SKU with no
// cost-bearing on-hand batch at all is Skipped, not zeroed.
//
// Recommended price is computed from the effective target margin (the
// SKU's product's category override, falling back to the tenant default --
// see EffectiveTargetMarginBasisPoints) and written alongside cost price to
// today's SKUCostPriceHistory row (upserted, so a same-day rerun is a
// no-op-that-overwrites rather than a duplicate).
type RecalculateCostPricesUseCase struct {
	db                 *gorm.DB
	skuRepo            pmsDomain.SKURepository
	variantRepo        pmsDomain.VariantRepository
	productRepo        pmsDomain.ProductRepository
	categoryRepo       pmsDomain.CategoryRepository
	settingsRepo       settingsDomain.Repository
	historyRepo        pmsDomain.SKUCostPriceHistoryRepository
	batchLister        batchListerForCostPrice
	purchaseCostLister purchaseCostLineListerForCostPrice
}

func NewRecalculateCostPricesUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
	settingsRepo settingsDomain.Repository,
	historyRepo pmsDomain.SKUCostPriceHistoryRepository,
	batchLister batchListerForCostPrice,
	purchaseCostLister purchaseCostLineListerForCostPrice,
) *RecalculateCostPricesUseCase {
	return &RecalculateCostPricesUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
		categoryRepo: categoryRepo, settingsRepo: settingsRepo, historyRepo: historyRepo,
		batchLister: batchLister, purchaseCostLister: purchaseCostLister,
	}
}

func (uc *RecalculateCostPricesUseCase) Execute(ctx context.Context, skuID string) (*RecalculateCostPricesResult, error) {
	var result RecalculateCostPricesResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		if sku.CostPriceSource == pmsDomain.SKUCostPriceSourceManual {
			// A human set this SKU's cost price directly (see
			// SetManualCostPriceUseCase) -- it sticks until
			// ClearCostPriceOverrideUseCase runs, so there's nothing for the
			// cron to average or overwrite.
			result.Skipped = true
			return nil
		}

		batches, err := uc.batchLister.Execute(ctx, skuID)
		if err != nil {
			return err
		}
		var totalQty, weightedCost int64
		// linkedLineItems tracks every PO line item a real Batch already
		// references (regardless of whether that batch itself has a cost
		// basis) -- purchaseCostLister below must skip these, or a properly
		// booked receipt would be counted twice.
		linkedLineItems := make(map[string]bool, len(batches))
		for _, b := range batches {
			if b.PurchaseOrderLineItemID != nil {
				linkedLineItems[*b.PurchaseOrderLineItemID] = true
			}
			if b.Quantity <= 0 || b.LandedCostMinorUnits == nil {
				continue
			}
			totalQty += b.Quantity
			weightedCost += b.Quantity * *b.LandedCostMinorUnits
		}

		poCostLines, err := uc.purchaseCostLister.Execute(ctx, skuID)
		if err != nil {
			return err
		}
		for _, l := range poCostLines {
			if linkedLineItems[l.PurchaseOrderLineItemID] || l.Quantity <= 0 {
				continue
			}
			totalQty += l.Quantity
			weightedCost += l.Quantity * l.LandedUnitCostMinorUnits
		}

		if totalQty <= 0 {
			result.Skipped = true
			return nil
		}
		costPrice := (weightedCost + totalQty/2) / totalQty

		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		category := resolveCategory(ctx, uc.variantRepo, uc.productRepo, uc.categoryRepo, sku.VariantID)
		margin := EffectiveTargetMarginBasisPoints(category, settings.DefaultTargetMarginBasisPoints)
		recommended := RecommendedPriceMinorUnits(costPrice, margin)

		if err := uc.skuRepo.UpdateCostPrice(ctx, skuID, costPrice); err != nil {
			return err
		}
		now := time.Now()
		if err := uc.historyRepo.Upsert(ctx, &pmsDomain.SKUCostPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			CostPriceMinorUnits: costPrice, RecommendedPriceMinorUnits: recommended,
			OccurredOn: now.Truncate(24 * time.Hour), CreatedAt: now,
		}); err != nil {
			return err
		}
		result.CostPriceMinorUnits = costPrice
		result.RecommendedPriceMinorUnits = recommended
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("recalculateCostPrices: %w", err)
	}
	return &result, nil
}

// resolveCategory walks SKU -> Variant -> Product -> Category, returning
// nil (not an error) at any broken link -- a product with no category, or a
// dangling reference, just means "use the tenant default margin", not a
// reason to fail the whole recompute.
func resolveCategory(ctx context.Context, variantRepo pmsDomain.VariantRepository, productRepo pmsDomain.ProductRepository, categoryRepo pmsDomain.CategoryRepository, variantID string) *pmsDomain.Category {
	variant, err := variantRepo.FindByID(ctx, variantID)
	if err != nil || variant == nil {
		return nil
	}
	product, err := productRepo.FindByID(ctx, variant.ProductID)
	if err != nil || product == nil || product.CategoryID == nil {
		return nil
	}
	category, err := categoryRepo.FindByID(ctx, *product.CategoryID)
	if err != nil {
		return nil
	}
	return category
}
