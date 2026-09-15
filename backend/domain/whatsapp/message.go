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
	CreatedAt  time.Time
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	// ListByPhone returns every message for this phone number, newest
	// first -- works even when it never matched a customer.
	ListByPhone(ctx context.Context, phoneE164 string) ([]Message, error)
	// ListByCustomer returns every message linked to this customer,
	// newest first -- the Customer detail page's conversation panel.
	ListByCustomer(ctx context.Context, customerID string) ([]Message, error)
}
