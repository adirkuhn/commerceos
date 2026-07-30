package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateCategoryInput struct {
	ID       string
	Name     string
	ParentID *string
}

type UpdateCategoryUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
}

func NewUpdateCategoryUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{db: db, categoryRepo: categoryRepo}
}

func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, in UpdateCategoryInput) (*pmsDomain.Category, error) {
	var category pmsDomain.Category
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.categoryRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.ParentID = in.ParentID
		existing.UpdatedAt = time.Now()
		if err := uc.categoryRepo.Update(ctx, existing); err != nil {
			return err
		}
		category = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateCategory: %w", err)
	}
	return &category, nil
}
