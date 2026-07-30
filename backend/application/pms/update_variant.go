package pms

import (
	"context"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateVariantInput struct {
	ID       string
	Name     string
	Position int
}

type UpdateVariantUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
}

func NewUpdateVariantUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository) *UpdateVariantUseCase {
	return &UpdateVariantUseCase{db: db, variantRepo: variantRepo}
}

func (uc *UpdateVariantUseCase) Execute(ctx context.Context, in UpdateVariantInput) (*pmsDomain.Variant, error) {
	var variant pmsDomain.Variant
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.variantRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.Position = in.Position
		existing.UpdatedAt = time.Now()
		if err := uc.variantRepo.Update(ctx, existing); err != nil {
			return err
		}
		variant = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateVariant: %w", err)
	}
	return &variant, nil
}
