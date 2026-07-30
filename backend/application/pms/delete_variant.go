package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteVariantUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
}

func NewDeleteVariantUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository) *DeleteVariantUseCase {
	return &DeleteVariantUseCase{db: db, variantRepo: variantRepo}
}

// Execute deletes the variant; its skus cascade via the DB's ON DELETE CASCADE.
func (uc *DeleteVariantUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.variantRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.variantRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteVariant: %w", err)
	}
	return nil
}
