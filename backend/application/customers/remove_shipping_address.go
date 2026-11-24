package customers

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RemoveShippingAddressUseCase removes one of a customer's shipping
// addresses; removing the default makes the next one the default.
type RemoveShippingAddressUseCase struct{ d ShippingAddressDeps }

func NewRemoveShippingAddressUseCase(d ShippingAddressDeps) *RemoveShippingAddressUseCase {
	return &RemoveShippingAddressUseCase{d: d}
}

func (uc *RemoveShippingAddressUseCase) Execute(ctx context.Context, customerID, id string) error {
	err := tenant.WithTenant(ctx, uc.d.DB, func(ctx context.Context) error {
		customer, err := uc.d.Customers.FindByID(ctx, customerID)
		if err != nil {
			return err
		}
		book, err := uc.d.Addresses.ListByCustomer(ctx, customerID)
		if err != nil {
			return err
		}
		next, err := book.NextDefaultAfterRemoving(id)
		if err != nil {
			return err
		}
		if err := uc.d.Addresses.Delete(ctx, customerID, id); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.d.Outbox, events.CustomerShippingAddressRemoved, id, eventrecord.Data{"customer_id": customerID}); err != nil {
			return err
		}
		if next != "" {
			if err := moveDefault(ctx, uc.d, customerID, next); err != nil {
				return err
			}
		}
		return syncShippingCopy(ctx, uc.d, customer)
	})
	if err != nil {
		return fmt.Errorf("removeShippingAddress: %w", err)
	}
	return nil
}
