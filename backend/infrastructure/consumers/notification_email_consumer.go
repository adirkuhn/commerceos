package consumers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// notificationEmailSender is application/notifications.
// SendNotificationEmailUseCase narrowed to point of use (ISP).
type notificationEmailSender interface {
	Execute(ctx context.Context, in appnotifications.SendNotificationEmailInput) error
}

const notificationEmailConsumerName = "notification-email-send"

// NotificationEmailConsumer sends the email half of a notification (raised
// by DeliverNotificationUseCase when the category has email on). Idempotent
// per EventID; a failed send is nak'd for redelivery (or dead-lettered).
type NotificationEmailConsumer struct {
	processed events.ProcessedEventRepository
	sender    notificationEmailSender
	dlq       dlqPublisher
}

func NewNotificationEmailConsumer(processed events.ProcessedEventRepository, sender notificationEmailSender, dlq dlqPublisher) *NotificationEmailConsumer {
	return &NotificationEmailConsumer{processed: processed, sender: sender, dlq: dlq}
}

func (c *NotificationEmailConsumer) Handle(msg *nats.Msg) {
	var event events.NotificationEmailRequested
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("notificationEmailConsumer: invalid payload, dropping (not retryable): %v", err)
		_ = msg.Ack()
		return
	}

	ctx := tenant.WithTenantID(context.Background(), event.TenantID)

	alreadyDone, err := c.processed.IsProcessed(ctx, event.EventID, notificationEmailConsumerName)
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "notificationEmailConsumer: check processed "+event.EventID, err)
		return
	}
	if alreadyDone {
		_ = msg.Ack()
		return
	}

	err = c.sender.Execute(ctx, appnotifications.SendNotificationEmailInput{
		Recipients: event.Recipients, Subject: event.Subject, Body: event.Body, Link: event.Link,
	})
	if err != nil {
		nakOrDeadLetter(c.dlq, msg, "notificationEmailConsumer: send "+event.EventID, err)
		return
	}

	if err := c.processed.MarkProcessed(ctx, event.EventID, notificationEmailConsumerName); err != nil {
		// A redelivery re-sends the email: a rare duplicate beats a lost alert.
		nakOrDeadLetter(c.dlq, msg, "notificationEmailConsumer: mark processed "+event.EventID, err)
		return
	}
	_ = msg.Ack()
}
