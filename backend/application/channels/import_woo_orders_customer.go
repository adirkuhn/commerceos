package channels

import (
	"context"
	"time"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// resolveOrCreateCustomer finds the customer behind a Woo order -- by their
// linked Woo account when there is one, otherwise by billing email (Woo
// requires a billing email even for guest checkout) -- creating one from
// the order's own billing details when no match exists at all. Without
// this, a named guest-checkout buyer (the overwhelming majority of orders
// on stores that don't require an account) never gets attached to a real
// Customer: no link on the order, no order history or lifetime-spend on
// the Customer page. Must be called with a context already bound to an
// open transaction (see tenant.WithTenant) -- it never opens its own.
func (uc *ImportWooOrdersUseCase) resolveOrCreateCustomer(ctx context.Context, wo integrationsDomain.WooOrder) (*string, error) {
	var wooCustomerID *int64
	switch {
	case wo.CustomerID != 0:
		id := int64(wo.CustomerID)
		wooCustomerID = &id
		c, err := uc.customerRepo.FindByWooCustomerID(ctx, id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			return &c.ID, nil
		}
	case wo.Billing.Email != "":
		c, err := uc.customerRepo.FindByEmail(ctx, wo.Billing.Email)
		if err != nil {
			return nil, err
		}
		if c != nil {
			return &c.ID, nil
		}
	default:
		// No Woo account and no billing email -- nothing to key a customer
		// off of. Leave the order unlinked, same as before this fix.
		return nil, nil
	}

	now := time.Now()
	customer := customersDomain.Customer{
		TenantID: tenant.IDFromContext(ctx), WooCustomerID: wooCustomerID,
		Email: wo.Billing.Email, FirstName: wo.Billing.FirstName, LastName: wo.Billing.LastName,
		Billing: wooAddressToCustomerAddress(wo.Billing), Shipping: wooAddressToCustomerAddress(wo.Shipping),
		DateCreated: now, DateModified: now,
	}
	if err := uc.customerRepo.Create(ctx, &customer); err != nil {
		return nil, err
	}
	return &customer.ID, nil
}

func wooAddressToOMS(a integrationsDomain.WooAddress) omsDomain.Address {
	return omsDomain.Address{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Email: a.Email, Phone: a.Phone,
	}
}
