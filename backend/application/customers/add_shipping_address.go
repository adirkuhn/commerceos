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

// AddShippingAddressUseCase adds a shipping address to a customer; the
// first one, or one added asDefault, becomes the default.
type AddShippingAddressUseCase struct{ d ShippingAddressDeps }

func NewAddShippingAddressUseCase(d ShippingAddressDeps) *AddShippingAddressUseCase {
	return &AddShippingAddressUseCase{d: d}
}

func (uc *AddShippingAddressUseCase) Execute(ctx context.Context, customerID string, in ShippingAddressInput, asDefault bool) (*customersDomain.ShippingAddress, error) {
	var draft customersDomain.ShippingAddress
	in.applyTo(&draft)
	if err := draft.Validate(); err != nil {
		return nil, fmt.Errorf("addShippingAddress: %w", err)
	}
	var added customersDomain.ShippingAddress
	err := tenant.WithTenant(ctx, uc.d.DB, func(ctx context.Context) error {
		customer, err := uc.d.Customers.FindByID(ctx, customerID)
		if err != nil {
			return err
		}
		book, err := uc.d.Addresses.ListByCustomer(ctx, customerID)
		if err != nil {
			return err
		}
		if added, err = book.Add(draft, asDefault); err != nil {
			return err
		}
		now := time.Now()
		added.TenantID, added.CustomerID, added.CreatedAt, added.UpdatedAt = tenant.IDFromContext(ctx), customerID, now, now
		// Stored without the flag, then moved onto it: the one-default
		// rule holds at every statement.
		becomesDefault := added.IsDefault
		added.IsDefault = false
		if err := uc.d.Addresses.Create(ctx, &added); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.d.Outbox, events.CustomerShippingAddressAdded, added.ID,
			eventrecord.Data{"customer_id": customerID, "default": becomesDefault}); err != nil {
			return err
		}
		if becomesDefault {
			if err := uc.d.Addresses.SetDefault(ctx, customerID, added.ID); err != nil {
				return err
			}
			added.IsDefault = true
		}
		return syncShippingCopy(ctx, uc.d, customer)
	})
	if err != nil {
		return nil, fmt.Errorf("addShippingAddress: %w", err)
	}
	return &added, nil
}
