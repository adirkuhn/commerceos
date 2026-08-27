package pms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateCategoryInput struct {
	ID       string
	Name     string
	ParentID *string
	// TargetMarkupBasisPoints is nil for "use the tenant default", or a
	// non-negative value to override it -- see
	// pms.Category.TargetMarkupBasisPoints. Full-replace semantics, same
	// as UpdateSKUInput.SalePriceMinorUnits: the caller always resends the
	// whole current value (nil included), there's no separate "leave
	// untouched" state.
	TargetMarkupBasisPoints *int
}

type UpdateCategoryUseCase struct {
	db           *gorm.DB
	categoryRepo pmsDomain.CategoryRepository
	outboxRepo   outbox.Repository
}

func NewUpdateCategoryUseCase(db *gorm.DB, categoryRepo pmsDomain.CategoryRepository, outboxRepo outbox.Repository) *UpdateCategoryUseCase {
	return &UpdateCategoryUseCase{db: db, categoryRepo: categoryRepo, outboxRepo: outboxRepo}
}

func (uc *UpdateCategoryUseCase) Execute(ctx context.Context, in UpdateCategoryInput) (*pmsDomain.Category, error) {
	if in.TargetMarkupBasisPoints != nil && *in.TargetMarkupBasisPoints < 0 {
		return nil, fmt.Errorf("updateCategory: %w", pmsDomain.ErrInvalidTargetMarkupBasisPoints)
	}

	var category pmsDomain.Category
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.categoryRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.ParentID = in.ParentID
		existing.TargetMarkupBasisPoints = in.TargetMarkupBasisPoints
		existing.UpdatedAt = time.Now()
		if err := uc.categoryRepo.Update(ctx, existing); err != nil {
			return err
		}
		category = *existing
		return enqueueCategoryChanged(ctx, uc.outboxRepo, category.ID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCategory: %w", err)
	}
	return &category, nil
}
