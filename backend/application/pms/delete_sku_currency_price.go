package pms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteSKUCurrencyPriceUseCase struct {
	db   *gorm.DB
	repo   pmsDomain.SKUCurrencyPriceRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteSKUCurrencyPriceUseCase(db *gorm.DB, repo pmsDomain.SKUCurrencyPriceRepository, outbox eventrecord.Enqueuer) *DeleteSKUCurrencyPriceUseCase {
	return &DeleteSKUCurrencyPriceUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteSKUCurrencyPriceUseCase) Execute(ctx context.Context, skuID, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, skuID, currencyCode); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SKUCurrencyPriceDeleted, skuID, eventrecord.Data{"currency_code": currencyCode})
	})
	if err != nil {
		return fmt.Errorf("deleteSKUCurrencyPrice: %w", err)
	}
	return nil
}
