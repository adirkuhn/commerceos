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

type CreateTaxClassInput struct {
	Name      string
	IsDefault bool
}

type CreateTaxClassUseCase struct {
	db     *gorm.DB
	repo   taxDomain.TaxClassRepository
	outbox eventrecord.Enqueuer
}

func NewCreateTaxClassUseCase(db *gorm.DB, repo taxDomain.TaxClassRepository, outbox eventrecord.Enqueuer) *CreateTaxClassUseCase {
	return &CreateTaxClassUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *CreateTaxClassUseCase) Execute(ctx context.Context, in CreateTaxClassInput) (*taxDomain.TaxClass, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("createTaxClass: %w", taxDomain.ErrTaxClassNameRequired)
	}

	var class taxDomain.TaxClass
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if in.IsDefault {
			if err := unsetExistingDefault(ctx, uc.repo, uc.outbox, ""); err != nil {
				return err
			}
		}
		now := time.Now()
		class = taxDomain.TaxClass{
			TenantID: tenant.IDFromContext(ctx), Name: name, IsDefault: in.IsDefault,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Create(ctx, &class); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.TaxClassCreated, class.ID, eventrecord.Data{
			"name": class.Name, "is_default": class.IsDefault,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createTaxClass: %w", err)
	}
	return &class, nil
}

// unsetExistingDefault flips off whichever TaxClass currently has
// IsDefault=true (if any, and it isn't excludeID -- Update re-saving the
// class that's already the default) so a tenant never ends up with two.
// Shared by Create and Update, since setting a new default is the same
// operation either way.
// TaxClassChangeReasonDefaultMoved: data.reason of a tax.tax_class.updated
// the system raised when another class became the default.
const TaxClassChangeReasonDefaultMoved = "default_moved"

func unsetExistingDefault(ctx context.Context, repo taxDomain.TaxClassRepository, outbox eventrecord.Enqueuer, excludeID string) error {
	classes, err := repo.List(ctx)
	if err != nil {
		return err
	}
	for _, c := range classes {
		if c.IsDefault && c.ID != excludeID {
			c.IsDefault = false
			c.UpdatedAt = time.Now()
			if err := repo.Update(ctx, &c); err != nil {
				return err
			}
			if err := eventrecord.Record(ctx, outbox, events.TaxClassUpdated, c.ID,
				eventrecord.NewChanges().Set("is_default", true, false),
				eventrecord.Data{"reason": TaxClassChangeReasonDefaultMoved}); err != nil {
				return err
			}
		}
	}
	return nil
}
