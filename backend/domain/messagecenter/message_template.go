// Package messagecenter holds merchant-configurable message templates and
// event-driven automation rules that send them -- the generic replacement
// for one-off hardcoded notification wording (see application/whatsapp's
// orderStatusMessage/deliveryMessage, left as-is, not migrated here yet).
// Sending itself reuses the existing domain/messaging.Client port; this
// package only owns "what to send" (MessageTemplate) and, in a later slice,
// "when to send it" (AutomationRule).
package messagecenter

import (
	"time"

	"github.com/adirkuhn/commerceos/domain/messaging"
)

// Category groups templates for the admin UI's filter/picker -- purely
// organizational, no behavior branches on it.
type Category string

const (
	CategoryPurchase    Category = "purchase"
	CategoryOrderStatus Category = "order_status"
	CategoryStockInfo   Category = "stock_info"
	CategoryTracking    Category = "tracking"
	CategoryCustom      Category = "custom"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryPurchase, CategoryOrderStatus, CategoryStockInfo, CategoryTracking, CategoryCustom:
		return true
	default:
		return false
	}
}

// MessageTemplate is one piece of merchant-authored message wording, sent
// over one or more channels. "All channels" (per the product ask) is just
// Channels containing every messaging.Channel value -- no separate "all"
// sentinel, so a channel added to messaging.Channel later doesn't silently
// fall outside an existing "all" template until re-saved.
//
// Body may contain {{token}} placeholders -- rendering (resolving tokens
// against a specific event's data) is an application-layer concern, not
// owned by this struct, since the available tokens depend on which event
// triggered the send.
type MessageTemplate struct {
	ID       string
	TenantID string
	Name     string
	Category Category
	Channels []messaging.Channel
	// Subject is only used when messaging.ChannelEmail is one of Channels --
	// harmless unused value otherwise, same convention as
	// SalesChannel.EnrollmentTag being ignored outside EnrollmentModeTag.
	Subject   string
	Body      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
