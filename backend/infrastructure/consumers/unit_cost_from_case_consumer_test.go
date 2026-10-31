package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubUnitCostSyncer struct {
	cases, products []string
	tenants         []string
	err             error
}

func (s *stubUnitCostSyncer) SyncFromCase(ctx context.Context, skuID string) error {
	s.cases = append(s.cases, skuID)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func (s *stubUnitCostSyncer) SyncProduct(ctx context.Context, productID string) error {
	s.products = append(s.products, productID)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func productChangedMsg(t *testing.T, eventID, productID string) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(events.ProductChanged{Base: events.Base{EventID: eventID}, TenantID: "tenant-1", ProductID: productID})
	require.NoError(t, err)
	return &nats.Msg{Subject: events.ProductChanged{}.EventName(), Data: payload}
}

func TestUnitCostFromCaseConsumer_costChangeSyncsTheSKUsUnit(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubUnitCostSyncer{}
	c := consumers.NewUnitCostOnCaseCostChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SKUCostPriceChanged.Subject, "evt-1", "box-1"))

	assert.Equal(t, []string{"box-1"}, syncer.cases)
	assert.Equal(t, []string{"tenant-1"}, syncer.tenants)
	assert.Equal(t, []string{"evt-1|" + consumers.UnitCostOnCaseCostChangedConsumerName}, processed.marked)
}

func TestUnitCostFromCaseConsumer_productChangeSyncsTheProductsCases(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubUnitCostSyncer{}
	c := consumers.NewUnitCostOnProductChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(productChangedMsg(t, "evt-2", "p-1"))

	assert.Equal(t, []string{"p-1"}, syncer.products)
	assert.Equal(t, []string{"evt-2|" + consumers.UnitCostOnProductChangedConsumerName}, processed.marked)
}

func TestUnitCostFromCaseConsumer_namesAreDistinct(t *testing.T) {
	assert.NotEqual(t, consumers.UnitCostOnCaseCostChangedConsumerName, consumers.UnitCostOnProductChangedConsumerName)
}

func TestUnitCostFromCaseConsumer_skipsAlreadyProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|" + consumers.UnitCostOnCaseCostChangedConsumerName: true}}
	syncer := &stubUnitCostSyncer{}
	c := consumers.NewUnitCostOnCaseCostChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SKUCostPriceChanged.Subject, "evt-1", "box-1"))

	assert.Empty(t, syncer.cases)
}

func TestUnitCostFromCaseConsumer_failureIsRetriedNotMarked(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubUnitCostSyncer{err: errors.New("db down")}
	c := consumers.NewUnitCostOnCaseCostChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(entityActionMsg(t, events.SKUCostPriceChanged.Subject, "evt-1", "box-1"))

	assert.Len(t, syncer.cases, 1)
	assert.Empty(t, processed.marked)
}

func TestUnitCostFromCaseConsumer_dropsMalformedPayload(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubUnitCostSyncer{}
	c := consumers.NewUnitCostOnProductChangedConsumer(processed, syncer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "pms.product.changed", Data: []byte("not json")})

	assert.Empty(t, syncer.products)
	assert.Empty(t, processed.marked)
}
