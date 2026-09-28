package newsletter

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ConfirmUseCase completes double opt-in from the emailed link.
type ConfirmUseCase struct{ deps Deps }

func NewConfirmUseCase(deps Deps) *ConfirmUseCase { return &ConfirmUseCase{deps: deps} }

func (uc *ConfirmUseCase) Execute(ctx context.Context, token string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		s, err := uc.deps.Repo.FindByConfirmTokenHash(ctx, newsletter.HashToken(token))
		if err != nil {
			return err
		}
		if s == nil {
			return newsletter.ErrInvalidToken
		}
		if err := s.Confirm(uc.deps.Now()); err != nil {
			return err
		}
		if err := uc.deps.Repo.Update(ctx, s); err != nil {
			return err
		}
		return uc.deps.enqueue(ctx, events.NewsletterSubscriberConfirmed{Base: uc.deps.base(), TenantID: s.TenantID, SubscriberID: s.ID})
	})
	if err != nil {
		return fmt.Errorf("confirmSubscription: %w", err)
	}
	return nil
}
