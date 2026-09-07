package apikeys

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveAPIKeyUseCase turns a storefront's bearer key into the tenant it
// belongs to. The caller (infrastructure/handlers/storefront) has no
// tenant identity yet -- that's exactly what a successful resolution
// establishes -- so the lookup runs under tenant.WithSuperAdmin (RLS
// bypass), the same cross-tenant-read-by-secret pattern
// application/iam.ClaimPairingCodeUseCase uses to resolve a scanner
// pairing code before any tenant/session exists.
type ResolveAPIKeyUseCase struct {
	db   *gorm.DB
	repo apikeysDomain.Repository
}

func NewResolveAPIKeyUseCase(db *gorm.DB, repo apikeysDomain.Repository) *ResolveAPIKeyUseCase {
	return &ResolveAPIKeyUseCase{db: db, repo: repo}
}

// Execute returns the ResolvedAPIKey (tenant, and sales channel if the key
// is scoped) the plaintext key belongs to, or apikeysDomain.ErrNotFound if
// it matches no active key -- the handler maps that to a 401, never
// leaking whether the key was wrong or revoked.
func (uc *ResolveAPIKeyUseCase) Execute(ctx context.Context, plaintextKey string) (apikeysDomain.ResolvedAPIKey, error) {
	if plaintextKey == "" {
		return apikeysDomain.ResolvedAPIKey{}, apikeysDomain.ErrNotFound
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var resolved apikeysDomain.ResolvedAPIKey
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		found, err := uc.repo.FindActiveTenantIDByHash(ctx, hashKey(plaintextKey))
		if err != nil {
			return err
		}
		resolved = found
		return nil
	})
	if err != nil {
		return apikeysDomain.ResolvedAPIKey{}, fmt.Errorf("resolveApiKey: %w", err)
	}
	return resolved, nil
}
