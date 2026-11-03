package channels

import (
	"errors"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// applyPushOutcome records a push's result on listing: synced, error, or --
// when the channel no longer has the product (integrationsDomain.
// ErrWooProductNotFound) -- missing. newlyMissing is true only on the push
// that discovered it, so channels.product_listing.orphaned is raised once.
func applyPushOutcome(listing *channelsDomain.ProductChannelListing, syncErr error, now time.Time) (newlyMissing bool) {
	listing.LastSyncedAt = &now
	listing.UpdatedAt = now
	switch {
	case syncErr == nil:
		listing.SyncStatus = channelsDomain.SyncStatusSynced
		listing.LastSyncError = ""
	case errors.Is(syncErr, integrationsDomain.ErrWooProductNotFound):
		newlyMissing = listing.SyncStatus != channelsDomain.SyncStatusMissing
		listing.SyncStatus = channelsDomain.SyncStatusMissing
		listing.LastSyncError = syncErr.Error()
	default:
		listing.SyncStatus = channelsDomain.SyncStatusError
		listing.LastSyncError = syncErr.Error()
	}
	return newlyMissing
}

// withoutMissingConnections drops every listing of a connection that has a
// missing listing: nothing is pushed there until the missing product is
// handled (HandleMissingListingUseCase) -- a variation's parent being gone
// makes the whole connection's set unpushable.
func withoutMissingConnections(listings []channelsDomain.ProductChannelListing) []channelsDomain.ProductChannelListing {
	missing := map[string]bool{}
	for _, l := range listings {
		if l.SyncStatus == channelsDomain.SyncStatusMissing {
			missing[l.ConnectionID] = true
		}
	}
	if len(missing) == 0 {
		return listings
	}
	out := make([]channelsDomain.ProductChannelListing, 0, len(listings))
	for _, l := range listings {
		if !missing[l.ConnectionID] {
			out = append(out, l)
		}
	}
	return out
}
