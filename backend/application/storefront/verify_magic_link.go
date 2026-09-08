package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const customerSessionTTL = 30 * 24 * time.Hour

type VerifyMagicLinkUseCase struct {
	db          *gorm.DB
	tokenRepo   storefrontDomain.MagicLinkTokenRepository
	sessionRepo storefrontDomain.CustomerSessionRepository
}

func NewVerifyMagicLinkUseCase(db *gorm.DB, tokenRepo storefrontDomain.MagicLinkTokenRepository, sessionRepo storefrontDomain.CustomerSessionRepository) *VerifyMagicLinkUseCase {
	return &VerifyMagicLinkUseCase{db: db, tokenRepo: tokenRepo, sessionRepo: sessionRepo}
}

// Execute exchanges a single-use magic-link token for a long-lived
// CustomerSession -- salesChannelID must match the token's own (a link
// emailed for channel A can't mint a session on channel B), enforced with
// the same collapsed ErrInvalidOrExpiredMagicLink as "not found"/"expired"/
// "already used" so none of those reasons is distinguishable from another.
func (uc *VerifyMagicLinkUseCase) Execute(ctx context.Context, salesChannelID, token string) (*storefrontDomain.CustomerSession, error) {
	var session *storefrontDomain.CustomerSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.tokenRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		now := time.Now()
		if found == nil || found.SalesChannelID != salesChannelID || !found.IsRedeemable(now) {
			return storefrontDomain.ErrInvalidOrExpiredMagicLink
		}
		if err := uc.tokenRepo.MarkUsed(ctx, found.ID, now); err != nil {
			return err
		}
		session = &storefrontDomain.CustomerSession{
			TenantID: found.TenantID, SalesChannelID: found.SalesChannelID, CustomerID: found.CustomerID,
			Token: idgen.NewV4(), ExpiresAt: now.Add(customerSessionTTL), CreatedAt: now,
		}
		return uc.sessionRepo.Create(ctx, session)
	})
	if err != nil {
		return nil, fmt.Errorf("verifyMagicLink: %w", err)
	}
	return session, nil
}
