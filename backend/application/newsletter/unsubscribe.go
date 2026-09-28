package newsletter

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UnsubscribeUseCase handles the unsubscribe link in every email.
type UnsubscribeUseCase struct{ deps Deps }

func NewUnsubscribeUseCase(deps Deps) *UnsubscribeUseCase { return &UnsubscribeUseCase{deps: deps} }

func (uc *UnsubscribeUseCase) Execute(ctx context.Context, token string) error {
	err := tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		s, err := uc.deps.Repo.FindByUnsubscribeToken(ctx, token)
		if err != nil {
			return err
		}
		if s == nil || token == "" {
			return newsletter.ErrInvalidToken
		}
		s.Unsubscribe(uc.deps.Now())
		return uc.deps.Repo.Update(ctx, s)
	})
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	return nil
}
