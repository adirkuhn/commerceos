package events

import "time"

// DomainEvent is the base contract every domain event must satisfy.
type DomainEvent interface {
	EventName() string
	GetEventID() string
	GetOccurredAt() time.Time
}

// Base holds common fields. Embed in every concrete event.
type Base struct {
	EventID    string    `json:"event_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (b Base) GetEventID() string       { return b.EventID }
func (b Base) GetOccurredAt() time.Time { return b.OccurredAt }
