package consumers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubClearanceSKUFinisher struct {
	calls   []string
	tenants []string
	err     error
}

func (s *stubClearanceSKUFinisher) ExecuteForSKU(ctx context.Context, skuID string) error {
	s.calls = append(s.calls, skuID)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func clearanceStockEvent() events.InventoryStockChanged {
	return events.InventoryStockChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", SKUID: "sku-1"}
}

func TestClearancePromotionStockConsumer_checksSKUUnderEventTenantAndMarksProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	finisher := &stubClearanceSKUFinisher{}
	c := consumers.NewClearancePromotionStockConsumer(processed, finisher, &stubDLQPublisher{})

	c.Handle(newStockMsg(t, clearanceStockEvent()))

	require.Equal(t, []string{"sku-1"}, finisher.calls)
	require.Equal(t, []string{"tenant-1"}, finisher.tenants)
	require.Equal(t, []string{"evt-1|inventory-stock-changed-clearance-promotions"}, processed.marked)
}

func TestClearancePromotionStockConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|inventory-stock-changed-clearance-promotions": true}}
	finisher := &stubClearanceSKUFinisher{}
	c := consumers.NewClearancePromotionStockConsumer(processed, finisher, &stubDLQPublisher{})

	c.Handle(newStockMsg(t, clearanceStockEvent()))

	require.Empty(t, finisher.calls)
}

// Dedup is per consumer name: the channel-sync consumer having processed the
// same event must not stop the clearance check.
func TestClearancePromotionStockConsumer_independentOfStockSyncConsumerDedup(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|inventory-stock-changed-sync": true}}
	finisher := &stubClearanceSKUFinisher{}
	c := consumers.NewClearancePromotionStockConsumer(processed, finisher, &stubDLQPublisher{})

	c.Handle(newStockMsg(t, clearanceStockEvent()))

	require.Equal(t, []string{"sku-1"}, finisher.calls)
}

func TestClearancePromotionStockConsumer_doesNotMarkProcessedWhenCheckFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	finisher := &stubClearanceSKUFinisher{err: errors.New("db down")}
	c := consumers.NewClearancePromotionStockConsumer(processed, finisher, &stubDLQPublisher{})

	c.Handle(newStockMsg(t, clearanceStockEvent()))

	require.Len(t, finisher.calls, 1)
	require.Empty(t, processed.marked)
}

func TestClearancePromotionStockConsumer_dropsMalformedPayload(t *testing.T) {
	finisher := &stubClearanceSKUFinisher{}
	c := consumers.NewClearancePromotionStockConsumer(&stubProcessedEvents{processed: map[string]bool{}}, finisher, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "inventory.stock.changed", Data: []byte("not json")})

	require.Empty(t, finisher.calls)
}
