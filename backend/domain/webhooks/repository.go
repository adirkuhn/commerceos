package webhooks

import (
	"context"
	"time"
)

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
	// error. e.ID is always populated on return (the real row's id, even
	// when this call hit the existing row rather than inserting) -- callers
	// (ProcessWebhookEventUseCase) depend on it as the ForwardDelivery FK.
	Save(ctx context.Context, e *WebhookEvent) error
}

// ForwardRuleRepository persists a tenant's webhook fan-out configuration
// -- the Integrations page's Webhooks settings CRUD.
type ForwardRuleRepository interface {
	Create(ctx context.Context, r *ForwardRule) error
	List(ctx context.Context) ([]ForwardRule, error)
	// Delete is idempotent -- deleting an already-gone id is not an error
	// (matches DeleteExternalCategoryLinkUseCase's convention elsewhere in
	// this codebase for the same reason: a double-click or retried request
	// must not surface a spurious 404).
	Delete(ctx context.Context, id string) error
	// Match returns every active rule for provider whose Topic is either
	// topic itself or "" (provider-wide) -- see ForwardRule's doc comment.
	Match(ctx context.Context, provider, topic string) ([]ForwardRule, error)
}

// ForwardDeliveryRepository persists the fan-out delivery queue --
// DeliverForwardedWebhooksUseCase drains it, the Integrations page's
// delivery log reads it.
type ForwardDeliveryRepository interface {
	// Create is called inside the same transaction as WebhookEventRepository.Save
	// -- see ForwardDelivery's doc comment. Idempotent on
	// (webhook_event_id, rule_id): a redelivered envelope must not fan out
	// twice (migration 116's unique constraint enforces this; a conflict is
	// a safe no-op here, not an error).
	Create(ctx context.Context, d *ForwardDelivery) error
	// ListDue is a cross-tenant read for the delivery worker -- call under
	// tenant.WithSuperAdmin, same convention as
	// channels.SyncScheduleRepository.ListDue.
	ListDue(ctx context.Context, now time.Time, limit int) ([]ForwardDelivery, error)
	MarkDelivered(ctx context.Context, id string) error
	// MarkFailed records one failed attempt. When attempts (after this one)
	// has reached maxAttempts the row's status becomes permanently Failed
	// (no further nextAttemptAt); otherwise it stays Pending with
	// nextAttemptAt as the caller's computed backoff.
	MarkFailed(ctx context.Context, id string, errMsg string, nextAttemptAt time.Time, maxAttempts int) error
	// ListRecent is tenant-scoped (normal RLS ctx) -- the admin UI's
	// delivery log, newest first.
	ListRecent(ctx context.Context, limit int) ([]ForwardDelivery, error)
}
