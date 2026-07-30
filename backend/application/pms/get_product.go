package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
}

func NewGetProductUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProduct: %w", err)
	}
	return &detail, nil
}
