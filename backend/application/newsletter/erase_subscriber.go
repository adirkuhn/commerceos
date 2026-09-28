package newsletter

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// EraseSubscriberUseCase deletes a subscriber entirely (right to erasure).
type EraseSubscriberUseCase struct{ deps Deps }

func NewEraseSubscriberUseCase(deps Deps) *EraseSubscriberUseCase {
	return &EraseSubscriberUseCase{deps: deps}
}

func (uc *EraseSubscriberUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		if err := uc.deps.Repo.Delete(ctx, id); err != nil {
			return err
		}
		return uc.deps.enqueue(ctx, events.NewsletterSubscriberErased{Base: uc.deps.base(), TenantID: tenant.IDFromContext(ctx), SubscriberID: id})
	})
	if err != nil {
		return fmt.Errorf("eraseSubscriber: %w", err)
	}
	return nil
}
