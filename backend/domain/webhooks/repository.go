package webhooks

import "context"

// FallbackEventRepository persists FallbackEvent rows -- the safety net
// written when ReceiveWebhookUseCase can't hand a webhook off to
// JetStream. All methods are tenant-scoped via ctx (RLS), matching every
// other repository in this codebase.
type FallbackEventRepository interface {
	Save(ctx context.Context, e *FallbackEvent) error
	Get(ctx context.Context, id string) (*FallbackEvent, error)
	List(ctx context.Context, status FallbackStatus) ([]FallbackEvent, error)
	UpdateStatus(ctx context.Context, id string, status FallbackStatus) error
}

// WebhookEventRepository persists the cold-path landing record -- see
// WebhookEvent's doc comment.
type WebhookEventRepository interface {
	// Save is idempotent on (tenant_id, provider, event_id): JetStream is
	// at-least-once delivery, so a redelivered envelope whose event was
	// already persisted must be a safe no-op, not a duplicate row or an
	// error.
	Save(ctx context.Context, e *WebhookEvent) error
}
