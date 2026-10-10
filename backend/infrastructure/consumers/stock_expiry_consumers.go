package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// StockExpiryBellConsumerName / StockExpiryDefaultsConsumerName are the
// processed_events keys (and JetStream durables) of the two consumers below.
const (
	StockExpiryBellConsumerName     = "stock-close-to-expiry-bell"
	StockExpiryDefaultsConsumerName = "stock-expiry-message-defaults"
)

// stockExpiryBell is application/notifications'
// NotifyStockCloseToExpiryUseCase narrowed to point of use.
type stockExpiryBell interface {
	Execute(ctx context.Context, windowDays, itemCount int) error
}

// StockExpiryBellConsumer puts each inventory.stock.close_to_expiry_reported
// report in the tenant's bell. Idempotent per EventID.
type StockExpiryBellConsumer struct {
	processed events.ProcessedEventRepository
	bell      stockExpiryBell
	dlq       dlqPublisher
}

func NewStockExpiryBellConsumer(processed events.ProcessedEventRepository, bell stockExpiryBell, dlq dlqPublisher) *StockExpiryBellConsumer {
	return &StockExpiryBellConsumer{processed: processed, bell: bell, dlq: dlq}
}

func (c *StockExpiryBellConsumer) Handle(msg *nats.Msg) {
	var event events.StockCloseToExpiryReported
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("stockExpiryBellConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "stockExpiryBellConsumer")
		return
	}
	handleOnce(msg, c.processed, c.dlq, event.EventID, event.TenantID, StockExpiryBellConsumerName, func(ctx context.Context) error {
		return c.bell.Execute(ctx, event.WindowDays, event.ItemCount)
	})
}

// stockExpiryDefaultsEnsurer is application/messagecenter's
// EnsureStockExpiryDefaultsUseCase narrowed to point of use.
type stockExpiryDefaultsEnsurer interface {
	Execute(ctx context.Context) error
}

// StockExpiryDefaultsConsumer gives a tenant the default "stock close to
// expiry" Message Center template + rule when a settings.tenant_settings.updated
// event shows it turned the report on. Idempotent per EventID (and the use
// case itself never duplicates an existing rule).
type StockExpiryDefaultsConsumer struct {
	processed events.ProcessedEventRepository
	ensurer   stockExpiryDefaultsEnsurer
	dlq       dlqPublisher
}

func NewStockExpiryDefaultsConsumer(processed events.ProcessedEventRepository, ensurer stockExpiryDefaultsEnsurer, dlq dlqPublisher) *StockExpiryDefaultsConsumer {
	return &StockExpiryDefaultsConsumer{processed: processed, ensurer: ensurer, dlq: dlq}
}

func (c *StockExpiryDefaultsConsumer) Handle(msg *nats.Msg) {
	var event struct {
		EventID  string                        `json:"event_id"`
		TenantID string                        `json:"tenant_id"`
		Changes  map[string]events.FieldChange `json:"changes"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("stockExpiryDefaultsConsumer: invalid payload, dropping (not retryable): %v", err)
		ackOrLog(msg, "stockExpiryDefaultsConsumer")
		return
	}
	turnedOn := event.Changes["expiry_alert_enabled"].After == true
	handleOnce(msg, c.processed, c.dlq, event.EventID, event.TenantID, StockExpiryDefaultsConsumerName, func(ctx context.Context) error {
		if !turnedOn {
			return nil
		}
		return c.ensurer.Execute(ctx)
	})
}

// handleOnce runs fn under the event's tenant unless consumer already
// processed eventID, then marks it processed and acks. A failure naks for
// redelivery (or dead-letters) without marking.
func handleOnce(msg *nats.Msg, processed events.ProcessedEventRepository, dlq dlqPublisher, eventID, tenantID, consumer string, fn func(ctx context.Context) error) {
	ctx := tenant.WithTenantID(context.Background(), tenantID)
	done, err := processed.IsProcessed(ctx, eventID, consumer)
	if err != nil {
		nakOrDeadLetter(dlq, msg, consumer+": check processed "+eventID, err)
		return
	}
	if done {
		ackOrLog(msg, consumer)
		return
	}
	if err := fn(ctx); err != nil {
		nakOrDeadLetter(dlq, msg, consumer+": handle "+eventID, err)
		return
	}
	if err := processed.MarkProcessed(ctx, eventID, consumer); err != nil {
		nakOrDeadLetter(dlq, msg, consumer+": mark processed "+eventID, err)
		return
	}
	ackOrLog(msg, consumer)
}
