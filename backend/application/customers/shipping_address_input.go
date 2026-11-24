package customers

import (
	"context"
	"reflect"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
)

// ShippingAddressInput is what staff fill in for a shipping address.
type ShippingAddressInput struct {
	Label   string
	Address customersDomain.Address
}

func (in ShippingAddressInput) applyTo(a *customersDomain.ShippingAddress) {
	a.Label, a.Address = in.Label, in.Address
	a.Address.Email = ""
}

// ShippingAddressDeps are the shipping address use cases' ports.
type ShippingAddressDeps struct {
	DB        *gorm.DB
	Customers customersDomain.CustomerRepository
	Addresses customersDomain.ShippingAddressRepository
	Outbox    eventrecord.Enqueuer
}

// syncShippingCopy keeps customer.Shipping equal to where goods go
// (Customer.EffectiveShipping) after the book changed, saving the customer
// only when the copy moved.
func syncShippingCopy(ctx context.Context, d ShippingAddressDeps, c *customersDomain.Customer) error {
	book, err := d.Addresses.ListByCustomer(ctx, c.ID)
	if err != nil {
		return err
	}
	effective := c.EffectiveShipping(book.Default())
	if effective == c.Shipping {
		return nil
	}
	c.Shipping, c.DateModified = effective, time.Now()
	return d.Customers.Update(ctx, c)
}

// moveDefault makes id the customer's default and records it.
func moveDefault(ctx context.Context, d ShippingAddressDeps, customerID, id string) error {
	if err := d.Addresses.SetDefault(ctx, customerID, id); err != nil {
		return err
	}
	return eventrecord.Record(ctx, d.Outbox, events.CustomerShippingAddressDefaultChanged, id, eventrecord.Data{"customer_id": customerID})
}

// changedShippingFields names the fields that differ -- names only, the
// values are personal data.
func changedShippingFields(before, after customersDomain.ShippingAddress) []string {
	b, a := before.Address, after.Address
	fields := []struct {
		name          string
		before, after any
	}{
		{"address_1", b.Address1, a.Address1},
		{"address_2", b.Address2, a.Address2},
		{"city", b.City, a.City},
		{"company", b.Company, a.Company},
		{"country", b.Country, a.Country},
		{"first_name", b.FirstName, a.FirstName},
		{"label", before.Label, after.Label},
		{"last_name", b.LastName, a.LastName},
		{"phone", b.Phone, a.Phone},
		{"postcode", b.Postcode, a.Postcode},
		{"state", b.State, a.State},
	}
	var out []string
	for _, f := range fields {
		if !reflect.DeepEqual(f.before, f.after) {
			out = append(out, f.name)
		}
	}
	return out
}
