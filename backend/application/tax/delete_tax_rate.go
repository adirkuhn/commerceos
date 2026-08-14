package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteTaxRateUseCase struct {
	db   *gorm.DB
	repo taxDomain.TaxRateRepository
}

func NewDeleteTaxRateUseCase(db *gorm.DB, repo taxDomain.TaxRateRepository) *DeleteTaxRateUseCase {
	return &DeleteTaxRateUseCase{db: db, repo: repo}
}

func (uc *DeleteTaxRateUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteTaxRate: %w", err)
	}
	return nil
}
