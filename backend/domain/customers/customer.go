package customers

import "time"

// Address mirrors WooCommerce's billing/shipping address object exactly --
// same shape as oms.Address, duplicated rather than imported since domain
// packages never import each other in this codebase.
type Address struct {
	FirstName string
	LastName  string
	Company   string
	Address1  string
	Address2  string
	City      string
	State     string
	Postcode  string
	Country   string
	// Email is only meaningful on Billing.
	Email string
	Phone string
}

// Customer mirrors WooCommerce's customer object field-for-field, so a
// customer synced from (or later pushed to) Woo round-trips cleanly.
type Customer struct {
	ID       string
	TenantID string
	// WooCustomerID is set once this customer is known to WooCommerce --
	// nil for a customer that only exists in CommerceOS (e.g. captured at
	// a POS register, never synced).
	WooCustomerID *int64

	// Type is company or private (see Type); empty reads as private.
	Type Type
	// CompanyName names a company customer (Type company). Billing.Company
	// is the address line, kept separately as Woo has it.
	CompanyName string
	// ClientNumber is the shop's own number for the customer, unique per
	// tenant; the repository numbers a new customer that has none.
	ClientNumber string
	// KVKNumber is the Dutch Chamber of Commerce number; VATNumber the
	// VAT ID. Both free text, for company customers.
	KVKNumber string
	VATNumber string

	// Email is optional.
	Email     string
	FirstName string
	LastName  string
	// PhoneE164 is a normalized E.164 number (no leading '+'), distinct
	// from Billing/Shipping.Phone (free-text, WooCommerce-shaped) --
	// blank until set explicitly (manual edit, or the WhatsApp
	// new-customer capture flow, see application/whatsapp). This is the
	// field an inbound WhatsApp message's sender number is matched
	// against to resolve an existing customer.
	PhoneE164 string
	// Username/Role mirror Woo's fields of the same name -- Role is Woo's
	// WordPress user role (e.g. "customer"), not a CommerceOS RBAC role.
	Username string
	Role     string

	Billing  Address
	Shipping Address

	// IsPayingCustomer mirrors Woo's field of the same name -- true once
	// this customer has completed at least one paid order.
	IsPayingCustomer bool
	AvatarURL        string
	// CustomerNote is a freeform staff note, editable from the customer
	// detail page -- not part of Woo's customer object (unlike every other
	// field on this struct), purely a CommerceOS addition.
	CustomerNote string

	DateCreated  time.Time
	DateModified time.Time
}
