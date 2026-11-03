package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// unitCostSyncer is application/pms.SyncUnitCostFromCaseUseCase (ISP).
type unitCostSyncer interface {
	SyncFromCase(ctx context.Context, skuID string) error
	SyncProduct(ctx context.Context, productID string) error
}

const (
	UnitCostOnCaseCostChangedConsumerName          = "unit-cost-on-case-cost-changed"
	UnitCostOnCaseInventoryCostChangedConsumerName = "unit-cost-on-case-inventory-cost-changed"
	UnitCostOnProductChangedConsumerName           = "unit-cost-on-product-changed"
)

// UnitCostFromCaseConsumer keeps a unit SKU's cost in line with the sealed
// case it is sold out of (see SyncUnitCostFromCaseUseCase): when a SKU's
// cost changes (a case was received) and when a product changes (a case
// link was made or changed). One instance per subject, built by the
// constructors below. Idempotent via ProcessedEventRepository, and the
// sync itself changes nothing when the unit is already right.
type UnitCostFromCaseConsumer struct {
	name      string
	processed events.ProcessedEventRepository
	dlq       dlqPublisher
	// decode reads the tenant, event id and SKU/product id off the payload.
	decode func(data []byte) (tenantID, eventID, id string, err error)
	sync   func(ctx context.Context, id string) error
}

// NewUnitCostOnCaseCostChangedConsumer handles pms.sku.cost_price_changed.
func NewUnitCostOnCaseCostChangedConsumer(processed events.ProcessedEventRepository, syncer unitCostSyncer, dlq dlqPublisher) *UnitCostFromCaseConsumer {
	return &UnitCostFromCaseConsumer{
		name: UnitCostOnCaseCostChangedConsumerName, processed: processed, dlq: dlq,
		decode: decodeEntityAction, sync: syncer.SyncFromCase,
	}
}

// NewUnitCostOnCaseInventoryCostChangedConsumer handles
// pms.sku.inventory_cost_changed: what the case stock cost moved, so a unit
// with nothing on hand takes it.
func NewUnitCostOnCaseInventoryCostChangedConsumer(processed events.ProcessedEventRepository, syncer unitCostSyncer, dlq dlqPublisher) *UnitCostFromCaseConsumer {
	return &UnitCostFromCaseConsumer{
		name: UnitCostOnCaseInventoryCostChangedConsumerName, processed: processed, dlq: dlq,
		decode: decodeEntityAction, sync: syncer.SyncFromCase,
	}
}

// NewUnitCostOnProductChangedConsumer handles pms.product.changed.
func NewUnitCostOnProductChangedConsumer(processed events.ProcessedEventRepository, syncer unitCostSyncer, dlq dlqPublisher) *UnitCostFromCaseConsumer {
	return &UnitCostFromCaseConsumer{
		name: UnitCostOnProductChangedConsumerName, processed: processed, dlq: dlq,
		decode: func(data []byte) (string, string, string, error) {
			var e events.ProductChanged
			err := json.Unmarshal(data, &e)
			return e.TenantID, e.EventID, e.ProductID, err
		},
		sync: syncer.SyncProduct,
	}
}

// Name is the durable consumer name (and dedup key).
func (c *UnitCostFromCaseConsumer) Name() string { return c.name }

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *UnitCostFromCaseConsumer) Handle(msg *nats.Msg) {
	tenantID, eventID, id, err := c.decode(msg.Data)
	if err != nil || id == "" {
		log.Printf("%s: invalid payload, dropping (not retryable): %v", c.name, err)
		ackOrLog(msg, c.name)
		return
	}
	ctx := tenant.WithTenantID(context.Background(), tenantID)

	done, err := c.processed.IsProcessed(ctx, eventID, c.name)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, c.name+": check processed "+eventID, err)
		return
	}
	if done {
		ackOrLog(msg, c.name)
		return
	}
	if err := c.sync(ctx, id); err != nil {
		nakOrDeadLetter(c.dlq, msg, c.name+": sync "+id, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, eventID, c.name); err != nil {
		// safe to retry: a repeated sync changes nothing
		nakOrDeadLetter(c.dlq, msg, c.name+": mark processed "+eventID, err)
		return
	}
	ackOrLog(msg, c.name)
}
