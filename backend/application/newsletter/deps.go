// Package newsletter holds the mailing-list use cases: double opt-in
// signup from the storefront, confirm/unsubscribe links, and the admin
// list, CSV export and erasure.
package newsletter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/idgen"
)

// mailer sends as the current tenant (its own email connection, or the
// platform's).
type mailer interface {
	Send(ctx context.Context, msg messaging.Message) error
}

type storeNamer interface {
	StoreName(ctx context.Context, salesChannelID string) (string, error)
}

type outboxEnqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

type Deps struct {
	DB       *gorm.DB
	Repo     newsletter.Repository
	Mailer   mailer
	Stores   storeNamer
	Outbox   outboxEnqueuer
	Now      func() time.Time
	NewToken func() (string, error)
	// ConfirmURL / UnsubscribeURL build the storefront links sent by email.
	ConfirmURL     func(salesChannelID, token string) string
	UnsubscribeURL func(salesChannelID, token string) string
}

func (d Deps) enqueue(ctx context.Context, ev events.DomainEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", ev.EventName(), err)
	}
	return d.Outbox.Enqueue(ctx, ev.EventName(), ev.GetEventID(), payload, ev.GetOccurredAt())
}

func (d Deps) base() events.Base { return events.Base{EventID: idgen.NewV4(), OccurredAt: d.Now()} }
