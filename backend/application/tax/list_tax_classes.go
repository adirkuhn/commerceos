package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListTaxClassesUseCase struct {
	db   *gorm.DB
	repo taxDomain.TaxClassRepository
}

func NewListTaxClassesUseCase(db *gorm.DB, repo taxDomain.TaxClassRepository) *ListTaxClassesUseCase {
	return &ListTaxClassesUseCase{db: db, repo: repo}
}

func (uc *ListTaxClassesUseCase) Execute(ctx context.Context) ([]taxDomain.TaxClass, error) {
	var classes []taxDomain.TaxClass
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		classes, err = uc.repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listTaxClasses: %w", err)
	}
	return classes, nil
}
