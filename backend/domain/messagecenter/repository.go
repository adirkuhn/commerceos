package messagecenter

import "context"

type MessageTemplateRepository interface {
	Create(ctx context.Context, t *MessageTemplate) error
	FindByID(ctx context.Context, id string) (*MessageTemplate, error)
	ListByTenant(ctx context.Context) ([]MessageTemplate, error)
	Update(ctx context.Context, t *MessageTemplate) error
	Delete(ctx context.Context, id string) error
}
