package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// storefrontKeyResolver is domain/settings.Repository narrowed to the one
// method this use case needs (ISP).
type storefrontKeyResolver interface {
	FindTenantIDByStorefrontKeyHash(ctx context.Context, hash string) (string, error)
}

// ResolveStorefrontKeyUseCase turns a storefront's bearer key into the
// tenant it belongs to. The caller (infrastructure/handlers/storefront) has
// no tenant identity yet -- that's exactly what a successful resolution
// establishes -- so the lookup runs under tenant.WithSuperAdmin (RLS
// bypass), the same cross-tenant-read-by-secret pattern
// application/iam.ClaimPairingCodeUseCase uses to resolve a scanner pairing
// code before any tenant/session exists.
type ResolveStorefrontKeyUseCase struct {
	db           *gorm.DB
	settingsRepo storefrontKeyResolver
}

func NewResolveStorefrontKeyUseCase(db *gorm.DB, settingsRepo storefrontKeyResolver) *ResolveStorefrontKeyUseCase {
	return &ResolveStorefrontKeyUseCase{db: db, settingsRepo: settingsRepo}
}

// Execute returns the TenantID the plaintext key belongs to, or
// settingsDomain.ErrStorefrontKeyNotFound if it matches no tenant -- the
// handler maps that to a 401, never leaking which part of the key was
// wrong.
func (uc *ResolveStorefrontKeyUseCase) Execute(ctx context.Context, plaintextKey string) (string, error) {
	if plaintextKey == "" {
		return "", settingsDomain.ErrStorefrontKeyNotFound
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var tenantID string
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		found, err := uc.settingsRepo.FindTenantIDByStorefrontKeyHash(ctx, hashStorefrontKey(plaintextKey))
		if err != nil {
			return err
		}
		tenantID = found
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("resolveStorefrontKey: %w", err)
	}
	return tenantID, nil
}
