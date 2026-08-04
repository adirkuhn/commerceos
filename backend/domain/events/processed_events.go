package events

import "context"

// ProcessedEventRepository backs consumer idempotency: a durable record of
// "this consumer already ran for this EventID". JetStream is at-least-once
// delivery, so every consumer must expect (and safely ignore) redelivery of
// the same message.
//
// IsProcessed must be checked *before* doing any side-effecting work, and
// MarkProcessed called only *after* that work succeeds — marking first
// would make a message that fails partway through look "already handled"
// on retry, permanently losing it instead of letting JetStream redeliver.
type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID, consumer string) (bool, error)
	MarkProcessed(ctx context.Context, eventID, consumer string) error
}
