package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveStorefrontKeyUseCase turns a storefront's bearer key into the
// tenant it belongs to. The caller (infrastructure/handlers/storefront) has
// no tenant identity yet -- that's exactly what a successful resolution
// establishes -- so the lookup runs under tenant.WithSuperAdmin (RLS
// bypass), the same cross-tenant-read-by-secret pattern
// application/iam.ClaimPairingCodeUseCase uses to resolve a scanner pairing
// code before any tenant/session exists.
type ResolveStorefrontKeyUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewResolveStorefrontKeyUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *ResolveStorefrontKeyUseCase {
	return &ResolveStorefrontKeyUseCase{db: db, connRepo: connRepo}
}

// Execute returns the TenantID the plaintext key belongs to, or
// integrationsDomain.ErrConnectionNotFound if it matches no connection --
// the handler maps that to a 401, never leaking which part of the key was
// wrong.
func (uc *ResolveStorefrontKeyUseCase) Execute(ctx context.Context, plaintextKey string) (string, error) {
	if plaintextKey == "" {
		return "", integrationsDomain.ErrConnectionNotFound
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var tenantID string
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByStorefrontKeyHash(ctx, hashStorefrontKey(plaintextKey))
		if err != nil {
			return err
		}
		tenantID = conn.TenantID
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("resolveStorefrontKey: %w", err)
	}
	return tenantID, nil
}
