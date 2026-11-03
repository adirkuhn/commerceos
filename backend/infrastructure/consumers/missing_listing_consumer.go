package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// missingListingHandler is application/channels.HandleMissingListingUseCase (ISP).
type missingListingHandler interface {
	Execute(ctx context.Context, in appchannels.HandleMissingListingInput) error
}

const MissingListingConsumerName = "missing-listing-handler"

// MissingListingConsumer handles channels.product_listing.orphaned (COM-18):
// a product deleted in WooCommerce is created again when its sales channel
// still allows it, else left marked. Idempotent via
// ProcessedEventRepository, and the use case is a no-op once nothing is
// missing any more.
type MissingListingConsumer struct {
	processed events.ProcessedEventRepository
	handler   missingListingHandler
	dlq       dlqPublisher
}

func NewMissingListingConsumer(processed events.ProcessedEventRepository, handler missingListingHandler, dlq dlqPublisher) *MissingListingConsumer {
	return &MissingListingConsumer{processed: processed, handler: handler, dlq: dlq}
}

// Name is the durable consumer name (and dedup key).
func (c *MissingListingConsumer) Name() string { return MissingListingConsumerName }

// Handle never panics: a malformed payload is acked and dropped, a failure
// is nacked for JetStream to redeliver.
func (c *MissingListingConsumer) Handle(msg *nats.Msg) {
	name := MissingListingConsumerName
	var e events.EntityAction
	err := json.Unmarshal(msg.Data, &e)
	productID, _ := e.Data["product_id"].(string)
	connectionID, _ := e.Data["connection_id"].(string)
	if err != nil || productID == "" || connectionID == "" {
		log.Printf("%s: invalid payload, dropping (not retryable): %v", name, err)
		ackOrLog(msg, name)
		return
	}
	ctx := tenant.WithTenantID(context.Background(), e.TenantID)

	done, err := c.processed.IsProcessed(ctx, e.EventID, name)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": check processed "+e.EventID, err)
		return
	}
	if done {
		ackOrLog(msg, name)
		return
	}
	if err := c.handler.Execute(ctx, appchannels.HandleMissingListingInput{ProductID: productID, ConnectionID: connectionID}); err != nil {
		nakOrDeadLetter(c.dlq, msg, name+": product "+productID, err)
		return
	}
	if err := c.processed.MarkProcessed(ctx, e.EventID, name); err != nil {
		// safe to retry: nothing missing any more is a no-op
		nakOrDeadLetter(c.dlq, msg, name+": mark processed "+e.EventID, err)
		return
	}
	ackOrLog(msg, name)
}
