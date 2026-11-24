package customers

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UpdateShippingAddressUseCase edits one of a customer's shipping
// addresses; editing the default one updates the customer's copy.
type UpdateShippingAddressUseCase struct{ d ShippingAddressDeps }

func NewUpdateShippingAddressUseCase(d ShippingAddressDeps) *UpdateShippingAddressUseCase {
	return &UpdateShippingAddressUseCase{d: d}
}

func (uc *UpdateShippingAddressUseCase) Execute(ctx context.Context, customerID, id string, in ShippingAddressInput) (*customersDomain.ShippingAddress, error) {
	var updated customersDomain.ShippingAddress
	err := tenant.WithTenant(ctx, uc.d.DB, func(ctx context.Context) error {
		customer, err := uc.d.Customers.FindByID(ctx, customerID)
		if err != nil {
			return err
		}
		book, err := uc.d.Addresses.ListByCustomer(ctx, customerID)
		if err != nil {
			return err
		}
		before, err := book.Find(id)
		if err != nil {
			return err
		}
		updated = before
		in.applyTo(&updated)
		if err := updated.Validate(); err != nil {
			return err
		}
		updated.UpdatedAt = time.Now()
		if err := uc.d.Addresses.Update(ctx, &updated); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.d.Outbox, events.CustomerShippingAddressUpdated, id,
			eventrecord.Data{"customer_id": customerID, "changed_fields": changedShippingFields(before, updated)}); err != nil {
			return err
		}
		return syncShippingCopy(ctx, uc.d, customer)
	})
	if err != nil {
		return nil, fmt.Errorf("updateShippingAddress: %w", err)
	}
	return &updated, nil
}
