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

type DeleteExternalTaxClassLinkUseCase struct {
	db     *gorm.DB
	repo   taxDomain.ExternalTaxClassLinkRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteExternalTaxClassLinkUseCase(db *gorm.DB, repo taxDomain.ExternalTaxClassLinkRepository, outbox eventrecord.Enqueuer) *DeleteExternalTaxClassLinkUseCase {
	return &DeleteExternalTaxClassLinkUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteExternalTaxClassLinkUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ExternalTaxClassLinkDeleted, id, eventrecord.Data{
			"provider": existing.Provider, "external_slug": existing.ExternalSlug, "tax_class_id": existing.TaxClassID,
		})
	})
	if err != nil {
		return fmt.Errorf("deleteExternalTaxClassLink: %w", err)
	}
	return nil
}
