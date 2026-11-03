package integrations

import (
	"fmt"
	"time"
)

// Origin is how a sales order came to exist: made in CommerceOS (manual,
// WhatsApp, the built-in storefront) or imported from another system (POS,
// a shop). It lives here, not on the order (COM-15): the order holds order
// data only.
type Origin string

const (
	OriginPOS         Origin = "pos"
	OriginManual      Origin = "manual"
	OriginWooCommerce Origin = "woocommerce"
	OriginBol         Origin = "bol"
	OriginWhatsApp    Origin = "whatsapp"
	OriginInternal    Origin = "internal"
)

// originRule says what reference an origin carries: a shop order points at
// the connection it came from plus that shop's order id; a POS sale has the
// POS's own id but no connection row; an order made here has neither.
type originRule struct{ connection, externalID bool }

var originRules = map[Origin]originRule{
	OriginWooCommerce: {connection: true, externalID: true},
	OriginBol:         {connection: true, externalID: true},
	OriginPOS:         {externalID: true},
	OriginManual:      {},
	OriginWhatsApp:    {},
	OriginInternal:    {},
}

// SalesOrderOrigin is the one row per sales order that says where it came
// from and, for an imported order, which external order it is. Generic
// across integrations: a new one adds an Origin, never a column. The
// connection is stable (disconnect keeps it, reconnect reuses it), so the
// link survives an integration being reconnected.
type SalesOrderOrigin struct {
	ID           string
	TenantID     string
	SalesOrderID string
	Origin       Origin
	// ConnectionID is the integration connection a shop order came from;
	// nil for every origin without one.
	ConnectionID *string
	// ExternalOrderID is the other system's own id for this order; empty
	// for an order made in CommerceOS.
	ExternalOrderID string
	// ExternalKey is a second reference some systems give (Woo's order_key);
	// empty when there is none.
	ExternalKey string
	CreatedAt   time.Time
}

type SalesOrderOriginInput struct {
	TenantID        string
	SalesOrderID    string
	Origin          Origin
	ConnectionID    *string
	ExternalOrderID string
	ExternalKey     string
	CreatedAt       time.Time
}

// NewSalesOrderOrigin validates in against its origin's rule.
func NewSalesOrderOrigin(in SalesOrderOriginInput) (SalesOrderOrigin, error) {
	rule, ok := originRules[in.Origin]
	switch {
	case !ok:
		return SalesOrderOrigin{}, fmt.Errorf("unknown origin %q: %w", in.Origin, ErrSalesOrderOriginInvalid)
	case in.SalesOrderID == "":
		return SalesOrderOrigin{}, fmt.Errorf("no sales order: %w", ErrSalesOrderOriginInvalid)
	case rule.connection != (in.ConnectionID != nil && *in.ConnectionID != ""):
		return SalesOrderOrigin{}, fmt.Errorf("%s order: connection mismatch: %w", in.Origin, ErrSalesOrderOriginInvalid)
	case rule.externalID != (in.ExternalOrderID != ""):
		return SalesOrderOrigin{}, fmt.Errorf("%s order: external id mismatch: %w", in.Origin, ErrSalesOrderOriginInvalid)
	}
	return SalesOrderOrigin{
		TenantID: in.TenantID, SalesOrderID: in.SalesOrderID, Origin: in.Origin, ConnectionID: in.ConnectionID,
		ExternalOrderID: in.ExternalOrderID, ExternalKey: in.ExternalKey, CreatedAt: in.CreatedAt,
	}, nil
}

// IsExternal reports whether the order was imported from another system.
func (o SalesOrderOrigin) IsExternal() bool { return o.ExternalOrderID != "" }

// ClaimConnection gives a shop order whose connection was lost (migration
// 246 could not tell which of several connections it came from) the
// connection it was just re-imported through.
func (o *SalesOrderOrigin) ClaimConnection(connectionID string) error {
	if !originRules[o.Origin].connection || o.ConnectionID != nil || connectionID == "" {
		return fmt.Errorf("%s order %s: cannot claim connection %s: %w", o.Origin, o.SalesOrderID, connectionID, ErrSalesOrderOriginInvalid)
	}
	o.ConnectionID = &connectionID
	return nil
}
