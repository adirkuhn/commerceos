package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SetStorefrontKeyResult carries the PLAINTEXT key back to the caller
// exactly once, at generation time -- only its hash persists
// (Connection.StorefrontKeyHash), so the merchant must paste it into the
// storefront's (e.g. the WooCommerce plugin's) settings right away.
type SetStorefrontKeyResult struct {
	ConnectionID string
	PlaintextKey string
}

// SetStorefrontKeyUseCase (re)generates the per-connection key a storefront
// presents as a bearer token to call the authenticated public-storefront
// API (see application/integrations.ResolveStorefrontKeyUseCase). Re-running
// it rotates the key -- the old value stops resolving immediately, so the
// merchant must update the storefront's configuration with the new one.
// Mirrors SetWebhookSecretUseCase, but stores a hash instead of an
// encrypted-and-reversible secret: a storefront key is only ever compared,
// never displayed back to the merchant after generation.
type SetStorefrontKeyUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewSetStorefrontKeyUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *SetStorefrontKeyUseCase {
	return &SetStorefrontKeyUseCase{db: db, connRepo: connRepo}
}

func (uc *SetStorefrontKeyUseCase) Execute(ctx context.Context, connectionID string) (*SetStorefrontKeyResult, error) {
	plaintext, err := generateSecret()
	if err != nil {
		return nil, fmt.Errorf("setStorefrontKey: generate: %w", err)
	}

	var result SetStorefrontKeyResult
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		existing.StorefrontKeyHash = hashStorefrontKey(plaintext)
		existing.UpdatedAt = time.Now()
		if err := uc.connRepo.Update(ctx, existing); err != nil {
			return err
		}
		result = SetStorefrontKeyResult{ConnectionID: existing.ID, PlaintextKey: plaintext}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("setStorefrontKey: %w", err)
	}
	return &result, nil
}

// hashStorefrontKey is the one hashing function both SetStorefrontKeyUseCase
// (writes it) and ResolveStorefrontKeyUseCase (looks it up by it) use --
// must stay identical between the two or every key silently stops working.
func hashStorefrontKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
