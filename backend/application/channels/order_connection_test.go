package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func strp(s string) *string { return &s }

func TestOrderConnection_followsTheSalesChannelToItsCurrentConnection(t *testing.T) {
	channels := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "ch-woo", ConnectionID: strp("conn-now")}}
	order := omsDomain.SalesOrder{ID: "so-1", SalesChannelID: strp("ch-woo"), ConnectionID: strp("conn-at-import")}

	id, err := appchannels.NewOrderConnectionResolver(channels).ConnectionIDFor(context.Background(), order)

	require.NoError(t, err)
	assert.Equal(t, "conn-now", id, "the channel decides, not what the order recorded at import")
}

// Until every order carries its sales channel (COM-15 backfill), the
// connection recorded on the order still works.
func TestOrderConnection_fallsBackToTheConnectionRecordedOnTheOrder(t *testing.T) {
	order := omsDomain.SalesOrder{ID: "so-1", ConnectionID: strp("conn-1")}

	id, err := appchannels.NewOrderConnectionResolver(&stubSalesChannelRepo{}).ConnectionIDFor(context.Background(), order)

	require.NoError(t, err)
	assert.Equal(t, "conn-1", id)
}

// The 2026-11-03 incident: an order that resolves to nothing used to be
// skipped silently (acked, never retried, never dead-lettered).
func TestOrderConnection_nothingToResolveIsATypedError(t *testing.T) {
	for name, order := range map[string]omsDomain.SalesOrder{
		"no channel, no connection":    {ID: "so-1"},
		"channel without a connection": {ID: "so-1", SalesChannelID: strp("ch-1")},
	} {
		t.Run(name, func(t *testing.T) {
			channels := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "ch-1"}}

			_, err := appchannels.NewOrderConnectionResolver(channels).ConnectionIDFor(context.Background(), order)

			require.ErrorIs(t, err, channelsDomain.ErrOrderChannelUnresolved)
		})
	}
}

func TestOrderConnection_aChannelLookupFailureIsReturned(t *testing.T) {
	boom := errors.New("db down")
	order := omsDomain.SalesOrder{ID: "so-1", SalesChannelID: strp("ch-1"), ConnectionID: strp("conn-1")}

	_, err := appchannels.NewOrderConnectionResolver(&stubSalesChannelRepo{findErr: boom}).ConnectionIDFor(context.Background(), order)

	require.ErrorIs(t, err, boom)
}
