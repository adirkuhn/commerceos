package whatsapp

import "context"

// ConversationRepository is upsert-shaped per (TenantID, PhoneE164) -- at
// most one open conversation per phone number, so every turn overwrites the
// prior state rather than accumulating rows. Same convention as
// channels.SalesChannelProductOverrideRepository.
type ConversationRepository interface {
	// FindByPhone returns nil (not an error) when no conversation exists
	// yet for this phone number -- the normal case for its first-ever
	// message.
	FindByPhone(ctx context.Context, phoneE164 string) (*Conversation, error)
	Upsert(ctx context.Context, c *Conversation) error
}
