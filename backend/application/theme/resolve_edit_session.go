package theme

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveEditSessionUseCase turns a bearer token from storefront-frontend's
// embedded Puck editor into the tenant/channel it authorizes -- the caller
// has no tenant identity yet (that's exactly what a successful resolution
// establishes), so the lookup runs under tenant.WithSuperAdmin (RLS
// bypass), the same cross-tenant-read-by-secret pattern
// application/apikeys.ResolveAPIKeyUseCase already uses for the (entirely
// separate, lower-privilege) storefront key.
type ResolveEditSessionUseCase struct {
	db   *gorm.DB
	repo themeDomain.EditSessionRepository
}

func NewResolveEditSessionUseCase(db *gorm.DB, repo themeDomain.EditSessionRepository) *ResolveEditSessionUseCase {
	return &ResolveEditSessionUseCase{db: db, repo: repo}
}

// Execute returns themeDomain.ErrEditSessionNotFound for an unknown or
// expired token -- the handler maps that to a 401, never leaking which.
func (uc *ResolveEditSessionUseCase) Execute(ctx context.Context, token string) (themeDomain.EditSession, error) {
	if token == "" {
		return themeDomain.EditSession{}, themeDomain.ErrEditSessionNotFound
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var found *themeDomain.EditSession
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		f, err := uc.repo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		found = f
		return nil
	})
	if err != nil {
		return themeDomain.EditSession{}, fmt.Errorf("resolveEditSession: %w", err)
	}
	if found == nil || found.IsExpired(time.Now()) {
		return themeDomain.EditSession{}, themeDomain.ErrEditSessionNotFound
	}
	return *found, nil
}
