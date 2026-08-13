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

	Email     string
	FirstName string
	LastName  string
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

	DateCreated  time.Time
	DateModified time.Time
}
