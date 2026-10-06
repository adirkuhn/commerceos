package tax

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteTaxRateUseCase struct {
	db     *gorm.DB
	repo   taxDomain.TaxRateRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteTaxRateUseCase(db *gorm.DB, repo taxDomain.TaxRateRepository, outbox eventrecord.Enqueuer) *DeleteTaxRateUseCase {
	return &DeleteTaxRateUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteTaxRateUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.TaxRateDeleted, id, eventrecord.Data{
			"tax_class_id": existing.TaxClassID, "country": existing.Country,
		})
	})
	if err != nil {
		return fmt.Errorf("deleteTaxRate: %w", err)
	}
	return nil
}
