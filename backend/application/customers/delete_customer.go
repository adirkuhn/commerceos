package customers

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
	outbox       eventrecord.Enqueuer
}

func NewDeleteCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository, outbox eventrecord.Enqueuer) *DeleteCustomerUseCase {
	return &DeleteCustomerUseCase{db: db, customerRepo: customerRepo, outbox: outbox}
}

func (uc *DeleteCustomerUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.customerRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.customerRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CustomerDeleted, id, eventrecord.Data{"woo_customer_id": existing.WooCustomerID})
	})
	if err != nil {
		return fmt.Errorf("deleteCustomer: %w", err)
	}
	return nil
}
