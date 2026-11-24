package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetDefaultShippingAddressUseCase makes one of a customer's shipping
// addresses the default, and returns the book.
type SetDefaultShippingAddressUseCase struct{ d ShippingAddressDeps }

func NewSetDefaultShippingAddressUseCase(d ShippingAddressDeps) *SetDefaultShippingAddressUseCase {
	return &SetDefaultShippingAddressUseCase{d: d}
}

func (uc *SetDefaultShippingAddressUseCase) Execute(ctx context.Context, customerID, id string) (customersDomain.ShippingBook, error) {
	var book customersDomain.ShippingBook
	err := tenant.WithTenant(ctx, uc.d.DB, func(ctx context.Context) error {
		customer, err := uc.d.Customers.FindByID(ctx, customerID)
		if err != nil {
			return err
		}
		if book, err = uc.d.Addresses.ListByCustomer(ctx, customerID); err != nil {
			return err
		}
		chosen, err := book.Find(id)
		if err != nil {
			return err
		}
		if !chosen.IsDefault {
			if err := moveDefault(ctx, uc.d, customerID, id); err != nil {
				return err
			}
			if err := syncShippingCopy(ctx, uc.d, customer); err != nil {
				return err
			}
		}
		book, err = uc.d.Addresses.ListByCustomer(ctx, customerID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("setDefaultShippingAddress: %w", err)
	}
	return book, nil
}
