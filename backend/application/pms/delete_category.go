package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteCategoryUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
}

func NewDeleteCategoryUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository) *DeleteCategoryUseCase {
	return &DeleteCategoryUseCase{db: db, categoryRepo: categoryRepo}
}

func (uc *DeleteCategoryUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.categoryRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.categoryRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteCategory: %w", err)
	}
	return nil
}
