// Package email holds the port for sending transactional email -- a
// generic capability, not scoped to one third-party provider. See
// infrastructure/email for the SMTP implementation (the only one built so
// far); the interface is shaped so a future REST-API provider (Resend,
// SendGrid, etc.) implements the same Client without callers changing.
package email

import "context"

// Message is a single transactional email -- deliberately plain (no
// templates, no attachments) since every current caller (magic-link login)
// only ever sends a short HTML body.
type Message struct {
	To       string
	Subject  string
	HTMLBody string
}

// Client sends one Message. Implementations must fail loudly on a
// non-2xx/non-accepted response from the underlying transport rather than
// silently dropping mail -- there is no outbox/retry for email today, so a
// swallowed error here means the recipient never finds out.
type Client interface {
	Send(ctx context.Context, msg Message) error
}
