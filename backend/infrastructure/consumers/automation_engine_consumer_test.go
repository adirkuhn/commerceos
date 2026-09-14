package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubAutomationEvaluator struct {
	calls []map[string]any
	err   error
}

func (s *stubAutomationEvaluator) Execute(_ context.Context, _, _ string, payload map[string]any) error {
	s.calls = append(s.calls, payload)
	return s.err
}

func newAutomationEventMsg(t *testing.T, subject string, payload map[string]any) *nats.Msg {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return &nats.Msg{Subject: subject, Data: data}
}

func TestAutomationEngineConsumer_evaluatesAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	evaluator := &stubAutomationEvaluator{}
	c := consumers.NewAutomationEngineConsumer("shipping.shipment.fulfillment_signal", processed, evaluator, &stubDLQPublisher{})
	payload := map[string]any{"event_id": "evt-1", "tenant_id": "tenant-1", "sales_order_id": "so-1", "status_code": float64(8)}

	c.Handle(newAutomationEventMsg(t, "shipping.shipment.fulfillment_signal", payload))

	require.Len(t, evaluator.calls, 1)
	require.Equal(t, "so-1", evaluator.calls[0]["sales_order_id"])
	require.Equal(t, []string{"evt-1|automation-engine-shipping.shipment.fulfillment_signal"}, processed.marked)
}

func TestAutomationEngineConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|automation-engine-shipping.shipment.fulfillment_signal": true}}
	evaluator := &stubAutomationEvaluator{}
	c := consumers.NewAutomationEngineConsumer("shipping.shipment.fulfillment_signal", processed, evaluator, &stubDLQPublisher{})
	payload := map[string]any{"event_id": "evt-1", "tenant_id": "tenant-1"}

	c.Handle(newAutomationEventMsg(t, "shipping.shipment.fulfillment_signal", payload))

	require.Empty(t, evaluator.calls, "already-processed event must not re-evaluate")
}

func TestAutomationEngineConsumer_doesNotMarkProcessedWhenEvaluateFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	evaluator := &stubAutomationEvaluator{err: errors.New("send failed")}
	c := consumers.NewAutomationEngineConsumer("shipping.shipment.fulfillment_signal", processed, evaluator, &stubDLQPublisher{})
	payload := map[string]any{"event_id": "evt-1", "tenant_id": "tenant-1"}

	c.Handle(newAutomationEventMsg(t, "shipping.shipment.fulfillment_signal", payload))

	require.Len(t, evaluator.calls, 1)
	require.Empty(t, processed.marked, "a failed evaluation must stay eligible for redelivery, not look already-handled")
}

func TestAutomationEngineConsumer_dropsMalformedPayloadWithoutCallingEvaluator(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	evaluator := &stubAutomationEvaluator{}
	c := consumers.NewAutomationEngineConsumer("shipping.shipment.fulfillment_signal", processed, evaluator, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "shipping.shipment.fulfillment_signal", Data: []byte("not json")})

	require.Empty(t, evaluator.calls)
}

func TestAutomationEngineConsumer_dropsPayloadMissingTenantID(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	evaluator := &stubAutomationEvaluator{}
	c := consumers.NewAutomationEngineConsumer("shipping.shipment.fulfillment_signal", processed, evaluator, &stubDLQPublisher{})
	payload := map[string]any{"event_id": "evt-1"}

	c.Handle(newAutomationEventMsg(t, "shipping.shipment.fulfillment_signal", payload))

	require.Empty(t, evaluator.calls)
}

// TestAutomationEngineConsumer_independentPerSubject proves two instances
// for different subjects (same shared Go type, per this consumer's own doc
// comment) use distinct processed-event keys -- one being marked done must
// never make the other skip.
func TestAutomationEngineConsumer_independentPerSubject(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|automation-engine-pms.product.deleted": true}}
	evaluator := &stubAutomationEvaluator{}
	c := consumers.NewAutomationEngineConsumer("oms.sales_order.status_changed", processed, evaluator, &stubDLQPublisher{})
	payload := map[string]any{"event_id": "evt-1", "tenant_id": "tenant-1"}

	c.Handle(newAutomationEventMsg(t, "oms.sales_order.status_changed", payload))

	require.Len(t, evaluator.calls, 1, "a different subject's processed marker must not suppress this one")
}
