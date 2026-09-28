package storefront

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type productBySlugFinder interface {
	FindBySlug(ctx context.Context, slug string) (*pmsDomain.Product, error)
}

type productDetailGetter interface {
	Execute(ctx context.Context, salesChannelID, productID string) (*ProductDetail, error)
}

// GetProductBySlugUseCase serves /products/{slug} storefront URLs: it finds
// the product id and delegates to GetProductUseCase, which owns the
// channel/enrollment/status rules.
type GetProductBySlugUseCase struct {
	db    *gorm.DB
	repo  productBySlugFinder
	getBy productDetailGetter
}

func NewGetProductBySlugUseCase(db *gorm.DB, repo productBySlugFinder, get productDetailGetter) *GetProductBySlugUseCase {
	return &GetProductBySlugUseCase{db: db, repo: repo, getBy: get}
}

func (uc *GetProductBySlugUseCase) Execute(ctx context.Context, salesChannelID, slug string) (*ProductDetail, error) {
	var id string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.repo.FindBySlug(ctx, slug)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrProductNotFound) {
				return storefrontDomain.ErrProductNotAvailable
			}
			return err
		}
		if p == nil {
			return storefrontDomain.ErrProductNotAvailable
		}
		id = p.ID
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductBySlug: %w", err)
	}
	return uc.getBy.Execute(ctx, salesChannelID, id)
}
