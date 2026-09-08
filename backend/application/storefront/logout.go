package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type LogoutUseCase struct {
	db          *gorm.DB
	sessionRepo storefrontDomain.CustomerSessionRepository
}

func NewLogoutUseCase(db *gorm.DB, sessionRepo storefrontDomain.CustomerSessionRepository) *LogoutUseCase {
	return &LogoutUseCase{db: db, sessionRepo: sessionRepo}
}

// Execute deletes the session outright -- logout has nothing to keep
// around (unlike a checked-out cart, there's no later state a deleted
// session needs to still represent), and a delete of an already-gone
// token is a no-op success, not an error: the shopper's goal ("I am now
// signed out") is already true either way.
func (uc *LogoutUseCase) Execute(ctx context.Context, sessionToken string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.sessionRepo.Delete(ctx, sessionToken)
	})
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}
