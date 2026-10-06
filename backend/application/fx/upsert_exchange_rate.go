package fx

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	db     *gorm.DB
	repo   fxDomain.ExchangeRateRepository
	outbox eventrecord.Enqueuer
}

func NewUpsertExchangeRateUseCase(db *gorm.DB, repo fxDomain.ExchangeRateRepository, outbox eventrecord.Enqueuer) *UpsertExchangeRateUseCase {
	return &UpsertExchangeRateUseCase{db: db, repo: repo, outbox: outbox}
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
		previous, err := uc.repo.FindByCurrency(ctx, in.CurrencyCode)
		if err != nil {
			return err
		}
		now := time.Now()
		rate = fxDomain.ExchangeRate{
			TenantID: tenant.IDFromContext(ctx), CurrencyCode: in.CurrencyCode, RateBasisPoints: in.RateBasisPoints,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Upsert(ctx, &rate); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ExchangeRateSet, rate.CurrencyCode,
			eventrecord.NewChanges().Set("rate_basis_points", previousRate(previous), rate.RateBasisPoints))
	})
	if err != nil {
		return nil, fmt.Errorf("upsertExchangeRate: %w", err)
	}
	return &rate, nil
}

// previousRate is the rate before an upsert, nil when the currency had none.
func previousRate(r *fxDomain.ExchangeRate) *int64 {
	if r == nil {
		return nil
	}
	return &r.RateBasisPoints
}
