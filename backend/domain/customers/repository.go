package customers

import "context"

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	FindByID(ctx context.Context, id string) (*Customer, error)
	// FindByWooCustomerID looks up a customer already linked to a Woo
	// customer -- used by order import to resolve/attach the buyer without
	// creating a duplicate. Nil (not an error) when none exists, same
	// convention as oms.SalesOrderRepository.FindByExternalID.
	FindByWooCustomerID(ctx context.Context, wooCustomerID int64) (*Customer, error)
	// FindByEmail looks up a customer by exact email match (case-insensitive)
	// -- used by WooCommerce order import to find-or-create the buyer
	// behind a guest checkout (no linked Woo account) by their billing
	// email, so a named guest order still attaches to a real Customer
	// instead of staying an unlinked billing snapshot forever. Nil (not an
	// error) when none exists, same convention as FindByWooCustomerID.
	FindByEmail(ctx context.Context, email string) (*Customer, error)
	// FindByPhoneE164 looks up a customer by exact PhoneE164 match -- used
	// by application/whatsapp.HandleInboundMessageUseCase to resolve the
	// customer behind an inbound message's sender number. Nil (not an
	// error) when none exists, same convention as FindByEmail.
	FindByPhoneE164(ctx context.Context, phoneE164 string) (*Customer, error)
	List(ctx context.Context) ([]Customer, error)
	// Search returns up to limit customers whose name or email matches
	// query -- name matching is accent-insensitive + typo-tolerant
	// (pg_trgm), email matching is plain substring (fuzzy-matching an
	// email produces confusing near-misses for what's really an exact
	// lookup). Ranked best match first. Powers global search (Cmd+K).
	// Empty query returns nil, not an error.
	Search(ctx context.Context, query string, limit int) ([]Customer, error)
	Update(ctx context.Context, c *Customer) error
	Delete(ctx context.Context, id string) error
}
