package fx

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteExchangeRateUseCase struct {
	db   *gorm.DB
	repo fxDomain.ExchangeRateRepository
}

func NewDeleteExchangeRateUseCase(db *gorm.DB, repo fxDomain.ExchangeRateRepository) *DeleteExchangeRateUseCase {
	return &DeleteExchangeRateUseCase{db: db, repo: repo}
}

func (uc *DeleteExchangeRateUseCase) Execute(ctx context.Context, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, currencyCode)
	})
	if err != nil {
		return fmt.Errorf("deleteExchangeRate: %w", err)
	}
	return nil
}
