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

type stubRepricer struct {
	calls []string
	err   error
}

func (s *stubRepricer) Reprice(_ context.Context, poolSKUID string) error {
	s.calls = append(s.calls, poolSKUID)
	return s.err
}

func newPerKiloMsg(t *testing.T, id string) *nats.Msg {
	t.Helper()
	event := events.SKUPricePerKiloChanged{Base: events.Base{EventID: id, OccurredAt: time.Now()}, TenantID: "tenant-1", SKUID: "sku-pool", NewPricePerKiloMinorUnits: 3299}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestSKUPricePerKiloChangedConsumer_repricesThePoolsPackagesAndMarksProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	repricer := &stubRepricer{}
	c := consumers.NewSKUPricePerKiloChangedConsumer(processed, repricer, &stubDLQPublisher{})

	c.Handle(newPerKiloMsg(t, "evt-1"))

	require.Equal(t, []string{"sku-pool"}, repricer.calls)
	require.Equal(t, []string{"evt-1|weight-package-repricer"}, processed.marked)
}

func TestSKUPricePerKiloChangedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|weight-package-repricer": true}}
	repricer := &stubRepricer{}
	c := consumers.NewSKUPricePerKiloChangedConsumer(processed, repricer, &stubDLQPublisher{})

	c.Handle(newPerKiloMsg(t, "evt-1"))

	require.Empty(t, repricer.calls)
}

func TestSKUPricePerKiloChangedConsumer_leavesAFailedRepriceForRedelivery(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	repricer := &stubRepricer{err: errors.New("db down")}
	c := consumers.NewSKUPricePerKiloChangedConsumer(processed, repricer, &stubDLQPublisher{})

	c.Handle(newPerKiloMsg(t, "evt-1"))

	require.Len(t, repricer.calls, 1)
	require.Empty(t, processed.marked)
}

func TestSKUPricePerKiloChangedConsumer_dropsMalformedPayload(t *testing.T) {
	repricer := &stubRepricer{}
	c := consumers.NewSKUPricePerKiloChangedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, repricer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "pms.sku.price_per_kilo_changed", Data: []byte("not json")})

	require.Empty(t, repricer.calls)
}
