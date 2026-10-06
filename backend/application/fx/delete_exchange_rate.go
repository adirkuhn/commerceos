package fx

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	"gorm.io/gorm"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteExchangeRateUseCase struct {
	db     *gorm.DB
	repo   fxDomain.ExchangeRateRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteExchangeRateUseCase(db *gorm.DB, repo fxDomain.ExchangeRateRepository, outbox eventrecord.Enqueuer) *DeleteExchangeRateUseCase {
	return &DeleteExchangeRateUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteExchangeRateUseCase) Execute(ctx context.Context, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByCurrency(ctx, currencyCode)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, currencyCode); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ExchangeRateDeleted, currencyCode,
			eventrecord.Data{"rate_basis_points": previousRate(existing)})
	})
	if err != nil {
		return fmt.Errorf("deleteExchangeRate: %w", err)
	}
	return nil
}
