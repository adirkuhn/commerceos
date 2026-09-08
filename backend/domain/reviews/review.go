// Package reviews holds shopper-submitted product reviews for the internal
// storefront -- one row per review, always scoped to the sales channel it
// was left on (a product can be sold on multiple internal channels with
// independent review sets, same scoping as domain/storefront.Cart).
package reviews

import "time"

const (
	MinRating = 1
	MaxRating = 5
)

// Review is a shopper's rating+comment on a product. CustomerName/Email are
// captured as free text at submission time (there is no storefront login,
// same as Checkout's guest-only Customer lookup by email) -- not a FK to
// domain/customers.Customer, which may not even have a matching row yet.
type Review struct {
	ID             string
	TenantID       string
	SalesChannelID string
	ProductID      string
	CustomerName   string
	CustomerEmail  string
	Rating         int
	Title          string
	Body           string
	// Verified is set at submission time by checking for a completed order
	// on this channel, from this email, containing this product -- never
	// re-checked afterwards, so it reflects purchase history as of when the
	// review was written.
	Verified  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r Review) ValidRating() bool {
	return r.Rating >= MinRating && r.Rating <= MaxRating
}

// Summary aggregates a product's reviews for display on its PDP -- Count
// zero means Average is meaningless (0, not NaN) and callers must check
// Count before showing Average.
type Summary struct {
	Average float64
	Count   int
}
