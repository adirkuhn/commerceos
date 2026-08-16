package outbox

import (
	"context"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
)

type Message struct {
	ID          uint
	EventID     string
	Subject     string
	Payload     []byte
	Status      Status
	OccurredAt  time.Time
	PublishedAt *time.Time
	CreatedAt   time.Time
	// RetryCount counts publish attempts that failed for a reason other
	// than NATS/JetStream connectivity (see cmd/relay's isConnectivityErr)
	// -- a relay unable to reach NATS at all never increments this, so an
	// outage of any length never strands a row in Failed on its own.
	RetryCount int
}

type Repository interface {
	// Enqueue must be called inside tenant.WithTenant — it extracts the tx from ctx.
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
	FetchPending(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, id uint) error
	MarkFailed(ctx context.Context, id uint) error
}
