package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// bolShipmentConfirmer is application/channels.ConfirmBolShipmentUseCase
// narrowed to point of use (ISP).
type bolShipmentConfirmer interface {
	Execute(ctx context.Context, salesOrderID string) error
}

const bolShipmentConfirmationConsumerName = "bol-shipment-confirmation"

// BolShipmentConfirmationConsumer reacts to oms.sales_order.status_changed
// by telling Bol that a shipped Bol order left the warehouse (see
// ConfirmBolShipmentUseCase -- it ignores every other order and status).
// An independent subscriber on the same subject as
// SalesOrderStatusChangedConsumer, with its own durable and processed-event
// key. Idempotent: ProcessedEventRepository skips redelivered events, and
// the use case itself never re-sends a shipment Bol already has.
type BolShipmentConfirmationConsumer struct {
	processed events.ProcessedEventRepository
	confirmer bolShipmentConfirmer
	dlq       dlqPublisher
}

func NewBolShipmentConfirmationConsumer(processed events.ProcessedEventRepository, confirmer bolShipmentConfirmer, dlq dlqPublisher) *BolShipmentConfirmationConsumer {
	return &BolShipmentConfirmationConsumer{processed: processed, confirmer: confirmer, dlq: dlq}
}

func (c *BolShipmentConfirmationConsumer) Handle(msg *nats.Msg) {
	var event events.SalesOrderStatusChanged
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("bolShipmentConfirmationConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "bolShipmentConfirmationConsumer")
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, bolShipmentConfirmationConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "bolShipmentConfirmationConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		ackOrLog(msg, "bolShipmentConfirmationConsumer")
		return
	}

	if err := c.confirmer.Execute(ctx, event.SalesOrderID); err != nil {
		nakOrDeadLetter(c.dlq, msg, "bolShipmentConfirmationConsumer: confirm order "+event.SalesOrderID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, bolShipmentConfirmationConsumerName); err != nil {
		// safe to retry: the use case skips a shipment already confirmed
		nakOrDeadLetter(c.dlq, msg, "bolShipmentConfirmationConsumer: mark processed "+event.EventID, err)
		return
	}

	ackOrLog(msg, "bolShipmentConfirmationConsumer")
}
