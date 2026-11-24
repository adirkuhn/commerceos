package customers

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// MaxShippingAddresses caps one customer's address book.
	MaxShippingAddresses = 20
	// MaxShippingLabelLength caps a label like "Warehouse" or "Home".
	MaxShippingLabelLength = 60
)

// ShippingAddress is one of the places a customer has goods delivered. A
// customer has any number of them; exactly one is the default while any
// exist, and Customer.Shipping holds a copy of the one in use (see
// Customer.EffectiveShipping) for everything that reads a single address.
type ShippingAddress struct {
	ID         string
	TenantID   string
	CustomerID string
	// Label tells the addresses apart, e.g. "Warehouse"; optional.
	Label string
	// Address.Email is unused: a shipping address has no email.
	Address   Address
	IsDefault bool
	// Position orders the book, oldest first.
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate requires an address line and a short label.
func (a ShippingAddress) Validate() error {
	if strings.TrimSpace(a.Address.Address1) == "" {
		return ErrShippingAddressLineRequired
	}
	if utf8.RuneCountInString(a.Label) > MaxShippingLabelLength {
		return ErrShippingLabelTooLong
	}
	return nil
}

// ShippingBook is a customer's shipping addresses in Position order.
type ShippingBook []ShippingAddress

// Add validates a and returns it ready to store at the end of the book:
// the first address is always the default, and asDefault makes a later one
// the default (the caller moves the flag off the old one).
func (b ShippingBook) Add(a ShippingAddress, asDefault bool) (ShippingAddress, error) {
	if err := a.Validate(); err != nil {
		return ShippingAddress{}, err
	}
	if len(b) >= MaxShippingAddresses {
		return ShippingAddress{}, ErrTooManyShippingAddresses
	}
	a.Position = 0
	for _, existing := range b {
		if existing.Position >= a.Position {
			a.Position = existing.Position + 1
		}
	}
	a.IsDefault = asDefault || len(b) == 0
	return a, nil
}

// Find returns the address with id, or ErrShippingAddressNotFound.
func (b ShippingBook) Find(id string) (ShippingAddress, error) {
	for _, a := range b {
		if a.ID == id {
			return a, nil
		}
	}
	return ShippingAddress{}, ErrShippingAddressNotFound
}

// Default is the default address, nil for an empty book.
func (b ShippingBook) Default() *ShippingAddress {
	for i := range b {
		if b[i].IsDefault {
			return &b[i]
		}
	}
	return nil
}

// NextDefaultAfterRemoving names the address that becomes the default when
// id is removed: the first remaining one when id is the default, else none
// (the default stays).
func (b ShippingBook) NextDefaultAfterRemoving(id string) (string, error) {
	removed, err := b.Find(id)
	if err != nil {
		return "", err
	}
	if !removed.IsDefault {
		return "", nil
	}
	for _, a := range b {
		if a.ID != id {
			return a.ID, nil
		}
	}
	return "", nil
}

// IsEmpty reports whether the address has nothing to deliver to -- an
// email alone doesn't count.
func (a Address) IsEmpty() bool {
	a.Email = ""
	return a == Address{}
}

// EffectiveShipping is where the customer's goods go: the billing address
// when ShipToBilling, else the default shipping address (none: empty).
// Shipping carries no email.
func (c Customer) EffectiveShipping(def *ShippingAddress) Address {
	var out Address
	switch {
	case c.ShipToBilling:
		out = c.Billing
	case def != nil:
		out = def.Address
	}
	out.Email = ""
	return out
}

type ShippingAddressRepository interface {
	// ListByCustomer returns the customer's book in Position order.
	ListByCustomer(ctx context.Context, customerID string) (ShippingBook, error)
	Create(ctx context.Context, a *ShippingAddress) error
	Update(ctx context.Context, a *ShippingAddress) error
	Delete(ctx context.Context, customerID, id string) error
	// SetDefault makes id the customer's only default address.
	SetDefault(ctx context.Context, customerID, id string) error
}
