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

type stubDeleter struct {
	calls [][]events.ProductDeletedListing
	err   error
}

func (s *stubDeleter) Execute(_ context.Context, listings []events.ProductDeletedListing) error {
	s.calls = append(s.calls, listings)
	return s.err
}

func newDeletedMsg(t *testing.T, event events.ProductDeleted) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestProductDeletedConsumer_deletesFromChannelsAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	deleter := &stubDeleter{}
	c := consumers.NewProductDeletedConsumer(processed, deleter, &stubDLQPublisher{})
	listings := []events.ProductDeletedListing{{ConnectionID: "conn-1", Provider: "woocommerce", ExternalID: "555"}}
	event := events.ProductDeleted{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", ProductID: "product-1", Listings: listings,
	}

	c.Handle(newDeletedMsg(t, event))

	require.Equal(t, [][]events.ProductDeletedListing{listings}, deleter.calls)
	require.Equal(t, []string{"evt-1|product-deleted-cleanup"}, processed.marked)
}

func TestProductDeletedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|product-deleted-cleanup": true}}
	deleter := &stubDeleter{}
	c := consumers.NewProductDeletedConsumer(processed, deleter, &stubDLQPublisher{})
	event := events.ProductDeleted{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newDeletedMsg(t, event))

	require.Empty(t, deleter.calls, "already-processed event must not re-run cleanup")
}

func TestProductDeletedConsumer_doesNotMarkProcessedWhenDeleteFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	deleter := &stubDeleter{err: errors.New("db down")}
	c := consumers.NewProductDeletedConsumer(processed, deleter, &stubDLQPublisher{})
	event := events.ProductDeleted{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newDeletedMsg(t, event))

	require.Len(t, deleter.calls, 1)
	require.Empty(t, processed.marked, "a failed cleanup must stay eligible for redelivery, not look already-handled")
}

func TestProductDeletedConsumer_dropsMalformedPayloadWithoutCallingDeleter(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	deleter := &stubDeleter{}
	c := consumers.NewProductDeletedConsumer(processed, deleter, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "pms.product.deleted", Data: []byte("not json")})

	require.Empty(t, deleter.calls)
}
