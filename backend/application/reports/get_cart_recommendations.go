package reports

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"sort"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var ErrCartEmpty = errs.New("RECOMMENDATIONS_CART_EMPTY", "cart is empty")

const (
	cartRecommendationsMaxItemsetSize = 5
	cartRecommendationsDefaultLimit   = 10
	// defaultBundleConfidenceThreshold, defaultRecommendationsMinSupport,
	// and defaultRecommendationsLookbackDays are used only when the tenant
	// has never saved a TenantSettings row (settingsDomain.TenantSettings'
	// zero-value-means-unset convention) -- every configured tenant uses
	// their own settings value instead. Not a hard limit on what a
	// merchant can configure.
	defaultBundleConfidenceThreshold   = 0.30
	defaultRecommendationsMinSupport   = 2
	defaultRecommendationsLookbackDays = 180
)

// skuLookupByCode resolves the WooCommerce-facing SKU code the storefront
// sends into CommerceOS' internal SKU ID -- the FP-Growth mining below (see
// fpGrowth in get_frequently_bought_together_report.go) operates on the
// same internal IDs every other report in this package already uses.
type skuLookupByCode interface {
	FindByCode(ctx context.Context, code string) (*pmsDomain.SKU, error)
}

// CartLine is one line of the shopper's cart, as the storefront reports it
// (SKU code + quantity) -- see infrastructure/handlers/storefront.
type CartLine struct {
	SKUCode  string
	Quantity int
}

// CartRecommendation is one candidate to cross-sell alongside the cart.
// Confidence is the classic association-rule definition: of the orders
// containing this candidate's best-supported trigger (the cart line it was
// most strongly co-purchased with), what fraction also contained the
// candidate -- support(trigger, candidate) / support(trigger alone), NOT a
// fraction of TotalOrders (that would make confidence collapse toward 0 for
// any store with many distinct SKUs, since it's diluted by every unrelated
// order in the lookback window regardless of whether it could ever have
// contained the trigger). Not inflated by quantity weighting either (that
// only affects ranking, see Execute). IsBundleCandidate flags confidence
// crossing the tenant's own settings.BundleConfidenceThreshold -- CommerceOS
// only signals this; the discount policy itself is the storefront's to own.
type CartRecommendation struct {
	SKUCode           string
	ProductName       string
	Confidence        float64
	IsBundleCandidate bool
}

type CartRecommendationsReport struct {
	Recommendations []CartRecommendation
	TotalOrders     int64
}

type GetCartRecommendationsUseCase struct {
	db            *gorm.DB
	basket        basketLister
	settingsRepo  settingsDomain.Repository
	skuLookup     skuCostPriceLookup
	skuCodeLookup skuLookupByCode
	variantLookup variantLookupForSlowMoving
	productLookup productLookupForSlowMoving
}

func NewGetCartRecommendationsUseCase(
	db *gorm.DB,
	basket basketLister,
	settingsRepo settingsDomain.Repository,
	skuLookup skuCostPriceLookup,
	skuCodeLookup skuLookupByCode,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetCartRecommendationsUseCase {
	return &GetCartRecommendationsUseCase{
		db: db, basket: basket, settingsRepo: settingsRepo,
		skuLookup: skuLookup, skuCodeLookup: skuCodeLookup,
		variantLookup: variantLookup, productLookup: productLookup,
	}
}

type cartCandidateStat struct {
	bestRawSupport int
	// bestTriggerSKUID is which cart line's itemset produced bestRawSupport
	// -- confidence divides by THIS trigger's own singleton support, not
	// some other cart line's, since a multi-line cart can link the same
	// candidate to several lines at different support levels.
	bestTriggerSKUID string
	weightedScore    int
}

// Execute mines cross-sell candidates for the whole cart at once (not one
// call per line): each cart SKU contributes its own best-supported
// co-occurring itemsets, weighted by that line's quantity, so a candidate
// strongly linked to a line the shopper is buying several of outranks one
// only weakly linked to a single-quantity line. A cart line whose SKU code
// isn't found is skipped (soft-fail, same "tolerate a deleted/unknown SKU"
// stance as GetProductRecommendationsUseCase) rather than failing the
// whole request.
func (uc *GetCartRecommendationsUseCase) Execute(ctx context.Context, lines []CartLine) (*CartRecommendationsReport, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("getCartRecommendations: %w", ErrCartEmpty)
	}

	report := &CartRecommendationsReport{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		currency := "USD"
		bundleThreshold := defaultBundleConfidenceThreshold
		minSupport := defaultRecommendationsMinSupport
		lookbackDays := defaultRecommendationsLookbackDays
		if settings, err := uc.settingsRepo.Get(ctx); err == nil {
			if settings.DefaultCurrency != "" {
				currency = settings.DefaultCurrency
			}
			// Zero-value-means-unset (see TenantSettings.BundleConfidenceThreshold's,
			// .RecommendationsMinSupport's, and .RecommendationsLookbackDays'
			// doc comments) -- only trust an explicit 0 from a tenant that
			// has actually saved settings before.
			if !settings.UpdatedAt.IsZero() {
				bundleThreshold = settings.BundleConfidenceThreshold
				minSupport = settings.RecommendationsMinSupport
				lookbackDays = settings.RecommendationsLookbackDays
			}
		}

		to := time.Now()
		from := to.AddDate(0, 0, -lookbackDays)
		orders, err := uc.basket.ListOrderSKUsBetween(ctx, from, to, currency)
		if err != nil {
			return err
		}
		report.TotalOrders = int64(len(orders))

		cartQuantities := make(map[string]int)
		for _, line := range lines {
			sku, err := uc.skuCodeLookup.FindByCode(ctx, line.SKUCode)
			if err != nil || sku == nil {
				continue
			}
			qty := line.Quantity
			if qty < 1 {
				qty = 1
			}
			cartQuantities[sku.ID] += qty
		}
		if len(cartQuantities) == 0 {
			return nil
		}

		transactions := make([][]string, len(orders))
		for i, o := range orders {
			transactions[i] = o.SKUIDs
		}
		itemsets := fpGrowth(transactions, minSupport, cartRecommendationsMaxItemsetSize)

		// Singleton (size-1) itemsets are how many orders contained that
		// SKU alone -- fpGrowth mines every itemset size >= 1, so this is
		// already in the same output rather than needing a separate pass
		// over transactions. Confidence below divides by this: it's always
		// present for any SKU that appears in a size-2+ itemset too, since
		// an item's own support can only be >= any pair support it's part
		// of, and mineFPTree emits every header-table item that clears
		// minSupport on its own.
		singletonSupport := make(map[string]int, len(itemsets))
		for _, is := range itemsets {
			if len(is.Items) == 1 {
				singletonSupport[is.Items[0]] = is.Support
			}
		}

		stats := make(map[string]*cartCandidateStat)
		for cartSKUID, qty := range cartQuantities {
			for _, is := range itemsets {
				if len(is.Items) < 2 {
					continue
				}
				contains := false
				for _, item := range is.Items {
					if item == cartSKUID {
						contains = true
						break
					}
				}
				if !contains {
					continue
				}
				for _, item := range is.Items {
					if item == cartSKUID {
						continue
					}
					if _, inCart := cartQuantities[item]; inCart {
						continue
					}
					st, ok := stats[item]
					if !ok {
						st = &cartCandidateStat{}
						stats[item] = st
					}
					if is.Support > st.bestRawSupport {
						st.bestRawSupport = is.Support
						st.bestTriggerSKUID = cartSKUID
					}
					st.weightedScore += is.Support * qty
				}
			}
		}

		candidates := make([]string, 0, len(stats))
		for id := range stats {
			candidates = append(candidates, id)
		}
		sort.Slice(candidates, func(i, j int) bool {
			if stats[candidates[i]].weightedScore != stats[candidates[j]].weightedScore {
				return stats[candidates[i]].weightedScore > stats[candidates[j]].weightedScore
			}
			return candidates[i] < candidates[j]
		})
		if len(candidates) > cartRecommendationsDefaultLimit {
			candidates = candidates[:cartRecommendationsDefaultLimit]
		}

		for _, candidateID := range candidates {
			// Unlike the admin-facing reports, a candidate that no longer
			// resolves to a live SKU is dropped entirely rather than kept
			// with a blank code -- the storefront can't add an empty SKU
			// to a cart, so an unresolvable candidate has no value here.
			sku, err := uc.skuLookup.FindByID(ctx, candidateID)
			if err != nil {
				continue
			}
			// support(trigger, candidate) / support(trigger alone) -- the
			// classic association-rule confidence, not diluted by every
			// unrelated order in the lookback window (see
			// CartRecommendation's doc comment for why that matters).
			confidence := 0.0
			if triggerSupport := singletonSupport[stats[candidateID].bestTriggerSKUID]; triggerSupport > 0 {
				confidence = float64(stats[candidateID].bestRawSupport) / float64(triggerSupport)
			}
			row := CartRecommendation{
				SKUCode:           sku.SKUCode,
				Confidence:        confidence,
				IsBundleCandidate: confidence >= bundleThreshold,
			}
			if variant, err := uc.variantLookup.FindByID(ctx, sku.VariantID); err == nil {
				if product, err := uc.productLookup.FindByID(ctx, variant.ProductID); err == nil {
					row.ProductName = product.Name
				}
			}
			report.Recommendations = append(report.Recommendations, row)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getCartRecommendations: %w", err)
	}
	return report, nil
}
