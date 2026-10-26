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

type customerFinder interface {
	FindByID(ctx context.Context, id string) (*customersDomain.Customer, error)
}

// AddContactPersonUseCase adds a contact person to a customer.
type AddContactPersonUseCase struct {
	db        *gorm.DB
	customers customerFinder
	persons   customersDomain.ContactPersonRepository
	outbox    eventrecord.Enqueuer
}

func NewAddContactPersonUseCase(db *gorm.DB, customers customerFinder, persons customersDomain.ContactPersonRepository, outbox eventrecord.Enqueuer) *AddContactPersonUseCase {
	return &AddContactPersonUseCase{db: db, customers: customers, persons: persons, outbox: outbox}
}

func (uc *AddContactPersonUseCase) Execute(ctx context.Context, customerID string, in ContactPersonInput) (*customersDomain.ContactPerson, error) {
	var person customersDomain.ContactPerson
	in.applyTo(&person)
	if err := person.Validate(); err != nil {
		return nil, fmt.Errorf("addContactPerson: %w", err)
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.customers.FindByID(ctx, customerID); err != nil {
			return err
		}
		return addContactPerson(ctx, uc.persons, uc.outbox, customerID, &person)
	})
	if err != nil {
		return nil, fmt.Errorf("addContactPerson: %w", err)
	}
	return &person, nil
}

// addContactPerson stores a validated person for customerID and records it,
// inside the caller's transaction.
func addContactPerson(ctx context.Context, persons customersDomain.ContactPersonRepository, outbox eventrecord.Enqueuer, customerID string, p *customersDomain.ContactPerson) error {
	now := time.Now()
	p.TenantID, p.CustomerID, p.CreatedAt, p.UpdatedAt = tenant.IDFromContext(ctx), customerID, now, now
	if err := persons.Create(ctx, p); err != nil {
		return err
	}
	return eventrecord.Record(ctx, outbox, events.CustomerContactPersonAdded, p.ID, eventrecord.Data{"customer_id": customerID})
}
