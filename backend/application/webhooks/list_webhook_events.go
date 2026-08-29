package webhooks

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListWebhookEventsUseCase is the admin-facing "see all messages" log --
// the Webhooks page's Messages tab, optionally filtered to one sender.
type ListWebhookEventsUseCase struct {
	db        *gorm.DB
	eventRepo webhooks.WebhookEventRepository
}

func NewListWebhookEventsUseCase(db *gorm.DB, eventRepo webhooks.WebhookEventRepository) *ListWebhookEventsUseCase {
	return &ListWebhookEventsUseCase{db: db, eventRepo: eventRepo}
}

func (uc *ListWebhookEventsUseCase) Execute(ctx context.Context, provider string, limit int) ([]webhooks.WebhookEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []webhooks.WebhookEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.eventRepo.List(ctx, provider, limit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWebhookEvents: %w", err)
	}
	return out, nil
}
