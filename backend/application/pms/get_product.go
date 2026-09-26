package pms

import (
	"context"
	"errors"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type getProductProducts interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

type getProductVariants interface {
	ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error)
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}

type getProductSKUs interface {
	ListByVariant(ctx context.Context, variantID string) ([]pmsDomain.SKU, error)
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

type GetProductUseCase struct {
	db          *gorm.DB
	productRepo getProductProducts
	variantRepo getProductVariants
	skuRepo     getProductSKUs
}

func NewGetProductUseCase(
	db *gorm.DB,
	productRepo getProductProducts,
	variantRepo getProductVariants,
	skuRepo getProductSKUs,
) *GetProductUseCase {
	return &GetProductUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo}
}

func (uc *GetProductUseCase) Execute(ctx context.Context, id string) (*ProductDetail, error) {
	var detail ProductDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		product, err := uc.productRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		detail.Product = *product

		variants, err := uc.variantRepo.ListByProduct(ctx, id)
		if err != nil {
			return err
		}

		detail.Variants = make([]VariantDetail, len(variants))
		for i, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			detail.Variants[i] = VariantDetail{Variant: v, SKUs: skus}
		}
		detail.LinkedSKUs, err = uc.linkedSKUs(ctx, detail.Variants)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getProduct: %w", err)
	}
	return &detail, nil
}

// linkedSKUs summarizes every SKU another SKU of this product is a case or
// a portion of, so the editor can name it instead of showing its id. A
// link whose SKU (or its product) no longer exists is left out rather than
// failing the whole product.
func (uc *GetProductUseCase) linkedSKUs(ctx context.Context, variants []VariantDetail) (map[string]LinkedSKU, error) {
	linked := make(map[string]LinkedSKU)
	for _, v := range variants {
		for _, s := range v.SKUs {
			for _, id := range []*string{s.CaseOfSKUID, s.PortionedFromSKUID} {
				if id == nil {
					continue
				}
				if _, done := linked[*id]; done {
					continue
				}
				summary, err := uc.summarize(ctx, *id)
				if errors.Is(err, pmsDomain.ErrSKUNotFound) || errors.Is(err, pmsDomain.ErrVariantNotFound) || errors.Is(err, pmsDomain.ErrProductNotFound) {
					continue
				}
				if err != nil {
					return nil, err
				}
				linked[*id] = *summary
			}
		}
	}
	return linked, nil
}

func (uc *GetProductUseCase) summarize(ctx context.Context, skuID string) (*LinkedSKU, error) {
	sku, err := uc.skuRepo.FindByID(ctx, skuID)
	if err != nil {
		return nil, err
	}
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return nil, err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return nil, err
	}
	return &LinkedSKU{ID: sku.ID, ProductName: product.Name, SKUCode: sku.SKUCode, Barcode: sku.Barcode}, nil
}
