// Package collections holds the collection use cases: the admin CRUD and
// the storefront reads (published collections and their products).
package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/idgen"
)

// MaxResolvedProducts caps how many products one collection page can page
// through on the storefront.
const MaxResolvedProducts = 2000

type outboxEnqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

type Deps struct {
	DB     *gorm.DB
	Repo   collections.Repository
	Outbox outboxEnqueuer
	Now    func() time.Time
}

func (d Deps) changed(ctx context.Context, c collections.Collection, deleted bool) error {
	ev := events.CollectionChanged{
		Base: events.Base{EventID: idgen.NewV4(), OccurredAt: d.Now()}, TenantID: c.TenantID, CollectionID: c.ID, Deleted: deleted,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", ev.EventName(), err)
	}
	return d.Outbox.Enqueue(ctx, ev.EventName(), ev.EventID, payload, ev.OccurredAt)
}

func pageWindow(page, perPage int) (int, int) {
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if page < 1 {
		page = 1
	}
	return perPage, (page - 1) * perPage
}
