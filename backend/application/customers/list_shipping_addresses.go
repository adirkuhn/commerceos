package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListShippingAddressesUseCase lists a customer's shipping addresses,
// oldest first.
type ListShippingAddressesUseCase struct{ d ShippingAddressDeps }

func NewListShippingAddressesUseCase(d ShippingAddressDeps) *ListShippingAddressesUseCase {
	return &ListShippingAddressesUseCase{d: d}
}

func (uc *ListShippingAddressesUseCase) Execute(ctx context.Context, customerID string) (customersDomain.ShippingBook, error) {
	var book customersDomain.ShippingBook
	err := tenant.WithTenant(ctx, uc.d.DB, func(ctx context.Context) error {
		if _, err := uc.d.Customers.FindByID(ctx, customerID); err != nil {
			return err
		}
		var err error
		book, err = uc.d.Addresses.ListByCustomer(ctx, customerID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listShippingAddresses: %w", err)
	}
	return book, nil
}
