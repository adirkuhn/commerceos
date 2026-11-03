package consumers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubCostPriceSyncer struct {
	skus, tenants []string
	err           error
}

func (s *stubCostPriceSyncer) Execute(ctx context.Context, skuID string) error {
	s.skus = append(s.skus, skuID)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func TestCostPriceOnSupplierCostRecorded_syncsTheSKU(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubCostPriceSyncer{}
	c := consumers.NewCostPriceOnSupplierCostRecordedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SupplierCostRecorded.Subject, "evt-1", "sku-1"))

	assert.Equal(t, []string{"sku-1"}, syncer.skus)
	assert.Equal(t, []string{"tenant-1"}, syncer.tenants)
	assert.Equal(t, []string{"evt-1|" + consumers.CostPriceOnSupplierCostRecordedConsumerName}, processed.marked)
}

func TestCostPriceOnSupplierCostRecorded_skipsAlreadyProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|" + consumers.CostPriceOnSupplierCostRecordedConsumerName: true}}
	syncer := &stubCostPriceSyncer{}
	c := consumers.NewCostPriceOnSupplierCostRecordedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SupplierCostRecorded.Subject, "evt-1", "sku-1"))

	assert.Empty(t, syncer.skus)
}

func TestCostPriceOnSupplierCostRecorded_failureIsNotMarkedProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	c := consumers.NewCostPriceOnSupplierCostRecordedConsumer(processed, &stubCostPriceSyncer{err: errors.New("boom")}, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SupplierCostRecorded.Subject, "evt-1", "sku-1"))

	assert.Empty(t, processed.marked)
}

func TestCostPriceOnSupplierCostRecorded_dropsAMalformedPayload(t *testing.T) {
	syncer := &stubCostPriceSyncer{}
	c := consumers.NewCostPriceOnSupplierCostRecordedConsumer(&stubProcessedEvents{processed: map[string]bool{}}, syncer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: events.SupplierCostRecorded.Subject, Data: []byte("nope")})

	assert.Empty(t, syncer.skus)
}
