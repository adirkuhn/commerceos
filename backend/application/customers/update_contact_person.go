package customers

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// UpdateContactPersonUseCase edits one of a customer's contact persons.
type UpdateContactPersonUseCase struct {
	db      *gorm.DB
	persons customersDomain.ContactPersonRepository
	outbox  eventrecord.Enqueuer
}

func NewUpdateContactPersonUseCase(db *gorm.DB, persons customersDomain.ContactPersonRepository, outbox eventrecord.Enqueuer) *UpdateContactPersonUseCase {
	return &UpdateContactPersonUseCase{db: db, persons: persons, outbox: outbox}
}

func (uc *UpdateContactPersonUseCase) Execute(ctx context.Context, customerID, id string, in ContactPersonInput) (*customersDomain.ContactPerson, error) {
	var person customersDomain.ContactPerson
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.persons.FindByID(ctx, customerID, id)
		if err != nil {
			return err
		}
		before := *existing
		in.applyTo(existing)
		if err := existing.Validate(); err != nil {
			return err
		}
		existing.UpdatedAt = time.Now()
		if err := uc.persons.Update(ctx, existing); err != nil {
			return err
		}
		person = *existing
		return eventrecord.Record(ctx, uc.outbox, events.CustomerContactPersonUpdated, id,
			eventrecord.Data{"customer_id": customerID, "changed_fields": changedContactFields(before, person)})
	})
	if err != nil {
		return nil, fmt.Errorf("updateContactPerson: %w", err)
	}
	return &person, nil
}
