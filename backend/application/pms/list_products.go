package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// listProductsLimit is a fixed, generous page size — real server-side
// pagination is a later refinement, not needed for the current UI.
const listProductsLimit = 200

type ListProductsUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
}

func NewListProductsUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{db: db, productRepo: productRepo}
}

func (uc *ListProductsUseCase) Execute(ctx context.Context) ([]pmsDomain.Product, error) {
	var products []pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, _, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{Limit: listProductsLimit})
		if err != nil {
			return err
		}
		products = p
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listProducts: %w", err)
	}
	return products, nil
}
