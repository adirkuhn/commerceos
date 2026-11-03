// Package deadletter is an event a consumer gave up on after its last
// redelivery, parked on a dead-letter stream until a super-admin retries
// it for the consumer that failed or discards it.
package deadletter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrForbidden     = errs.New("DEAD_LETTER_FORBIDDEN", "only a super-admin can manage dead letters")
	ErrNotFound      = errs.New("DEAD_LETTER_NOT_FOUND", "dead letter not found")
	ErrInvalidStream = errs.New("DEAD_LETTER_INVALID_STREAM", "dead letter stream must be events or webhooks")
	// ErrRetryFailed: the event could not be published again; the dead
	// letter is still there.
	ErrRetryFailed = errs.NewWithCause("DEAD_LETTER_RETRY_FAILED", "dead letter could not be retried")
)

// Stream is which dead-letter stream a dead letter sits on: domain events
// or inbound webhooks.
type Stream string

const (
	StreamEvents   Stream = "events"
	StreamWebhooks Stream = "webhooks"
)

// Streams lists every dead-letter stream.
func Streams() []Stream { return []Stream{StreamEvents, StreamWebhooks} }

func ParseStream(s string) (Stream, error) {
	for _, known := range Streams() {
		if Stream(s) == known {
			return known, nil
		}
	}
	return "", ErrInvalidStream
}

// DeadLetter is one dead-lettered event. Consumer, Error, NumDelivered and
// DeadLetteredAt are empty for messages dead-lettered before failure
// details were recorded.
type DeadLetter struct {
	Stream   Stream
	Sequence uint64
	// Subject is the event's original subject, where a retry publishes it.
	Subject        string
	Consumer       string
	EventID        string
	TenantID       string
	Error          string
	NumDelivered   int
	DeadLetteredAt *time.Time
	// StoredAt is when the dead-letter stream received it.
	StoredAt time.Time
	Data     []byte
}

// ConsumerKnown: false means a retry reaches every consumer on the
// subject (each skips it if it already processed the event).
func (d DeadLetter) ConsumerKnown() bool { return d.Consumer != "" }

// Identify reads the event and tenant IDs from a dead letter's data: a
// domain event (event_id, tenant_id) or a webhook envelope (EventID,
// TenantID). Data that is not a JSON object yields empty IDs.
func Identify(data []byte) (eventID, tenantID string) {
	var ids struct {
		EventID          string `json:"event_id"`
		TenantID         string `json:"tenant_id"`
		EnvelopeEventID  string `json:"EventID"`
		EnvelopeTenantID string `json:"TenantID"`
	}
	if json.Unmarshal(data, &ids) != nil {
		return "", ""
	}
	return firstNonEmpty(ids.EventID, ids.EnvelopeEventID), firstNonEmpty(ids.TenantID, ids.EnvelopeTenantID)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Filter narrows a listing of one stream. After is the cursor: only dead
// letters with a higher sequence are listed.
type Filter struct {
	Stream   Stream
	Subject  string
	Consumer string
	TenantID string
	After    uint64
	Limit    int
}

func (f Filter) Matches(d DeadLetter) bool {
	return (f.Subject == "" || f.Subject == d.Subject) &&
		(f.Consumer == "" || f.Consumer == d.Consumer) &&
		(f.TenantID == "" || f.TenantID == d.TenantID)
}

func (f Filter) LimitOrDefault() int {
	switch {
	case f.Limit <= 0:
		return DefaultLimit
	case f.Limit > MaxLimit:
		return MaxLimit
	default:
		return f.Limit
	}
}

// Page is one page of a listing. NextCursor is the Filter.After for the
// next page; 0 when this is the last page. Total is every dead letter on
// the stream, filtered or not.
type Page struct {
	Items      []DeadLetter
	NextCursor uint64
	Total      uint64
}

// Store is where dead letters are kept. Reading never removes one.
type Store interface {
	List(ctx context.Context, f Filter) (Page, error)
	Get(ctx context.Context, stream Stream, seq uint64) (DeadLetter, error)
	Delete(ctx context.Context, stream Stream, seq uint64) error
}

// RetryPublisher publishes a dead letter's data again on its original
// subject, aimed at its consumer when known.
type RetryPublisher interface {
	Republish(ctx context.Context, d DeadLetter) error
}
