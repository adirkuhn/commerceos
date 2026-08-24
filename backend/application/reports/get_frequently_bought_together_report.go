package reports

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	frequentlyBoughtTogetherDefaultMinSupport = 2
	frequentlyBoughtTogetherMaxItemsetSize    = 5
	frequentlyBoughtTogetherMaxRows           = 50
)

// basketLister narrows domain/oms.BasketAnalysisRepository to the one
// method this report needs (ISP).
type basketLister interface {
	ListOrderSKUsBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.OrderSKUs, error)
}

// FrequentlyBoughtTogetherItem is one SKU within a mined itemset. SKUCode
// and ProductName are left empty (soft-fail) if the SKU has since been
// deleted -- the itemset itself is still valid and reported.
type FrequentlyBoughtTogetherItem struct {
	SKUID       string
	SKUCode     string
	ProductName string
}

// FrequentlyBoughtTogetherRow is one FP-Growth-mined itemset of size >= 2
// with its support (number of qualifying orders containing every item in
// Items) and that support as a percentage of TotalOrders.
type FrequentlyBoughtTogetherRow struct {
	Items          []FrequentlyBoughtTogetherItem
	Support        int64
	SupportPercent float64
}

// FrequentlyBoughtTogetherReport is the Phase 1 "Frequently Bought
// Together" report: every SKU itemset of size 2+ that co-occurred in at
// least minSupport orders in [From, To), most-supported first.
type FrequentlyBoughtTogetherReport struct {
	Rows        []FrequentlyBoughtTogetherRow
	TotalOrders int64
}

type GetFrequentlyBoughtTogetherReportUseCase struct {
	db            *gorm.DB
	basket        basketLister
	settingsRepo  settingsDomain.Repository
	skuLookup     skuCostPriceLookup
	variantLookup variantLookupForSlowMoving
	productLookup productLookupForSlowMoving
}

func NewGetFrequentlyBoughtTogetherReportUseCase(
	db *gorm.DB,
	basket basketLister,
	settingsRepo settingsDomain.Repository,
	skuLookup skuCostPriceLookup,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetFrequentlyBoughtTogetherReportUseCase {
	return &GetFrequentlyBoughtTogetherReportUseCase{
		db: db, basket: basket, settingsRepo: settingsRepo,
		skuLookup: skuLookup, variantLookup: variantLookup, productLookup: productLookup,
	}
}

// Execute mines frequent SKU itemsets from orders placed in [from, to).
// minSupport <= 0 means "use the default" (frequentlyBoughtTogetherDefaultMinSupport),
// not "no threshold".
func (uc *GetFrequentlyBoughtTogetherReportUseCase) Execute(ctx context.Context, from, to time.Time, minSupport int) (*FrequentlyBoughtTogetherReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getFrequentlyBoughtTogetherReport: %w", ErrToBeforeFrom)
	}
	if minSupport <= 0 {
		minSupport = frequentlyBoughtTogetherDefaultMinSupport
	}

	var report FrequentlyBoughtTogetherReport
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
		itemsets := fpGrowth(transactions, minSupport, frequentlyBoughtTogetherMaxItemsetSize)

		var multiItemsets []frequentItemset
		for _, is := range itemsets {
			if len(is.Items) >= 2 {
				multiItemsets = append(multiItemsets, is)
			}
		}
		sort.Slice(multiItemsets, func(i, j int) bool {
			if multiItemsets[i].Support != multiItemsets[j].Support {
				return multiItemsets[i].Support > multiItemsets[j].Support
			}
			if len(multiItemsets[i].Items) != len(multiItemsets[j].Items) {
				return len(multiItemsets[i].Items) > len(multiItemsets[j].Items)
			}
			return strings.Join(multiItemsets[i].Items, ",") < strings.Join(multiItemsets[j].Items, ",")
		})
		if len(multiItemsets) > frequentlyBoughtTogetherMaxRows {
			multiItemsets = multiItemsets[:frequentlyBoughtTogetherMaxRows]
		}

		for _, is := range multiItemsets {
			row := FrequentlyBoughtTogetherRow{Support: int64(is.Support)}
			if report.TotalOrders > 0 {
				row.SupportPercent = float64(is.Support) / float64(report.TotalOrders) * 100
			}
			for _, skuID := range is.Items {
				item := FrequentlyBoughtTogetherItem{SKUID: skuID}
				if sku, err := uc.skuLookup.FindByID(ctx, skuID); err == nil {
					item.SKUCode = sku.SKUCode
					if variant, err := uc.variantLookup.FindByID(ctx, sku.VariantID); err == nil {
						if product, err := uc.productLookup.FindByID(ctx, variant.ProductID); err == nil {
							item.ProductName = product.Name
						}
					}
				}
				row.Items = append(row.Items, item)
			}
			report.Rows = append(report.Rows, row)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getFrequentlyBoughtTogetherReport: %w", err)
	}

	return &report, nil
}
