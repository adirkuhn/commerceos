package pms

import (
	"context"
	"fmt"
	"sort"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultMarginReportPerPage = 50
	maxMarginReportPerPage     = 200
)

// MarginReportRow is one line of the tenant-wide margin report -- every
// active SKU the cron has computed a cost price for, current price/margin,
// and how that compares to its recommended price/target margin.
type MarginReportRow struct {
	SKUID                      string
	SKUCode                    string
	ProductName                string
	CostPriceMinorUnits        int64
	PriceMinorUnits            int64
	ProfitMinorUnits           int64
	MarginBasisPoints          int
	TargetMarginBasisPoints    int
	RecommendedPriceMinorUnits int64
	// BelowTarget is true when MarginBasisPoints is under
	// TargetMarginBasisPoints -- includes negative margin (selling below
	// cost), the report's flagging signal.
	BelowTarget bool
}

type ListMarginReportInput struct {
	Page    int
	PerPage int
	// BelowTargetOnly, when true, returns only flagged (BelowTarget) rows.
	BelowTargetOnly bool
}

type ListMarginReportResult struct {
	Rows  []MarginReportRow
	Total int64
}

// ListMarginReportUseCase computes the report by loading every active SKU
// with a cost price and joining in display/category data per SKU -- a
// report-page query, not a hot path, so the per-SKU Variant/Product/Category
// lookups (see resolveCategory) are an acceptable simplification over a
// bulk join for v1.
type ListMarginReportUseCase struct {
	db           *gorm.DB
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
	categoryRepo pmsDomain.CategoryRepository
	settingsRepo settingsDomain.Repository
}

func NewListMarginReportUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
	settingsRepo settingsDomain.Repository,
) *ListMarginReportUseCase {
	return &ListMarginReportUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
		categoryRepo: categoryRepo, settingsRepo: settingsRepo,
	}
}

func (uc *ListMarginReportUseCase) Execute(ctx context.Context, in ListMarginReportInput) (*ListMarginReportResult, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultMarginReportPerPage
	}
	if perPage > maxMarginReportPerPage {
		perPage = maxMarginReportPerPage
	}

	var result ListMarginReportResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		skus, err := uc.skuRepo.ListActive(ctx)
		if err != nil {
			return err
		}
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}

		rows := make([]MarginReportRow, 0, len(skus))
		for _, sku := range skus {
			if sku.CostPriceMinorUnits == nil {
				continue
			}
			cost := *sku.CostPriceMinorUnits
			category := resolveCategory(ctx, uc.variantRepo, uc.productRepo, uc.categoryRepo, sku.VariantID)
			targetMargin := EffectiveTargetMarginBasisPoints(category, settings.DefaultTargetMarginBasisPoints)

			row := MarginReportRow{
				SKUID: sku.ID, SKUCode: sku.SKUCode,
				CostPriceMinorUnits: cost, PriceMinorUnits: sku.PriceMinorUnits,
				ProfitMinorUnits:           sku.PriceMinorUnits - cost,
				TargetMarginBasisPoints:    targetMargin,
				RecommendedPriceMinorUnits: RecommendedPriceMinorUnits(cost, targetMargin),
			}
			if sku.PriceMinorUnits > 0 {
				row.MarginBasisPoints = int(row.ProfitMinorUnits * 10000 / sku.PriceMinorUnits)
			}
			row.BelowTarget = row.MarginBasisPoints < targetMargin
			if variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID); err == nil && variant != nil {
				if product, err := uc.productRepo.FindByID(ctx, variant.ProductID); err == nil && product != nil {
					row.ProductName = product.Name
				}
			}
			if in.BelowTargetOnly && !row.BelowTarget {
				continue
			}
			rows = append(rows, row)
		}

		sort.Slice(rows, func(i, j int) bool { return rows[i].MarginBasisPoints < rows[j].MarginBasisPoints })
		result.Total = int64(len(rows))

		start := (page - 1) * perPage
		if start > len(rows) {
			start = len(rows)
		}
		end := start + perPage
		if end > len(rows) {
			end = len(rows)
		}
		result.Rows = rows[start:end]
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listMarginReport: %w", err)
	}
	return &result, nil
}
