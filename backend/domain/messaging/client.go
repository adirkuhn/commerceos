// Package messaging holds the shared port for sending an outbound message
// on any channel (email, SMS, WhatsApp, ...) -- a generic capability, not
// scoped to one channel or one third-party provider. This supersedes the
// former domain/email package: every channel's message is the same shape,
// so a caller that just wants to notify a customer depends on one
// interface regardless of which channel ends up carrying it.
package messaging

import "context"

// Channel identifies which transport a Message travels over. Concrete
// Client implementations validate the Channel matches what they carry
// (e.g. the SMTP client rejects anything but ChannelEmail) so a message
// built for the wrong channel fails loudly instead of being silently
// misdelivered.
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// Message is a single outbound message. Subject/HTMLBody are email-only;
// Body is the plain-text payload SMS/WhatsApp implementations send.
// Deliberately plain (no templates, no attachments) -- every current
// caller only ever sends a short message.
type Message struct {
	Channel  Channel
	To       string
	Subject  string
	HTMLBody string
	Body     string
}

// Client sends one Message. Implementations must fail loudly on a
// non-2xx/non-accepted response from the underlying transport rather than
// silently dropping it -- there is no outbox/retry for messaging today, so
// a swallowed error here means the recipient never finds out.
type Client interface {
	Send(ctx context.Context, msg Message) error
}

// EmailClientFactory builds a Client scoped to one tenant's own connected
// email integration (see domain/integrations.ProviderEmail) -- same DI
// shape as integrations.EvolutionAPIClientFactory: the use case that needs
// a per-tenant email client (application/messagecenter's automation
// engine) never constructs a concrete smtp.Client itself. Unlike
// EvolutionAPIClientFactory there's no per-provider dispatch here (only
// one Client implementation exists, infrastructure/messaging/smtp) --
// still a factory, not a direct constructor call, so this package stays
// free of an infrastructure import.
type EmailClientFactory interface {
	NewClient(host, port, username, password, from string) Client
}
