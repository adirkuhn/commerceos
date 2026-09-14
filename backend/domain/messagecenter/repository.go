package messagecenter

import "context"

type MessageTemplateRepository interface {
	Create(ctx context.Context, t *MessageTemplate) error
	FindByID(ctx context.Context, id string) (*MessageTemplate, error)
	ListByTenant(ctx context.Context) ([]MessageTemplate, error)
	Update(ctx context.Context, t *MessageTemplate) error
	Delete(ctx context.Context, id string) error
}

type AutomationRuleRepository interface {
	Create(ctx context.Context, r *AutomationRule) error
	FindByID(ctx context.Context, id string) (*AutomationRule, error)
	ListByTenant(ctx context.Context) ([]AutomationRule, error)
	// ListActiveBySubject returns every enabled rule whose TriggerSubject
	// matches subject, scoped to whichever tenant is set on ctx -- the
	// automation engine (a later slice) calls this once per received event,
	// after setting the event's own TenantID on ctx the same way every
	// other consumer does.
	ListActiveBySubject(ctx context.Context, subject string) ([]AutomationRule, error)
	Update(ctx context.Context, r *AutomationRule) error
	Delete(ctx context.Context, id string) error
}
