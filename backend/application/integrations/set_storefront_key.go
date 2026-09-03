package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

// storefrontKeyHashSetter is domain/settings.Repository narrowed to the one
// method this use case needs (ISP).
type storefrontKeyHashSetter interface {
	SetStorefrontKeyHash(ctx context.Context, hash string) error
}

// SetStorefrontKeyResult carries the PLAINTEXT key back to the caller
// exactly once, at generation time -- only its hash persists, so the
// merchant must paste it into the storefront's (e.g. the WooCommerce
// plugin's) settings right away.
type SetStorefrontKeyResult struct {
	PlaintextKey string
}

// SetStorefrontKeyUseCase (re)generates the current tenant's storefront
// key -- the bearer token a storefront presents to call the authenticated
// public-storefront API (see application/integrations.
// ResolveStorefrontKeyUseCase). Tenant-wide, not tied to any one
// integration connection: a tenant's storefront calls aren't scoped to a
// specific connection, and a tenant can have several (see
// domain/integrations.ConnectionRepository.ListByTenant). Re-running this
// rotates the key -- the old value stops resolving immediately, so the
// merchant must update the storefront's configuration with the new one.
// Stores a hash instead of an encrypted-and-reversible secret: a
// storefront key is only ever compared, never displayed back to the
// merchant after generation.
type SetStorefrontKeyUseCase struct {
	db           *gorm.DB
	settingsRepo storefrontKeyHashSetter
}

func NewSetStorefrontKeyUseCase(db *gorm.DB, settingsRepo storefrontKeyHashSetter) *SetStorefrontKeyUseCase {
	return &SetStorefrontKeyUseCase{db: db, settingsRepo: settingsRepo}
}

func (uc *SetStorefrontKeyUseCase) Execute(ctx context.Context) (*SetStorefrontKeyResult, error) {
	plaintext, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("setStorefrontKey: generate: %w", err)
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.settingsRepo.SetStorefrontKeyHash(ctx, hashStorefrontKey(plaintext))
	})
	if err != nil {
		return nil, fmt.Errorf("setStorefrontKey: %w", err)
	}
	return &SetStorefrontKeyResult{PlaintextKey: plaintext}, nil
}

// hashStorefrontKey is the one hashing function both SetStorefrontKeyUseCase
// (writes it) and ResolveStorefrontKeyUseCase (looks it up by it) use --
// must stay identical between the two or every key silently stops working.
func hashStorefrontKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
