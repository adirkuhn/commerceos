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

type stubShipmentFulfillmentSignalApplier struct {
	calls []stubShipmentFulfillmentSignalApplyCall
	err   error
}

type stubShipmentFulfillmentSignalApplyCall struct {
	salesOrderID, kind, statusMain string
	statusCode                     int
}

func (s *stubShipmentFulfillmentSignalApplier) Execute(_ context.Context, salesOrderID, kind, statusMain string, statusCode int) error {
	s.calls = append(s.calls, stubShipmentFulfillmentSignalApplyCall{salesOrderID: salesOrderID, kind: kind, statusMain: statusMain, statusCode: statusCode})
	return s.err
}

func newShipmentFulfillmentSignalMsg(t *testing.T, event events.ShipmentFulfillmentSignal) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestShipmentFulfillmentSignalConsumer_appliesAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	applier := &stubShipmentFulfillmentSignalApplier{}
	c := consumers.NewShipmentFulfillmentSignalConsumer(processed, applier, &stubDLQPublisher{})
	event := events.ShipmentFulfillmentSignal{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", SalesOrderID: "order-1",
		Kind: events.ShipmentFulfillmentSignalKindShipmentStatus, StatusMain: "handed_to_carrier", StatusCode: 3,
	}

	c.Handle(newShipmentFulfillmentSignalMsg(t, event))

	require.Len(t, applier.calls, 1)
	require.Equal(t, "order-1", applier.calls[0].salesOrderID)
	require.Equal(t, events.ShipmentFulfillmentSignalKindShipmentStatus, applier.calls[0].kind)
	require.Equal(t, "handed_to_carrier", applier.calls[0].statusMain)
	require.Equal(t, 3, applier.calls[0].statusCode)
	require.Equal(t, []string{"evt-1|shipment-fulfillment-signal-apply"}, processed.marked)
}

func TestShipmentFulfillmentSignalConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|shipment-fulfillment-signal-apply": true}}
	applier := &stubShipmentFulfillmentSignalApplier{}
	c := consumers.NewShipmentFulfillmentSignalConsumer(processed, applier, &stubDLQPublisher{})
	event := events.ShipmentFulfillmentSignal{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "order-1"}

	c.Handle(newShipmentFulfillmentSignalMsg(t, event))

	require.Empty(t, applier.calls, "already-processed event must not re-run the apply")
}

func TestShipmentFulfillmentSignalConsumer_doesNotMarkProcessedWhenApplyFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	applier := &stubShipmentFulfillmentSignalApplier{err: errors.New("db down")}
	c := consumers.NewShipmentFulfillmentSignalConsumer(processed, applier, &stubDLQPublisher{})
	event := events.ShipmentFulfillmentSignal{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SalesOrderID: "order-1"}

	c.Handle(newShipmentFulfillmentSignalMsg(t, event))

	require.Len(t, applier.calls, 1)
	require.Empty(t, processed.marked, "a failed apply must stay eligible for redelivery, not look already-handled")
}

func TestShipmentFulfillmentSignalConsumer_dropsMalformedPayloadWithoutCallingApplier(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	applier := &stubShipmentFulfillmentSignalApplier{}
	c := consumers.NewShipmentFulfillmentSignalConsumer(processed, applier, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "shipping.shipment.fulfillment_signal", Data: []byte("not json")})

	require.Empty(t, applier.calls)
}
