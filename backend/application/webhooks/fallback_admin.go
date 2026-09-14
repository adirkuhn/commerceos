package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListWebhookFallbackEventsUseCase is the admin-facing read side of the
// fallback table -- what an operator triages after a NATS outage.
type ListWebhookFallbackEventsUseCase struct {
	db           *gorm.DB
	fallbackRepo webhooks.FallbackEventRepository
}

func NewListWebhookFallbackEventsUseCase(db *gorm.DB, fallbackRepo webhooks.FallbackEventRepository) *ListWebhookFallbackEventsUseCase {
	return &ListWebhookFallbackEventsUseCase{db: db, fallbackRepo: fallbackRepo}
}

func (uc *ListWebhookFallbackEventsUseCase) Execute(ctx context.Context, status webhooks.FallbackStatus) ([]webhooks.FallbackEvent, error) {
	var out []webhooks.FallbackEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.fallbackRepo.List(ctx, status)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWebhookFallbackEvents: %w", err)
	}
	return out, nil
}

// RetryWebhookFallbackEventUseCase re-attempts publishing one fallback
// row's saved payload to JetStream -- the manual half of "retry or
// remove". On success the row is marked Republished (so it drops out of
// the default pending_republish list); on failure it's left
// PendingRepublish for another attempt, and the publish error is
// returned so the admin UI can show why.
type RetryWebhookFallbackEventUseCase struct {
	db           *gorm.DB
	fallbackRepo webhooks.FallbackEventRepository
	publisher    Publisher
}

func NewRetryWebhookFallbackEventUseCase(db *gorm.DB, fallbackRepo webhooks.FallbackEventRepository, publisher Publisher) *RetryWebhookFallbackEventUseCase {
	return &RetryWebhookFallbackEventUseCase{db: db, fallbackRepo: fallbackRepo, publisher: publisher}
}

func (uc *RetryWebhookFallbackEventUseCase) Execute(ctx context.Context, id string) error {
	var event *webhooks.FallbackEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		event, err = uc.fallbackRepo.Get(ctx, id)
		return err
	})
	if err != nil {
		return fmt.Errorf("retryWebhookFallbackEvent: %w", err)
	}

	// Must match the hot path's shape (ReceiveWebhookUseCase.Execute) --
	// consumers expect the Envelope wrapper, not the bare raw body.
	envelope := webhooks.Envelope{
		EventID: event.EventID, TenantID: event.TenantID, Provider: event.Provider,
		ReceivedAt: event.ReceivedAt, RawPayload: event.Payload, Headers: event.Headers,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("retryWebhookFallbackEvent: marshal envelope: %w", err)
	}

	subject := webhookSubjectPrefix + event.Provider
	if err := uc.publisher.Publish(subject, data); err != nil {
		return fmt.Errorf("retryWebhookFallbackEvent: publish: %w", err)
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.fallbackRepo.UpdateStatus(ctx, id, webhooks.FallbackStatusRepublished)
	})
	if err != nil {
		return fmt.Errorf("retryWebhookFallbackEvent: mark republished: %w", err)
	}
	return nil
}

// AbandonWebhookFallbackEventUseCase is the other half of "retry or
// remove" -- marks a fallback row Abandoned rather than hard-deleting it,
// keeping an audit trail of what was received and deliberately not
// processed.
type AbandonWebhookFallbackEventUseCase struct {
	db           *gorm.DB
	fallbackRepo webhooks.FallbackEventRepository
}

func NewAbandonWebhookFallbackEventUseCase(db *gorm.DB, fallbackRepo webhooks.FallbackEventRepository) *AbandonWebhookFallbackEventUseCase {
	return &AbandonWebhookFallbackEventUseCase{db: db, fallbackRepo: fallbackRepo}
}

func (uc *AbandonWebhookFallbackEventUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.fallbackRepo.UpdateStatus(ctx, id, webhooks.FallbackStatusAbandoned)
	})
	if err != nil {
		return fmt.Errorf("abandonWebhookFallbackEvent: %w", err)
	}
	return nil
}
