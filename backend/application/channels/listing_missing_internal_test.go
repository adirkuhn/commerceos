package channels

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestApplyPushOutcome(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	gone := fmt.Errorf("push: %w", integrationsDomain.ErrWooProductNotFound)

	l := channelsDomain.ProductChannelListing{SyncStatus: channelsDomain.SyncStatusSynced}
	assert.True(t, applyPushOutcome(&l, gone, now), "first discovery")
	assert.Equal(t, channelsDomain.SyncStatusMissing, l.SyncStatus)
	assert.Contains(t, l.LastSyncError, "no longer exists")
	assert.False(t, applyPushOutcome(&l, gone, now), "already known missing: raised once")

	assert.False(t, applyPushOutcome(&l, errors.New("timeout"), now))
	assert.Equal(t, channelsDomain.SyncStatusError, l.SyncStatus)

	assert.False(t, applyPushOutcome(&l, nil, now))
	assert.Equal(t, channelsDomain.SyncStatusSynced, l.SyncStatus)
	assert.Empty(t, l.LastSyncError)
	assert.Equal(t, now, *l.LastSyncedAt)
}

func TestWithoutMissingConnections(t *testing.T) {
	listings := []channelsDomain.ProductChannelListing{
		{ID: "parent", ConnectionID: "woo", SyncStatus: channelsDomain.SyncStatusMissing},
		{ID: "variation", ConnectionID: "woo", SyncStatus: channelsDomain.SyncStatusSynced},
		{ID: "bol", ConnectionID: "bol", SyncStatus: channelsDomain.SyncStatusSynced},
	}

	got := withoutMissingConnections(listings)

	assert.Equal(t, []channelsDomain.ProductChannelListing{listings[2]}, got)
	assert.Len(t, withoutMissingConnections(listings[1:]), 2, "nothing missing: untouched")
}
