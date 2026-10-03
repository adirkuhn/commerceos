package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubVariantDeleter struct {
	calls [][]events.VariantUnlistedListing
	err   error
}

func (s *stubVariantDeleter) Execute(_ context.Context, listings []events.VariantUnlistedListing) error {
	s.calls = append(s.calls, listings)
	return s.err
}

func newUnlistedMsg(t *testing.T, event events.VariantUnlisted) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func unlisted(id string) events.VariantUnlisted {
	return events.VariantUnlisted{
		Base: events.Base{EventID: id, OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1", VariantID: "variant-222",
		Listings: []events.VariantUnlistedListing{{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901"}},
	}
}

func TestVariantUnlistedConsumer_takesTheVariantOffChannelsAndMarksProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	deleter := &stubVariantDeleter{}
	c := consumers.NewVariantUnlistedConsumer(processed, deleter, &stubDLQPublisher{})

	c.Handle(newUnlistedMsg(t, unlisted("evt-1")))

	require.Equal(t, [][]events.VariantUnlistedListing{unlisted("evt-1").Listings}, deleter.calls)
	require.Equal(t, []string{"evt-1|variant-unlisted-cleanup"}, processed.marked)
}

func TestVariantUnlistedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|variant-unlisted-cleanup": true}}
	deleter := &stubVariantDeleter{}
	c := consumers.NewVariantUnlistedConsumer(processed, deleter, &stubDLQPublisher{})

	c.Handle(newUnlistedMsg(t, unlisted("evt-1")))

	require.Empty(t, deleter.calls)
}

func TestVariantUnlistedConsumer_doesNotMarkProcessedWhenCleanupFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	deleter := &stubVariantDeleter{err: errors.New("db down")}
	c := consumers.NewVariantUnlistedConsumer(processed, deleter, &stubDLQPublisher{})

	c.Handle(newUnlistedMsg(t, unlisted("evt-1")))

	require.Len(t, deleter.calls, 1)
	require.Empty(t, processed.marked)
}

func TestVariantUnlistedConsumer_dropsMalformedPayload(t *testing.T) {
	deleter := &stubVariantDeleter{}
	c := consumers.NewVariantUnlistedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, deleter, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "pms.variant.unlisted", Data: []byte("not json")})

	require.Empty(t, deleter.calls)
}
