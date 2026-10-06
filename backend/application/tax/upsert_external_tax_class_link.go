package tax

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"strings"
	"time"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// UpsertExternalTaxClassLinkInput.ExternalSlug deliberately has no
// required-non-empty validation -- "" is a legitimate value (e.g.
// WooCommerce's own Standard rate).
type UpsertExternalTaxClassLinkInput struct {
	Provider     string
	ExternalSlug string
	TaxClassID   string
}

type UpsertExternalTaxClassLinkUseCase struct {
	db        *gorm.DB
	linkRepo  taxDomain.ExternalTaxClassLinkRepository
	classRepo taxDomain.TaxClassRepository
	outbox    eventrecord.Enqueuer
}

func NewUpsertExternalTaxClassLinkUseCase(db *gorm.DB, linkRepo taxDomain.ExternalTaxClassLinkRepository, classRepo taxDomain.TaxClassRepository, outbox eventrecord.Enqueuer) *UpsertExternalTaxClassLinkUseCase {
	return &UpsertExternalTaxClassLinkUseCase{db: db, linkRepo: linkRepo, classRepo: classRepo, outbox: outbox}
}

func (uc *UpsertExternalTaxClassLinkUseCase) Execute(ctx context.Context, in UpsertExternalTaxClassLinkInput) (*taxDomain.ExternalTaxClassLink, error) {
	provider := strings.TrimSpace(in.Provider)
	if provider == "" {
		return nil, fmt.Errorf("upsertExternalTaxClassLink: %w", taxDomain.ErrProviderRequired)
	}

	var link taxDomain.ExternalTaxClassLink
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.classRepo.FindByID(ctx, in.TaxClassID); err != nil {
			return err
		}
		now := time.Now()
		link = taxDomain.ExternalTaxClassLink{
			TenantID: tenant.IDFromContext(ctx), Provider: provider, ExternalSlug: in.ExternalSlug,
			TaxClassID: in.TaxClassID, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.linkRepo.Upsert(ctx, &link); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ExternalTaxClassLinkSet, link.ID, eventrecord.Data{
			"provider": link.Provider, "external_slug": link.ExternalSlug, "tax_class_id": link.TaxClassID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("upsertExternalTaxClassLink: %w", err)
	}
	return &link, nil
}
