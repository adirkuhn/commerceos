package pms

import (
	"context"
	"fmt"
	"math"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const shrinkageTrailingDays = 30

// discardListerForMarginInsight is the narrow slice of
// *inventory.ListDiscardsUseCase this use case needs (ISP) -- see
// batchListerForCostPrice's doc comment for why a use case, not a repo.
type discardListerForMarginInsight interface {
	Execute(ctx context.Context, in invapp.ListDiscardsInput) (*invapp.ListDiscardsResult, error)
}

// GetSKUMarginInsightResult is the product page's "insight" panel: what a
// unit costs, what it's recommended to sell for, and what's actually being
// made at the price(s) currently set. All derived/read-only -- nothing here
// is persisted beyond SKU.CostPriceMinorUnits itself (written by the cron).
type GetSKUMarginInsightResult struct {
	// CostPriceAvailable is false until the daily cron has computed at
	// least one value for this SKU (see SKU.CostPriceMinorUnits's doc
	// comment) -- every other field is zero when false, there's no cost
	// basis to derive them from yet.
	CostPriceAvailable  bool
	CostPriceMinorUnits int64
	// CostPriceSource says whether CostPriceMinorUnits came from the daily
	// cron or a human's manual override (see pmsDomain.SKUCostPriceSource) --
	// the product edit page uses this to decide whether to show "override"
	// or "clear override".
	CostPriceSource                  pmsDomain.SKUCostPriceSource
	EffectiveTargetMarkupBasisPoints int
	RecommendedPriceMinorUnits       int64
	// ProfitAtPriceMinorUnits/MarginAtPriceBasisPoints/MarkupAtPriceBasisPoints
	// are computed against SKU.PriceMinorUnits -- margin is profit as a % of
	// price, markup is profit as a % of cost, same number expressed two
	// ways. The *AtSalePrice* fields mirror them against SalePriceMinorUnits,
	// only populated (non-nil) when a sale is active.
	ProfitAtPriceMinorUnits      int64
	MarginAtPriceBasisPoints     int
	MarkupAtPriceBasisPoints     int
	ProfitAtSalePriceMinorUnits  *int64
	MarginAtSalePriceBasisPoints *int
	MarkupAtSalePriceBasisPoints *int
	// TotalOnHandQuantity is summed across every batch and every
	// warehouse's general ledger (ListBatchesUseCase's full result, not
	// just cost-bearing rows -- this is physical count, unlike the cron's
	// cost-weighted average which only counts cost-bearing batches).
	TotalOnHandQuantity int64
	// TotalProfitPotentialMinorUnits is profit-per-unit at the currently
	// effective selling price (SalePriceMinorUnits when a sale is active,
	// else PriceMinorUnits) times TotalOnHandQuantity -- "what selling
	// everything in stock right now would net".
	TotalProfitPotentialMinorUnits int64
	// ShrinkageMinorUnits is the cost value of everything discarded
	// (damaged/lost) for this SKU in the trailing shrinkageTrailingDays --
	// a separate figure, deliberately not blended into CostPriceMinorUnits
	// (see RecalculateCostPricesUseCase's doc comment). A discard from a
	// batch with a known landed cost uses that batch's cost; a
	// general-ledger (un-lotted) discard falls back to the SKU's current
	// CostPriceMinorUnits as the best available estimate.
	ShrinkageMinorUnits int64
}

type GetSKUMarginInsightUseCase struct {
	db            *gorm.DB
	skuRepo       pmsDomain.SKURepository
	variantRepo   pmsDomain.VariantRepository
	productRepo   pmsDomain.ProductRepository
	categoryRepo  pmsDomain.CategoryRepository
	settingsRepo  settingsDomain.Repository
	batchLister   batchListerForCostPrice
	discardLister discardListerForMarginInsight
}

func NewGetSKUMarginInsightUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
	settingsRepo settingsDomain.Repository,
	batchLister batchListerForCostPrice,
	discardLister discardListerForMarginInsight,
) *GetSKUMarginInsightUseCase {
	return &GetSKUMarginInsightUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
		categoryRepo: categoryRepo, settingsRepo: settingsRepo, batchLister: batchLister, discardLister: discardLister,
	}
}

func (uc *GetSKUMarginInsightUseCase) Execute(ctx context.Context, skuID string) (*GetSKUMarginInsightResult, error) {
	var result GetSKUMarginInsightResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		result.CostPriceSource = sku.CostPriceSource
		if result.CostPriceSource == "" {
			result.CostPriceSource = pmsDomain.SKUCostPriceSourceComputed
		}

		batches, err := uc.batchLister.Execute(ctx, skuID)
		if err != nil {
			return err
		}
		costBySKUBatch := make(map[string]int64, len(batches))
		for _, b := range batches {
			result.TotalOnHandQuantity += b.Quantity
			if b.LandedCostMinorUnits != nil {
				costBySKUBatch[b.ID] = *b.LandedCostMinorUnits
			}
		}

		if sku.CostPriceMinorUnits == nil {
			return uc.computeShrinkage(ctx, skuID, nil, costBySKUBatch, &result)
		}
		result.CostPriceAvailable = true
		result.CostPriceMinorUnits = *sku.CostPriceMinorUnits

		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		category := resolveCategory(ctx, uc.variantRepo, uc.productRepo, uc.categoryRepo, sku.VariantID)
		result.EffectiveTargetMarkupBasisPoints = EffectiveTargetMarkupBasisPoints(category, settings.DefaultTargetMarkupBasisPoints)
		result.RecommendedPriceMinorUnits = RecommendedPriceMinorUnits(result.CostPriceMinorUnits, result.EffectiveTargetMarkupBasisPoints)

		result.ProfitAtPriceMinorUnits = sku.PriceMinorUnits - result.CostPriceMinorUnits
		if sku.PriceMinorUnits > 0 {
			result.MarginAtPriceBasisPoints = basisPoints(result.ProfitAtPriceMinorUnits, sku.PriceMinorUnits)
		}
		if result.CostPriceMinorUnits > 0 {
			result.MarkupAtPriceBasisPoints = basisPoints(result.ProfitAtPriceMinorUnits, result.CostPriceMinorUnits)
		}

		effectivePrice := sku.PriceMinorUnits
		if sku.SalePriceMinorUnits != nil {
			effectivePrice = *sku.SalePriceMinorUnits
			profit := *sku.SalePriceMinorUnits - result.CostPriceMinorUnits
			result.ProfitAtSalePriceMinorUnits = &profit
			if *sku.SalePriceMinorUnits > 0 {
				margin := basisPoints(profit, *sku.SalePriceMinorUnits)
				result.MarginAtSalePriceBasisPoints = &margin
			}
			if result.CostPriceMinorUnits > 0 {
				markup := basisPoints(profit, result.CostPriceMinorUnits)
				result.MarkupAtSalePriceBasisPoints = &markup
			}
		}
		result.TotalProfitPotentialMinorUnits = (effectivePrice - result.CostPriceMinorUnits) * result.TotalOnHandQuantity

		return uc.computeShrinkage(ctx, skuID, sku.CostPriceMinorUnits, costBySKUBatch, &result)
	})
	if err != nil {
		return nil, fmt.Errorf("getSKUMarginInsight: %w", err)
	}
	return &result, nil
}

func (uc *GetSKUMarginInsightUseCase) computeShrinkage(ctx context.Context, skuID string, fallbackCost *int64, costBySKUBatch map[string]int64, result *GetSKUMarginInsightResult) error {
	from := time.Now().AddDate(0, 0, -shrinkageTrailingDays)
	discards, err := uc.discardLister.Execute(ctx, invapp.ListDiscardsInput{SKUID: &skuID, From: &from, Limit: 200})
	if err != nil {
		return err
	}
	for _, d := range discards.Discards {
		var unitCost int64
		if d.BatchID != nil {
			if c, ok := costBySKUBatch[*d.BatchID]; ok {
				unitCost = c
			} else if fallbackCost != nil {
				unitCost = *fallbackCost
			}
		} else if fallbackCost != nil {
			unitCost = *fallbackCost
		}
		result.ShrinkageMinorUnits += unitCost * d.Quantity
	}
	return nil
}

// basisPoints is part/whole in basis points, rounded half up -- the same as
// the product page's profitAt (Math.round), so the figure it shows while a
// price is typed matches the one saved.
func basisPoints(part, whole int64) int {
	return int(math.Floor(float64(part)*10000/float64(whole) + 0.5))
}
