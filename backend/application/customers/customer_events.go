package customers

import (
	"context"
	"reflect"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
)

// recordCustomerUpdated records customers.customer.updated: non-personal
// fields with their values, personal ones by name only (data.changed_fields)
// so no personal data lands in the event stream.
func recordCustomerUpdated(ctx context.Context, outbox eventrecord.Enqueuer, before, after customersDomain.Customer) error {
	changes := eventrecord.NewChanges().
		Set("woo_customer_id", before.WooCustomerID, after.WooCustomerID).
		Set("role", before.Role, after.Role).
		Set("is_paying_customer", before.IsPayingCustomer, after.IsPayingCustomer).
		Set("customer_type", before.Type, after.Type).
		Set("company_name", before.CompanyName, after.CompanyName).
		Set("client_number", before.ClientNumber, after.ClientNumber).
		Set("kvk_number", before.KVKNumber, after.KVKNumber).
		Set("vat_number", before.VATNumber, after.VATNumber)
	opts := []eventrecord.Option{changes}
	if personal := changedPersonalFields(before, after); len(personal) > 0 {
		opts = append(opts, eventrecord.Data{"changed_fields": personal})
	}
	return eventrecord.Record(ctx, outbox, events.CustomerUpdated, after.ID, opts...)
}

// changedPersonalFields names the personal fields that differ, sorted.
func changedPersonalFields(before, after customersDomain.Customer) []string {
	fields := []struct {
		name          string
		before, after any
	}{
		{"avatar_url", before.AvatarURL, after.AvatarURL},
		{"billing", before.Billing, after.Billing},
		{"customer_note", before.CustomerNote, after.CustomerNote},
		{"email", before.Email, after.Email},
		{"first_name", before.FirstName, after.FirstName},
		{"last_name", before.LastName, after.LastName},
		{"shipping", before.Shipping, after.Shipping},
		{"username", before.Username, after.Username},
	}
	var out []string
	for _, f := range fields {
		if !reflect.DeepEqual(f.before, f.after) {
			out = append(out, f.name)
		}
	}
	return out
}
