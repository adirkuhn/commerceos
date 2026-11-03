package pms

//events:none daily cost/recommended-price history snapshot (report data); the SKU itself is not changed.

import (
	"context"
	"fmt"
	"log"
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
	// Skipped is true when there is nothing to reconcile: cost_price_source
	// is manual (a human override -- see SetManualCostPriceUseCase), or the
	// SKU has no cost_price_minor_units at all yet (cost_price_source
	// unknown -- see pms.SKUCostPriceSourceUnknown, resolved by the first
	// real receipt, not by this job).
	Skipped bool
	// CostPriceMinorUnits is the SKU's actual, current cost -- written
	// incrementally at receipt time by ApplyReceiptCostUseCase. This job
	// never writes it; it only checks it against ExpectedCostPriceMinorUnits.
	CostPriceMinorUnits        int64
	RecommendedPriceMinorUnits int64
	// InventoryCostMinorUnits is the SKU's current moving average (see
	// pmsDomain.SKU.ActualCostMinorUnits) -- what Reconciled checks.
	InventoryCostMinorUnits int64
	// Reconciled is false when the independently-derived weighted average
	// of on-hand batches/open PO lines (ExpectedCostPriceMinorUnits)
	// disagrees with InventoryCostMinorUnits -- logged as a drift warning
	// (Execute never auto-corrects; see the type's doc comment).
	Reconciled                  bool
	ExpectedCostPriceMinorUnits int64
}

// RecalculateCostPricesUseCase is the daily reconciliation cron's per-SKU
// unit of work (see cmd/scheduler.runRecalculateCostPrices for the
// cross-tenant dispatch loop) -- one call, one transaction, same shape as
// ExpireSKUSalePriceUseCase/ExpireBatchUseCase so a single SKU's failure
// can't affect any other's.
//
// Despite the name (kept to avoid an unrelated rename ripple across ~14
// files), this no longer computes or writes cost_price_minor_units -- that
// is now ApplyReceiptCostUseCase's job, done incrementally (true Moving
// Weighted Average Cost) at every goods receipt, not recomputed nightly
// from a batch snapshot. This job is now a SAFETY NET: it independently
// re-derives what the weighted average SHOULD be from on-hand
// inventory.Batch.LandedCostMinorUnits (weight = remaining quantity;
// batches with no captured cost are excluded from both numerator and
// weight, not treated as zero) plus any received/open PO line not yet
// backed by a real Batch, and compares that to the SKU's actual current
// cost_price_minor_units. A mismatch is logged as a drift warning --
// evidence the incremental path and the ledger have diverged (a bug, or a
// manual DB fix) -- never silently auto-corrected.
//
// Recommended price is computed from the SKU's actual current cost (not
// the reconciliation figure) via the effective target markup (the SKU's
// product's category override, falling back to the tenant default -- see
// EffectiveTargetMarkupBasisPoints), and written alongside it to today's
// SKUCostPriceHistory row (upserted, so a same-day rerun is a
// no-op-that-overwrites rather than a duplicate) -- this part is unchanged,
// still the margin-trend-over-time chart's data source.
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
			// SetManualCostPriceUseCase) -- intentionally different from
			// whatever the ledger would derive, so there's nothing to
			// reconcile against.
			result.Skipped = true
			return nil
		}
		if sku.CostPriceMinorUnits == nil || sku.ActualCostMinorUnits() == nil {
			// cost_price_source unknown -- no receipt has given this SKU a
			// cost yet (see pms.SKUCostPriceSourceUnknown), so there's
			// nothing current to check the ledger against. Resolved by the
			// first real receipt, not by this job.
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
			// Nothing currently on the ledger to check the SKU's cost
			// against (fully sold out, or brand new) -- not a drift signal.
			result.Skipped = true
			return nil
		}
		expected := (weightedCost + totalQty/2) / totalQty
		actual := *sku.ActualCostMinorUnits()
		costPrice := *sku.CostPriceMinorUnits

		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		category := resolveCategory(ctx, uc.variantRepo, uc.productRepo, uc.categoryRepo, sku.VariantID)
		markup := EffectiveTargetMarkupBasisPoints(category, settings.DefaultTargetMarkupBasisPoints)
		recommended := RecommendedPriceMinorUnits(costPrice, markup)

		result.Reconciled = actual == expected
		if !result.Reconciled {
			log.Printf("[WARN] inventory cost drift for SKU %s: ledger-derived cost %d does not match current inventory cost %d",
				skuID, expected, actual)
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
		result.InventoryCostMinorUnits = actual
		result.ExpectedCostPriceMinorUnits = expected
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
