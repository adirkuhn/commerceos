package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveSessionUseCase turns a bearer token into its session. The tenant
// isn't known yet (the token is the only credential), so the lookup runs
// with RLS bypassed -- by exact token hash only, never listing anything.
// Every failure is ErrSessionInvalid, so callers can't probe tokens.
type ResolveSessionUseCase struct{ deps Deps }

func NewResolveSessionUseCase(deps Deps) *ResolveSessionUseCase {
	return &ResolveSessionUseCase{deps: deps}
}

func (uc *ResolveSessionUseCase) Execute(ctx context.Context, rawToken string, purpose theme.SessionPurpose) (*theme.Session, error) {
	var found *theme.Session
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.deps.DB, func(ctx context.Context) error {
		var err error
		found, err = uc.deps.Sessions.FindByTokenHash(ctx, theme.HashToken(rawToken))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("resolveSession: %w", err)
	}
	if found == nil {
		return nil, fmt.Errorf("resolveSession: %w", theme.ErrSessionInvalid)
	}
	if err := found.Authorize(purpose, uc.deps.Now()); err != nil {
		return nil, fmt.Errorf("resolveSession: %w", err)
	}
	return found, nil
}
