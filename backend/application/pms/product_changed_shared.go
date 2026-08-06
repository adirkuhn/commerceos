package pms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// enqueueProductChanged writes a pms.product.changed outbox row in the
// caller's current transaction — must be called from inside the same
// tenant.WithTenant block as the write that changed the product, so the
// event and the write commit atomically. Consumers (see
// infrastructure/consumers) use this to re-push the product to every
// channel it's already linked to.
func enqueueProductChanged(ctx context.Context, outboxRepo outbox.Repository, productID string) error {
	event := events.ProductChanged{
		Base:      events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:  tenant.IDFromContext(ctx),
		ProductID: productID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}

// enqueueProductDeleted writes a pms.product.deleted outbox row in the
// caller's current transaction, carrying a snapshot of the channel listings
// that existed immediately before the product row (and its cascaded
// listings, see migrations/007_channels.sql/037_sales_channels.sql) was
// deleted -- by the time a consumer processes this asynchronously, the
// listing rows are gone, so the event is the only place left to learn what
// needs cleaning up on each external channel.
func enqueueProductDeleted(ctx context.Context, outboxRepo outbox.Repository, productID string, listings []channelsDomain.ProductChannelListing) error {
	refs := make([]events.ProductDeletedListing, len(listings))
	for i, l := range listings {
		refs[i] = events.ProductDeletedListing{ConnectionID: l.ConnectionID, Provider: l.Provider, ExternalID: l.ExternalID}
	}
	event := events.ProductDeleted{
		Base:      events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:  tenant.IDFromContext(ctx),
		ProductID: productID,
		Listings:  refs,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
