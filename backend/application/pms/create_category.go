package pms

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateCategoryInput struct {
	Name     string
	ParentID *string
}

type CreateCategoryUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
}

func NewCreateCategoryUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository) *CreateCategoryUseCase {
	return &CreateCategoryUseCase{db: db, categoryRepo: categoryRepo}
}

func (uc *CreateCategoryUseCase) Execute(ctx context.Context, in CreateCategoryInput) (*pmsDomain.Category, error) {
	var category pmsDomain.Category
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		slug := Slugify(in.Name)
		_, err := uc.categoryRepo.FindBySlug(ctx, slug)
		if err == nil {
			return pmsDomain.ErrDuplicateSlug
		}
		if !errors.Is(err, pmsDomain.ErrCategoryNotFound) {
			return err
		}

		now := time.Now()
		category = pmsDomain.Category{
			TenantID:  tenant.IDFromContext(ctx),
			ParentID:  in.ParentID,
			Name:      in.Name,
			Slug:      slug,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return uc.categoryRepo.Create(ctx, &category)
	})
	if err != nil {
		return nil, fmt.Errorf("createCategory: %w", err)
	}
	return &category, nil
}
