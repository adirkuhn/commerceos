package storefront

import (
	"context"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (uc *CheckoutCartUseCase) loadOpenCart(ctx context.Context, token string) (*storefrontDomain.Cart, error) {
	var cart *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		if found.IsCheckedOut() {
			return storefrontDomain.ErrCartAlreadyCheckedOut
		}
		if len(found.Items) == 0 {
			return storefrontDomain.ErrCartEmpty
		}
		cart = found
		return nil
	})
	return cart, err
}

// resolveWarehouse looks up the channel's configured
// SalesChannel.DefaultWarehouseID -- every SalesOrder needs a real
// warehouse to decrement stock from (sales_orders.warehouse_id is a
// NOT NULL FK), and unlike a WooCommerce/Bol import (which carries its
// own Connection.DefaultWarehouseID), an internal-storefront checkout has
// no other source for one.
func (uc *CheckoutCartUseCase) resolveWarehouse(ctx context.Context, salesChannelID string) (string, error) {
	var warehouseID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := findInternalChannel(ctx, uc.channelRepo, salesChannelID)
		if err != nil {
			return err
		}
		if channel.DefaultWarehouseID == nil {
			return storefrontDomain.ErrChannelWarehouseNotConfigured
		}
		warehouseID = *channel.DefaultWarehouseID
		return nil
	})
	return warehouseID, err
}

func (uc *CheckoutCartUseCase) resolveCustomer(ctx context.Context, email string, billing, shipping CheckoutAddressInput) (string, error) {
	var existing *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.customerRepo.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		existing = found
		return nil
	})
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.ID, nil
	}

	created, err := uc.createCustomer.Execute(ctx, appcustomers.CreateCustomerInput{
		Email: email, FirstName: billing.FirstName, LastName: billing.LastName,
		Billing:  toCustomerAddress(billing, email),
		Shipping: toCustomerAddress(shipping, ""),
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

// resolveShipping re-resolves the real shipping options for shipping
// itself (never trusts a client-supplied price -- see CheckoutInput's doc
// comment on ShippingMethodTitle) and picks wantTitle among them if it's
// non-empty and still matches one, else the cheapest. No options
// available at all (see shippingOptionsResolver's doc comment for every
// reason that happens) falls back to (0, ""), never blocking checkout.
func (uc *CheckoutCartUseCase) resolveShipping(ctx context.Context, shipping CheckoutAddressInput, wantTitle string) (int64, string) {
	options := uc.shippingResolver.resolve(ctx, shipping)
	if len(options) == 0 {
		return 0, ""
	}
	if wantTitle != "" {
		for _, o := range options {
			if o.Title == wantTitle {
				return o.AmountMinorUnits, o.Title
			}
		}
	}
	cheapest := options[0]
	for _, o := range options[1:] {
		if o.AmountMinorUnits < cheapest.AmountMinorUnits {
			cheapest = o
		}
	}
	return cheapest.AmountMinorUnits, cheapest.Title
}

// shippingMethodSlug is the machine-readable counterpart to the human
// title estimateShipping returns -- empty when no estimate was applied,
// same "both empty means no shipping method was recorded" convention
// oms.SalesOrder.ShippingMethod's own doc comment describes.
func shippingMethodSlug(shippingAmount int64) string {
	if shippingAmount == 0 {
		return ""
	}
	return "myparcel"
}

func toOMSAddress(in CheckoutAddressInput, email string) omsDomain.Address {
	return omsDomain.Address{
		FirstName: in.FirstName, LastName: in.LastName, Company: in.Company,
		Address1: in.Address1, Address2: in.Address2, City: in.City, State: in.State,
		Postcode: in.Postcode, Country: in.Country, Email: email, Phone: in.Phone,
	}
}

func toCustomerAddress(in CheckoutAddressInput, email string) customersDomain.Address {
	return customersDomain.Address{
		FirstName: in.FirstName, LastName: in.LastName, Company: in.Company,
		Address1: in.Address1, Address2: in.Address2, City: in.City, State: in.State,
		Postcode: in.Postcode, Country: in.Country, Email: email, Phone: in.Phone,
	}
}
