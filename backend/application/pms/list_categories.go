package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

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
		c, err := uc.categoryRepo.List(ctx)
		if err != nil {
			return err
		}
		categories = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listCategories: %w", err)
	}
	return categories, nil
}
