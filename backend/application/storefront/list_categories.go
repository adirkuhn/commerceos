package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListCategoriesUseCase returns the tenant's whole category tree, flat
// with ParentID -- same convention the admin PMS API already uses (client
// builds the nested tree). Categories aren't channel-scoped: a channel's
// enrollment rule decides which PRODUCTS are visible, never which
// categories exist to browse -- a category with zero enrolled products
// for this channel is still a legitimate (if empty) branch to show.
type ListCategoriesUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
}

func NewListCategoriesUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{db: db, categoryRepo: categoryRepo}
}

func (uc *ListCategoriesUseCase) Execute(ctx context.Context) ([]pmsDomain.Category, error) {
	var categories []pmsDomain.Category
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		categories, err = uc.categoryRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCategories: %w", err)
	}
	return categories, nil
}
