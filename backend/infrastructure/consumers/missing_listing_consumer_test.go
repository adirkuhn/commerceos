package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubMissingListingHandler struct {
	calls []appchannels.HandleMissingListingInput
	err   error
}

func (s *stubMissingListingHandler) Execute(_ context.Context, in appchannels.HandleMissingListingInput) error {
	s.calls = append(s.calls, in)
	return s.err
}

func listingMissingMsg(t *testing.T, data map[string]any) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(events.EntityAction{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", EntityID: "listing-1", Data: data})
	require.NoError(t, err)
	return &nats.Msg{Subject: events.ProductListingOrphaned.Subject, Data: payload}
}

func TestMissingListingConsumer_handlesTheProductOnItsConnection(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	handler := &stubMissingListingHandler{}
	c := consumers.NewMissingListingConsumer(processed, handler, &stubDLQPublisher{})

	c.Handle(listingMissingMsg(t, map[string]any{"product_id": "product-1", "connection_id": "conn-1"}))

	assert.Equal(t, []appchannels.HandleMissingListingInput{{ProductID: "product-1", ConnectionID: "conn-1"}}, handler.calls)
	assert.Equal(t, []string{"evt-1|" + consumers.MissingListingConsumerName}, processed.marked)
}

func TestMissingListingConsumer_failureIsNotMarkedProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	c := consumers.NewMissingListingConsumer(processed, &stubMissingListingHandler{err: errors.New("woo down")}, &stubDLQPublisher{})

	c.Handle(listingMissingMsg(t, map[string]any{"product_id": "product-1", "connection_id": "conn-1"}))

	assert.Empty(t, processed.marked)
}

func TestMissingListingConsumer_dropsAPayloadWithoutProductOrConnection(t *testing.T) {
	handler := &stubMissingListingHandler{}
	c := consumers.NewMissingListingConsumer(&stubProcessedEvents{processed: map[string]bool{}}, handler, &stubDLQPublisher{})

	c.Handle(listingMissingMsg(t, map[string]any{"product_id": "product-1"}))

	assert.Empty(t, handler.calls)
}
