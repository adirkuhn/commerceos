// Package whatsapp holds the WhatsApp integration's own domain state: the
// per-phone-number conversation a chat-based checkout is carried across
// turns in. Everything else about the integration (product lookup, status
// notifications) is stateless and needed no domain package of its own (see
// application/whatsapp) -- this one exists because order placement is a
// genuine multi-turn flow.
package whatsapp

import "time"

// ConversationState is where one phone number's in-progress checkout
// currently is. The zero value (ConversationStateIdle) is the default/rest
// state -- no cart, no pending flow -- so a Conversation row's State column
// defaults to it without any special-casing.
type ConversationState string

const (
	// ConversationStateIdle is the rest state: no active checkout. Inbound
	// text is parsed as a command (STOCK/PRICE/.../ORDER/CART/CHECKOUT).
	ConversationStateIdle ConversationState = ""
	// ConversationStateAwaitingNewCustomerName/Email/Address are the
	// new-customer capture steps, entered only when CHECKOUT finds no
	// customer resolved for this phone number -- see
	// application/whatsapp.OrderFlowUseCase. Skipped entirely for an
	// already-known customer (matched by PhoneE164), who goes straight to
	// AwaitingOrderConfirmation using whatever's already on file.
	ConversationStateAwaitingNewCustomerName    ConversationState = "awaiting_new_customer_name"
	ConversationStateAwaitingNewCustomerEmail   ConversationState = "awaiting_new_customer_email"
	ConversationStateAwaitingNewCustomerAddress ConversationState = "awaiting_new_customer_address"
	// ConversationStateAwaitingOrderConfirmation is the final step for both
	// paths above -- inbound text here is parsed as YES/NO, not a command
	// or a capture answer.
	ConversationStateAwaitingOrderConfirmation ConversationState = "awaiting_order_confirmation"
)

// CartItem is one line of an in-progress WhatsApp order -- PriceMinorUnits
// is snapshotted at ORDER-command time (the price the customer was quoted),
// not re-read at confirmation, so a price change mid-conversation can never
// silently change what they agreed to buy.
type CartItem struct {
	SKUID           string
	SKUCode         string
	ProductName     string
	Quantity        int64
	PriceMinorUnits int64
	CurrencyCode    string
}

// Conversation is one phone number's in-progress interaction with the
// WhatsApp integration -- at most one per (TenantID, PhoneE164), upserted
// on every inbound message (see ConversationRepository.Upsert).
type Conversation struct {
	ID        string
	TenantID  string
	PhoneE164 string
	State     ConversationState
	Cart      []CartItem
	// PendingCustomerName/Email/Address hold the new-customer capture
	// flow's answers as they come in, one per turn -- only meaningful
	// while State is one of the AwaitingNewCustomer* states or
	// AwaitingOrderConfirmation (reached via that path). Cleared (reset to
	// "") once the order is placed or the flow is cancelled.
	PendingCustomerName    string
	PendingCustomerEmail   string
	PendingCustomerAddress string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
