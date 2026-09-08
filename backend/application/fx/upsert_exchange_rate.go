package fx

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpsertExchangeRateInput struct {
	CurrencyCode    string
	RateBasisPoints int64
}

type UpsertExchangeRateUseCase struct {
	db   *gorm.DB
	repo fxDomain.ExchangeRateRepository
}

func NewUpsertExchangeRateUseCase(db *gorm.DB, repo fxDomain.ExchangeRateRepository) *UpsertExchangeRateUseCase {
	return &UpsertExchangeRateUseCase{db: db, repo: repo}
}

func (uc *UpsertExchangeRateUseCase) Execute(ctx context.Context, in UpsertExchangeRateInput) (*fxDomain.ExchangeRate, error) {
	if !fxDomain.ValidCurrencyCode(in.CurrencyCode) {
		return nil, fmt.Errorf("upsertExchangeRate: %w", fxDomain.ErrInvalidCurrencyCode)
	}
	if in.RateBasisPoints <= 0 {
		return nil, fmt.Errorf("upsertExchangeRate: %w", fxDomain.ErrInvalidRate)
	}

	var rate fxDomain.ExchangeRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		rate = fxDomain.ExchangeRate{
			TenantID: tenant.IDFromContext(ctx), CurrencyCode: in.CurrencyCode, RateBasisPoints: in.RateBasisPoints,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.repo.Upsert(ctx, &rate)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertExchangeRate: %w", err)
	}
	return &rate, nil
}
