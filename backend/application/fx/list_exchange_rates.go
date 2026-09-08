// Package fx holds the admin CRUD use cases for domain/fx.ExchangeRate --
// list/upsert/delete, tenant-scoped via tenant.WithTenant same as every
// other admin-managed resource in this codebase.
package fx

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListExchangeRatesUseCase struct {
	db   *gorm.DB
	repo fxDomain.ExchangeRateRepository
}

func NewListExchangeRatesUseCase(db *gorm.DB, repo fxDomain.ExchangeRateRepository) *ListExchangeRatesUseCase {
	return &ListExchangeRatesUseCase{db: db, repo: repo}
}

func (uc *ListExchangeRatesUseCase) Execute(ctx context.Context) ([]fxDomain.ExchangeRate, error) {
	var rates []fxDomain.ExchangeRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.repo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		rates = r
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listExchangeRates: %w", err)
	}
	return rates, nil
}
