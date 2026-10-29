package consumers

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// moneybirdAutoSyncer is application/moneybird.AutoSyncUseCase (ISP): it
// sends the quote/order when the shop has Moneybird auto sync on, and
// returns an error only for a failure worth retrying.
type moneybirdAutoSyncer interface {
	SalesOrder(ctx context.Context, orderID string) error
	Quote(ctx context.Context, quoteID string) error
}

const (
	MoneybirdSalesOrderCreatedConsumerName       = "moneybird-sales-order-created"
	MoneybirdSalesOrderStatusChangedConsumerName = "moneybird-sales-order-status-changed"
)

// MoneybirdQuoteChangedConsumerName is the durable name for one quote
// subject (created/updated) -- each subject has its own consumer.
func MoneybirdQuoteChangedConsumerName(subject string) string {
	return "moneybird-" + strings.ReplaceAll(subject, ".", "-")
}

// MoneybirdAutoSyncConsumer sends a quote or sales order to Moneybird when
// one of its events arrives (see AutoSyncUseCase -- a no-op unless auto
// sync is on). One instance per event subject, built by the constructors
// below. Idempotent via ProcessedEventRepository; the send itself never
// books twice either.
type MoneybirdAutoSyncConsumer struct {
	name      string
	processed events.ProcessedEventRepository
	dlq       dlqPublisher
	// decode reads the tenant, event id and quote/order id off the payload.
	decode func(data []byte) (tenantID, eventID, sourceID string, err error)
	send   func(ctx context.Context, sourceID string) error
}

func decodeEntityAction(data []byte) (string, string, string, error) {
	var e events.EntityAction
	err := json.Unmarshal(data, &e)
	return e.TenantID, e.EventID, e.EntityID, err
}

// NewMoneybirdSalesOrderCreatedConsumer books a new order.
func NewMoneybirdSalesOrderCreatedConsumer(processed events.ProcessedEventRepository, syncer moneybirdAutoSyncer, dlq dlqPublisher) *MoneybirdAutoSyncConsumer {
	return &MoneybirdAutoSyncConsumer{
		name: MoneybirdSalesOrderCreatedConsumerName, processed: processed, dlq: dlq,
		decode: decodeEntityAction, send: syncer.SalesOrder,
	}
}

// NewMoneybirdSalesOrderStatusChangedConsumer catches an order becoming
// paid after it was booked, so its payment follows.
func NewMoneybirdSalesOrderStatusChangedConsumer(processed events.ProcessedEventRepository, syncer moneybirdAutoSyncer, dlq dlqPublisher) *MoneybirdAutoSyncConsumer {
	return &MoneybirdAutoSyncConsumer{
		name: MoneybirdSalesOrderStatusChangedConsumerName, processed: processed, dlq: dlq,
		decode: func(data []byte) (string, string, string, error) {
			var e events.SalesOrderStatusChanged
			err := json.Unmarshal(data, &e)
			return e.TenantID, e.EventID, e.SalesOrderID, err
		},
		send: syncer.SalesOrder,
	}
}

// NewMoneybirdQuoteChangedConsumer sends a quote on subject
// (quotes.quote.created or quotes.quote.updated).
func NewMoneybirdQuoteChangedConsumer(subject string, processed events.ProcessedEventRepository, syncer moneybirdAutoSyncer, dlq dlqPublisher) *MoneybirdAutoSyncConsumer {
	return &MoneybirdAutoSyncConsumer{
		name: MoneybirdQuoteChangedConsumerName(subject), processed: processed, dlq: dlq,
		decode: decodeEntityAction, send: syncer.Quote,
	}
}

// Name is the durable consumer name (and dedup key).
func (c *MoneybirdAutoSyncConsumer) Name() string { return c.name }

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *MoneybirdAutoSyncConsumer) Handle(msg *nats.Msg) {
	tenantID, eventID, sourceID, err := c.decode(msg.Data)
	if err != nil || sourceID == "" {
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
	if err := c.send(ctx, sourceID); err != nil {
		nakOrDeadLetter(c.dlq, msg, c.name+": send "+sourceID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, eventID, c.name); err != nil {
		// safe to retry: sending again never books twice
		nakOrDeadLetter(c.dlq, msg, c.name+": mark processed "+eventID, err)
		return
	}
	ackOrLog(msg, c.name)
}
