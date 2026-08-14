package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListTaxRatesUseCase struct {
	db   *gorm.DB
	repo taxDomain.TaxRateRepository
}

func NewListTaxRatesUseCase(db *gorm.DB, repo taxDomain.TaxRateRepository) *ListTaxRatesUseCase {
	return &ListTaxRatesUseCase{db: db, repo: repo}
}

func (uc *ListTaxRatesUseCase) Execute(ctx context.Context) ([]taxDomain.TaxRate, error) {
	var rates []taxDomain.TaxRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		rates, err = uc.repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listTaxRates: %w", err)
	}
	return rates, nil
}
