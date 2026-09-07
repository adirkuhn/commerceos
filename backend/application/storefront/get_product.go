package storefront

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SKUDetail is one SKU plus its current stock -- StockQuantity has to be
// filled in as a separate step (see GetProductUseCase.Execute), since
// StockLevelGetter opens its own transaction and this codebase never nests
// tenant.WithTenant calls.
type SKUDetail struct {
	pmsDomain.SKU
	StockQuantity int64
}

// VariantDetail is one variant plus every SKU sold under it -- a product
// detail page's "pick a size/color, see its price and stock" unit.
type VariantDetail struct {
	pmsDomain.Variant
	SKUs []SKUDetail
}

// ProductDetail is the full storefront product-page shape -- unlike
// ProductSummary (one representative price/image for a listing row), this
// carries every variant/SKU and every image, same "detail response has no
// data the caller has to guess at" convention the admin PMS API's own
// Product/Variant/SKU detail shapes already use.
type ProductDetail struct {
	pmsDomain.Product
	Images   []string
	Variants []VariantDetail
}

type GetProductUseCase struct {
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	productRepo  pmsDomain.ProductRepository
	variantRepo  pmsDomain.VariantRepository
	skuRepo      pmsDomain.SKURepository
	mediaRepo    pmsDomain.MediaRepository
	stockLevel   StockLevelGetter
}

func NewGetProductUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	mediaRepo pmsDomain.MediaRepository,
	stockLevel StockLevelGetter,
) *GetProductUseCase {
	return &GetProductUseCase{
		db: db, channelRepo: channelRepo, overrideRepo: overrideRepo,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo, mediaRepo: mediaRepo, stockLevel: stockLevel,
	}
}

// Execute returns storefrontDomain.ErrProductNotAvailable for a product
// that doesn't exist, isn't Active, or isn't enrolled in salesChannelID --
// deliberately the same error/status for all three, same "never let a
// caller distinguish which reason" rule application/storefront's cart
// lookups already follow.
func (uc *GetProductUseCase) Execute(ctx context.Context, salesChannelID, productID string) (*ProductDetail, error) {
	var detail ProductDetail
	var skuIDs []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := findInternalChannel(ctx, uc.channelRepo, salesChannelID)
		if err != nil {
			return err
		}

		product, err := uc.productRepo.FindByID(ctx, productID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrProductNotFound) {
				return storefrontDomain.ErrProductNotAvailable
			}
			return err
		}
		if product.Status != pmsDomain.ProductStatusActive {
			return storefrontDomain.ErrProductNotAvailable
		}
		override, err := uc.overrideRepo.FindByChannelAndProduct(ctx, salesChannelID, productID)
		if err != nil {
			return err
		}
		if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
			return storefrontDomain.ErrProductNotAvailable
		}

		variants, err := uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		variantDetails := make([]VariantDetail, len(variants))
		for i, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			skuDetails := make([]SKUDetail, len(skus))
			for j, s := range skus {
				skuDetails[j] = SKUDetail{SKU: s}
				skuIDs = append(skuIDs, s.ID)
			}
			variantDetails[i] = VariantDetail{Variant: v, SKUs: skuDetails}
		}

		media, err := uc.mediaRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		images := make([]string, len(media))
		for i, m := range media {
			images[i] = m.URL
		}

		detail = ProductDetail{Product: *product, Images: images, Variants: variantDetails}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProduct: %w", err)
	}

	stockBySKU := make(map[string]int64, len(skuIDs))
	for _, id := range skuIDs {
		level, err := uc.stockLevel.Execute(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("getProduct: %w", err)
		}
		stockBySKU[id] = level.Syncable
	}
	for i := range detail.Variants {
		for j := range detail.Variants[i].SKUs {
			detail.Variants[i].SKUs[j].StockQuantity = stockBySKU[detail.Variants[i].SKUs[j].ID]
		}
	}
	return &detail, nil
}
