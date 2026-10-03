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

// listingsForUnlist is what deleting a variant or SKU needs from the
// channel listings (ISP): the product's listings to snapshot, and removing
// a listing row whose variant stays behind with nothing to sell.
type listingsForUnlist interface {
	ListByProduct(ctx context.Context, productID string) ([]channelsDomain.ProductChannelListing, error)
	Delete(ctx context.Context, id string) error
}

// enqueueVariantUnlisted writes a pms.variant.unlisted outbox row in the
// caller's transaction -- nothing is written when the variant was never
// listed anywhere.
func enqueueVariantUnlisted(ctx context.Context, outboxRepo outbox.Repository, productID, variantID string, refs []channelsDomain.VariantListingRef) error {
	if len(refs) == 0 {
		return nil
	}
	listings := make([]events.VariantUnlistedListing, len(refs))
	for i, r := range refs {
		listings[i] = events.VariantUnlistedListing{
			ConnectionID: r.ConnectionID, Provider: r.Provider, ExternalID: r.ExternalID, ParentExternalID: r.ParentExternalID,
		}
	}
	event := events.VariantUnlisted{
		Base:      events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID:  tenant.IDFromContext(ctx),
		ProductID: productID, VariantID: variantID, Listings: listings,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
