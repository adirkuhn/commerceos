package shipping

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// ShippingPreferences is the tenant's own shipping choices that span
// connections -- today only which shipping connection (MyParcel, PostNL,
// ...) packing uses unless someone picks another one for a label. One row
// per tenant.
type ShippingPreferences struct {
	TenantID string
	// DefaultConnectionID is the merchant's explicit pick; nil means "not
	// chosen", in which case ResolveDefaultConnection falls back to the
	// oldest shipping connection.
	DefaultConnectionID *string
	UpdatedAt           time.Time
}

// ShippingPreferencesRepository persists the tenant's ShippingPreferences.
// Get returns nil, nil when the tenant never saved any.
type ShippingPreferencesRepository interface {
	Get(ctx context.Context) (*ShippingPreferences, error)
	Upsert(ctx context.Context, p *ShippingPreferences) error
}

// ConnectionCandidate is one shipping-capable connection as far as picking
// a default is concerned.
type ConnectionCandidate struct {
	ID        string
	CreatedAt time.Time
}

// ResolveDefaultConnection is which of conns packing uses by default: the
// configured one while it's still among conns, otherwise the oldest (so a
// tenant with a single shipping connection never has to configure
// anything). "" when there are no shipping connections at all.
func ResolveDefaultConnection(conns []ConnectionCandidate, configured *string) string {
	if configured != nil {
		for _, c := range conns {
			if c.ID == *configured {
				return c.ID
			}
		}
	}
	oldest := ""
	var oldestAt time.Time
	for _, c := range conns {
		if oldest == "" || c.CreatedAt.Before(oldestAt) || (c.CreatedAt.Equal(oldestAt) && c.ID < oldest) {
			oldest, oldestAt = c.ID, c.CreatedAt
		}
	}
	return oldest
}

// ErrNotAShippingConnection: the connection picked as the default can't
// create shipping labels (e.g. a Bol or Stripe connection).
var ErrNotAShippingConnection = errs.New("NOT_A_SHIPPING_CONNECTION", "shipping: this connection cannot create shipping labels")

// ErrShippingPreferencesForbidden: only an owner or admin may change which
// connection packing uses by default.
var ErrShippingPreferencesForbidden = errs.New("SHIPPING_PREFERENCES_FORBIDDEN", "only an owner or admin can change the default shipping provider")
