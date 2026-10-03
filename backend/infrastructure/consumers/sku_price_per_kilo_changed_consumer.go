package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// weightPackageRepricer is what this consumer needs (ISP): reprice one
// pool's packages -- see application/pms.RepriceWeightPackagesUseCase.
type weightPackageRepricer interface {
	Reprice(ctx context.Context, poolSKUID string) error
}

const skuPricePerKiloChangedConsumerName = "weight-package-repricer"

// SKUPricePerKiloChangedConsumer reacts to pms.sku.price_per_kilo_changed by
// repricing the weight pool's packages to the new rate. Idempotent via
// ProcessedEventRepository (and the repricer itself skips packages already
// at the right price); acks only after success, nacks on failure.
type SKUPricePerKiloChangedConsumer struct {
	processed events.ProcessedEventRepository
	repricer  weightPackageRepricer
	dlq       dlqPublisher
}

func NewSKUPricePerKiloChangedConsumer(processed events.ProcessedEventRepository, repricer weightPackageRepricer, dlq dlqPublisher) *SKUPricePerKiloChangedConsumer {
	return &SKUPricePerKiloChangedConsumer{processed: processed, repricer: repricer, dlq: dlq}
}

func (c *SKUPricePerKiloChangedConsumer) Handle(msg *nats.Msg) {
	var event events.SKUPricePerKiloChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("skuPricePerKiloChangedConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "skuPricePerKiloChangedConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, skuPricePerKiloChangedConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "skuPricePerKiloChangedConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "skuPricePerKiloChangedConsumer")
		return
	}

	if err := c.repricer.Reprice(ctx, event.SKUID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "skuPricePerKiloChangedConsumer: reprice packages of "+event.SKUID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, skuPricePerKiloChangedConsumerName); err != nil {
		// safe to retry: repricing skips packages already at the right price
		nakOrDeadLetter(c.dlq, msg, "skuPricePerKiloChangedConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "skuPricePerKiloChangedConsumer")
}
