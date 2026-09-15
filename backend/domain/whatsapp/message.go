package whatsapp

import (
	"context"
	"time"
)

// MessageDirection is which way a Message travelled.
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// Message is one append-only entry in a phone number's WhatsApp message
// log -- distinct from Conversation (same-row-overwritten checkout state).
// CustomerID is nil unless the phone number matched an existing customer
// at the time this message was recorded (see
// customers.CustomerRepository.FindByPhoneE164) -- a message from an
// unmatched number is still logged, just unlinked. Same
// append-only-log-tied-to-a-parent shape as oms.SalesOrderActivity.
type Message struct {
	ID         string
	TenantID   string
	PhoneE164  string
	CustomerID *string
	Direction  MessageDirection
	Body       string
	// WebhookEventID is set only for an inbound message recorded from a
	// webhook envelope (see application/whatsapp.HandleInboundMessageUseCase)
	// -- nil for an outbound reply/notification, which has no envelope of
	// its own. Lets Create dedupe a redelivered envelope instead of
	// re-logging the same customer message on every NATS retry (see
	// WhatsAppMessageRepo.Create's doc comment).
	WebhookEventID *string
	CreatedAt      time.Time
}

// ConversationSummary is one row of the WhatsApp conversations list (the
// "WhatsApp Web"-style admin view) -- one per phone number that has ever
// exchanged a message, carrying only its most recent message so the list
// can render without pulling every message for every conversation.
// CustomerName is "" when CustomerID is nil (never matched) or the
// customer record itself has no name on file.
type ConversationSummary struct {
	PhoneE164     string
	CustomerID    *string
	CustomerName  string
	LastDirection MessageDirection
	LastBody      string
	LastMessageAt time.Time
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	// ListByPhone returns every message for this phone number, newest
	// first -- works even when it never matched a customer.
	ListByPhone(ctx context.Context, phoneE164 string) ([]Message, error)
	// ListByCustomer returns every message linked to this customer,
	// newest first -- the Customer detail page's conversation panel.
	ListByCustomer(ctx context.Context, customerID string) ([]Message, error)
	// ListConversations returns one summary per phone number that has
	// exchanged at least one message, newest conversation first -- the
	// conversations list view.
	ListConversations(ctx context.Context) ([]ConversationSummary, error)
	// DeleteByPhone permanently removes every message for this phone
	// number -- the conversations list's "Delete chat" action. A genuine
	// hard delete (not a soft/audit-kept status change like
	// webhooks.FallbackEvent's Abandon): this log is a tenant-owned
	// convenience history with no compliance/audit requirement, matching
	// the "Delete chat" metaphor real WhatsApp clients use. Idempotent --
	// deleting an already-empty or never-existed phone number is not an
	// error.
	DeleteByPhone(ctx context.Context, phoneE164 string) error
}
