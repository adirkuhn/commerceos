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

type UpdateTaxClassInput struct {
	ID        string
	Name      string
	IsDefault bool
}

type UpdateTaxClassUseCase struct {
	db     *gorm.DB
	repo   taxDomain.TaxClassRepository
	outbox eventrecord.Enqueuer
}

func NewUpdateTaxClassUseCase(db *gorm.DB, repo taxDomain.TaxClassRepository, outbox eventrecord.Enqueuer) *UpdateTaxClassUseCase {
	return &UpdateTaxClassUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *UpdateTaxClassUseCase) Execute(ctx context.Context, in UpdateTaxClassInput) (*taxDomain.TaxClass, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("updateTaxClass: %w", taxDomain.ErrTaxClassNameRequired)
	}

	var class *taxDomain.TaxClass
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if in.IsDefault && !existing.IsDefault {
			if err := unsetExistingDefault(ctx, uc.repo, uc.outbox, existing.ID); err != nil {
				return err
			}
		}
		before := *existing
		existing.Name = name
		existing.IsDefault = in.IsDefault
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		class = existing
		return eventrecord.Record(ctx, uc.outbox, events.TaxClassUpdated, existing.ID, eventrecord.NewChanges().
			Set("name", before.Name, existing.Name).
			Set("is_default", before.IsDefault, existing.IsDefault))
	})
	if err != nil {
		return nil, fmt.Errorf("updateTaxClass: %w", err)
	}
	return class, nil
}
