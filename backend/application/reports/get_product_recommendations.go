package reports

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var ErrSKUIDRequired = errors.New("sku id is required")

const (
	productRecommendationsDefaultMinSupport = 2
	productRecommendationsMaxItemsetSize    = 5
	productRecommendationsDefaultLimit      = 10
)

// ProductRecommendationRow is one candidate SKU frequently bought together
// with the queried SKU. Support is the strongest support among every
// FP-Growth-mined itemset (any size) containing both the queried SKU and
// this candidate -- not a sum, so a candidate that co-occurs via several
// overlapping itemsets isn't inflated above its best single grouping.
type ProductRecommendationRow struct {
	SKUID       string
	SKUCode     string
	ProductName string
	Support     int64
}

// ProductRecommendationsReport is "given this product, what else is
// frequently bought with it" -- the per-SKU counterpart to the tenant-wide
// Frequently Bought Together report, reusing the same FP-Growth mining
// over the same [from, to) order data.
type ProductRecommendationsReport struct {
	SKUID       string
	Rows        []ProductRecommendationRow
	TotalOrders int64
}

type GetProductRecommendationsUseCase struct {
	db            *gorm.DB
	basket        basketLister
	settingsRepo  settingsDomain.Repository
	skuLookup     skuCostPriceLookup
	variantLookup variantLookupForSlowMoving
	productLookup productLookupForSlowMoving
}

func NewGetProductRecommendationsUseCase(
	db *gorm.DB,
	basket basketLister,
	settingsRepo settingsDomain.Repository,
	skuLookup skuCostPriceLookup,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetProductRecommendationsUseCase {
	return &GetProductRecommendationsUseCase{
		db: db, basket: basket, settingsRepo: settingsRepo,
		skuLookup: skuLookup, variantLookup: variantLookup, productLookup: productLookup,
	}
}

// Execute mines recommendations for skuID from orders placed in [from, to).
// minSupport/limit <= 0 mean "use the default", not "no threshold"/"no cap".
func (uc *GetProductRecommendationsUseCase) Execute(ctx context.Context, skuID string, from, to time.Time, minSupport, limit int) (*ProductRecommendationsReport, error) {
	if skuID == "" {
		return nil, fmt.Errorf("getProductRecommendations: %w", ErrSKUIDRequired)
	}
	if !to.After(from) {
		return nil, fmt.Errorf("getProductRecommendations: %w", ErrToBeforeFrom)
	}
	if minSupport <= 0 {
		minSupport = productRecommendationsDefaultMinSupport
	}
	if limit <= 0 {
		limit = productRecommendationsDefaultLimit
	}

	report := ProductRecommendationsReport{SKUID: skuID}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		currency := "USD"
		if settings, err := uc.settingsRepo.Get(ctx); err == nil && settings.DefaultCurrency != "" {
			currency = settings.DefaultCurrency
		}
		orders, err := uc.basket.ListOrderSKUsBetween(ctx, from, to, currency)
		if err != nil {
			return err
		}
		report.TotalOrders = int64(len(orders))

		transactions := make([][]string, len(orders))
		for i, o := range orders {
			transactions[i] = o.SKUIDs
		}
		itemsets := fpGrowth(transactions, minSupport, productRecommendationsMaxItemsetSize)

		bestSupport := make(map[string]int)
		for _, is := range itemsets {
			if len(is.Items) < 2 {
				continue
			}
			contains := false
			for _, item := range is.Items {
				if item == skuID {
					contains = true
					break
				}
			}
			if !contains {
				continue
			}
			for _, item := range is.Items {
				if item == skuID {
					continue
				}
				if is.Support > bestSupport[item] {
					bestSupport[item] = is.Support
				}
			}
		}

		candidates := make([]string, 0, len(bestSupport))
		for item := range bestSupport {
			candidates = append(candidates, item)
		}
		sort.Slice(candidates, func(i, j int) bool {
			if bestSupport[candidates[i]] != bestSupport[candidates[j]] {
				return bestSupport[candidates[i]] > bestSupport[candidates[j]]
			}
			return candidates[i] < candidates[j]
		})
		if len(candidates) > limit {
			candidates = candidates[:limit]
		}

		for _, candidateID := range candidates {
			row := ProductRecommendationRow{SKUID: candidateID, Support: int64(bestSupport[candidateID])}
			if sku, err := uc.skuLookup.FindByID(ctx, candidateID); err == nil {
				row.SKUCode = sku.SKUCode
				if variant, err := uc.variantLookup.FindByID(ctx, sku.VariantID); err == nil {
					if product, err := uc.productLookup.FindByID(ctx, variant.ProductID); err == nil {
						row.ProductName = product.Name
					}
				}
			}
			report.Rows = append(report.Rows, row)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductRecommendations: %w", err)
	}
	return &report, nil
}
