package customers

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RemoveContactPersonUseCase deletes one of a customer's contact persons.
type RemoveContactPersonUseCase struct {
	db      *gorm.DB
	persons customersDomain.ContactPersonRepository
	outbox  eventrecord.Enqueuer
}

func NewRemoveContactPersonUseCase(db *gorm.DB, persons customersDomain.ContactPersonRepository, outbox eventrecord.Enqueuer) *RemoveContactPersonUseCase {
	return &RemoveContactPersonUseCase{db: db, persons: persons, outbox: outbox}
}

func (uc *RemoveContactPersonUseCase) Execute(ctx context.Context, customerID, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.persons.Delete(ctx, customerID, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CustomerContactPersonRemoved, id, eventrecord.Data{"customer_id": customerID})
	})
	if err != nil {
		return fmt.Errorf("removeContactPerson: %w", err)
	}
	return nil
}
