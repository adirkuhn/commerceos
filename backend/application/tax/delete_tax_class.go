package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuTaxClassChecker narrows pms.SKURepository to the one method this use
// case needs (ISP) -- defined here at point of use rather than importing
// the whole domain/pms.SKURepository interface.
type skuTaxClassChecker interface {
	ExistsWithTaxClass(ctx context.Context, taxClassID string) (bool, error)
}

type DeleteTaxClassUseCase struct {
	db         *gorm.DB
	repo       taxDomain.TaxClassRepository
	skuChecker skuTaxClassChecker
}

func NewDeleteTaxClassUseCase(db *gorm.DB, repo taxDomain.TaxClassRepository, skuChecker skuTaxClassChecker) *DeleteTaxClassUseCase {
	return &DeleteTaxClassUseCase{db: db, repo: repo, skuChecker: skuChecker}
}

func (uc *DeleteTaxClassUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if existing.IsDefault {
			return taxDomain.ErrCannotDeleteDefaultTaxClass
		}
		inUse, err := uc.skuChecker.ExistsWithTaxClass(ctx, id)
		if err != nil {
			return err
		}
		if inUse {
			return taxDomain.ErrTaxClassInUse
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteTaxClass: %w", err)
	}
	return nil
}
