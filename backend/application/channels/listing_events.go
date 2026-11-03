package channels

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
)

// How a listing came to exist -- data.source of channels.product_listing.created.
const (
	ListingSourcePush         = "push"
	ListingSourceVariantAdded = "variant_added"
	ListingSourceLinkExisting = "link_existing"
	ListingSourceImport       = "import"
)

// Why a listing was removed -- data.reason of channels.product_listing.deleted.
const (
	ListingDeletedReasonShapeMigration = "shape_migration"
	ListingDeletedReasonVariantRemoved = "variant_removed"
	// ListingDeletedReasonMissingInChannel: the channel's product was gone
	// and the product is being created there again (COM-18).
	ListingDeletedReasonMissingInChannel = "missing_in_channel"
)

func listingData(l channelsDomain.ProductChannelListing) eventrecord.Data {
	return eventrecord.Data{
		"product_id": l.ProductID, "variant_id": l.VariantID, "connection_id": l.ConnectionID,
		"provider": l.Provider, "external_id": l.ExternalID,
	}
}

func recordListingCreated(ctx context.Context, outbox eventrecord.Enqueuer, l channelsDomain.ProductChannelListing, source string) error {
	data := listingData(l)
	data["is_variation"] = l.IsVariation
	data["source"] = source
	return eventrecord.Record(ctx, outbox, events.ProductListingCreated, l.ID, data)
}

func recordListingUpdated(ctx context.Context, outbox eventrecord.Enqueuer, before, after channelsDomain.ProductChannelListing) error {
	return eventrecord.Record(ctx, outbox, events.ProductListingUpdated, after.ID,
		eventrecord.Data{"product_id": after.ProductID, "connection_id": after.ConnectionID, "provider": after.Provider},
		eventrecord.NewChanges().
			Set("variant_id", before.VariantID, after.VariantID).
			Set("external_id", before.ExternalID, after.ExternalID).
			Set("not_for_sale_note", before.NotForSaleNote, after.NotForSaleNote))
}

func recordListingDeleted(ctx context.Context, outbox eventrecord.Enqueuer, l channelsDomain.ProductChannelListing, reason string) error {
	data := listingData(l)
	data["reason"] = reason
	return eventrecord.Record(ctx, outbox, events.ProductListingDeleted, l.ID, data)
}

func recordListingGate(ctx context.Context, outbox eventrecord.Enqueuer, l channelsDomain.ProductChannelListing, gated bool) error {
	action := events.ProductListingUngated
	if gated {
		action = events.ProductListingGated
	}
	return eventrecord.Record(ctx, outbox, action, l.ID, eventrecord.Data{
		"product_id": l.ProductID, "connection_id": l.ConnectionID, "provider": l.Provider, "external_id": l.ExternalID,
	})
}

// syncLogChanges turns a sync log entry's field diff into event changes.
func syncLogChanges(diff map[string]channelsDomain.FieldChange) eventrecord.Changes {
	c := eventrecord.NewChanges()
	for field, ch := range diff {
		c = c.Set(field, ch.From, ch.To)
	}
	return c
}

func recordListingOrphaned(ctx context.Context, outbox eventrecord.Enqueuer, l channelsDomain.ProductChannelListing) error {
	return eventrecord.Record(ctx, outbox, events.ProductListingOrphaned, l.ID, listingData(l))
}
