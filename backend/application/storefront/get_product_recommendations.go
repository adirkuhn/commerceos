package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const productRecommendationsLimit = 8

// cartRecommendationsGetter narrows *appreports.GetCartRecommendationsUseCase
// to point of use.
type cartRecommendationsGetter interface {
	Execute(ctx context.Context, lines []appreports.CartLine) (*appreports.CartRecommendationsReport, error)
}

// GetProductRecommendationsUseCase powers a PDP's "you might also like"
// section by driving the exact same FP-Growth cross-sell engine cart
// recommendations use (cartRecommendationsGetter), seeded with a
// single-line pseudo-cart -- this product's own representative SKU
// (first variant's first SKU, same "representative SKU" convention
// ProductSummary's own doc comment uses), quantity 1 -- instead of the
// shopper's real cart. The engine already excludes any SKU present in the
// input lines from its own candidate set (see
// GetCartRecommendationsUseCase.Execute's doc comment), so the triggering
// product can never recommend itself. Every candidate is re-checked for
// channel availability (active + enrolled) exactly like GetProductUseCase,
// since a SKU that co-occurred historically may since have been
// deactivated or unenrolled from this channel.
type GetProductRecommendationsUseCase struct {
	db                     *gorm.DB
	channelRepo            channelsDomain.SalesChannelRepository
	overrideRepo           channelsDomain.SalesChannelProductOverrideRepository
	productRepo            pmsDomain.ProductRepository
	variantRepo            pmsDomain.VariantRepository
	skuRepo                pmsDomain.SKURepository
	mediaRepo              pmsDomain.MediaRepository
	getCartRecommendations cartRecommendationsGetter
}

func NewGetProductRecommendationsUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	mediaRepo pmsDomain.MediaRepository,
	getCartRecommendations cartRecommendationsGetter,
) *GetProductRecommendationsUseCase {
	return &GetProductRecommendationsUseCase{
		db: db, channelRepo: channelRepo, overrideRepo: overrideRepo, productRepo: productRepo,
		variantRepo: variantRepo, skuRepo: skuRepo, mediaRepo: mediaRepo, getCartRecommendations: getCartRecommendations,
	}
}

func (uc *GetProductRecommendationsUseCase) Execute(ctx context.Context, salesChannelID, productID string) ([]ProductSummary, error) {
	var channel *channelsDomain.SalesChannel
	var triggerSKUCode string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		channel, err = findInternalChannel(ctx, uc.channelRepo, salesChannelID)
		if err != nil {
			return err
		}
		variants, err := uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			if len(skus) > 0 {
				triggerSKUCode = skus[0].SKUCode
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductRecommendations: %w", err)
	}
	if triggerSKUCode == "" {
		// No SKU to seed the engine with -- nothing to recommend from,
		// not an error (same "empty is a normal outcome" stance
		// getShippingOptions takes for an unresolvable address).
		return nil, nil
	}

	report, err := uc.getCartRecommendations.Execute(ctx, []appreports.CartLine{{SKUCode: triggerSKUCode, Quantity: 1}})
	if err != nil {
		return nil, fmt.Errorf("getProductRecommendations: %w", err)
	}

	var summaries []ProductSummary
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, rec := range report.Recommendations {
			if len(summaries) >= productRecommendationsLimit {
				break
			}
			sku, err := uc.skuRepo.FindByCode(ctx, rec.SKUCode)
			if err != nil || sku == nil {
				continue
			}
			variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
			if err != nil {
				continue
			}
			product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
			if err != nil || product.Status != pmsDomain.ProductStatusActive {
				continue
			}
			override, err := uc.overrideRepo.FindByChannelAndProduct(ctx, salesChannelID, product.ID)
			if err != nil {
				return err
			}
			if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
				continue
			}

			summary := ProductSummary{Product: *product}
			price := channel.EffectivePriceMinorUnits(sku.PriceMinorUnits)
			currency := sku.CurrencyCode
			summary.PriceMinorUnits, summary.CurrencyCode = &price, &currency
			if sku.SalePriceMinorUnits != nil {
				salePrice := channel.EffectivePriceMinorUnits(*sku.SalePriceMinorUnits)
				summary.SalePriceMinorUnits = &salePrice
			}
			if media, err := uc.mediaRepo.ListByProduct(ctx, product.ID); err == nil && len(media) > 0 {
				url := media[0].URL
				summary.ImageURL = &url
			}
			summaries = append(summaries, summary)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductRecommendations: %w", err)
	}
	return summaries, nil
}
