package consumers_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

func newStockMsg(t *testing.T, event events.InventoryStockChanged) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestInventoryStockChangedConsumer_syncsAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{}
	c := consumers.NewInventoryStockChangedConsumer(processed, syncer, &stubDLQPublisher{})
	event := events.InventoryStockChanged{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", SKUID: "sku-1",
	}

	c.Handle(newStockMsg(t, event))

	require.Equal(t, []string{"sku-1"}, syncer.calls)
	require.Equal(t, []string{"evt-1|inventory-stock-changed-sync"}, processed.marked)
}

func TestInventoryStockChangedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|inventory-stock-changed-sync": true}}
	syncer := &stubSyncer{}
	c := consumers.NewInventoryStockChangedConsumer(processed, syncer, &stubDLQPublisher{})
	event := events.InventoryStockChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SKUID: "sku-1"}

	c.Handle(newStockMsg(t, event))

	require.Empty(t, syncer.calls, "already-processed event must not re-run the sync")
}

func TestInventoryStockChangedConsumer_doesNotMarkProcessedWhenSyncFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{err: errors.New("db down")}
	c := consumers.NewInventoryStockChangedConsumer(processed, syncer, &stubDLQPublisher{})
	event := events.InventoryStockChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SKUID: "sku-1"}

	c.Handle(newStockMsg(t, event))

	require.Len(t, syncer.calls, 1)
	require.Empty(t, processed.marked, "a failed sync must stay eligible for redelivery, not look already-handled")
}

func TestInventoryStockChangedConsumer_dropsMalformedPayloadWithoutCallingSyncer(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{}
	c := consumers.NewInventoryStockChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "inventory.stock.changed", Data: []byte("not json")})

	require.Empty(t, syncer.calls)
}
