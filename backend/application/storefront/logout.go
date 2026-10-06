package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type LogoutUseCase struct {
	db          *gorm.DB
	sessionRepo storefrontDomain.CustomerSessionRepository
	outbox      eventrecord.Enqueuer
}

func NewLogoutUseCase(db *gorm.DB, sessionRepo storefrontDomain.CustomerSessionRepository, outbox eventrecord.Enqueuer) *LogoutUseCase {
	return &LogoutUseCase{db: db, sessionRepo: sessionRepo, outbox: outbox}
}

// Execute deletes the session outright -- logout has nothing to keep
// around (unlike a checked-out cart, there's no later state a deleted
// session needs to still represent), and a delete of an already-gone
// token is a no-op success, not an error: the shopper's goal ("I am now
// signed out") is already true either way.
func (uc *LogoutUseCase) Execute(ctx context.Context, sessionToken string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByToken(ctx, sessionToken)
		if err != nil {
			return err
		}
		if err := uc.sessionRepo.Delete(ctx, sessionToken); err != nil {
			return err
		}
		if session == nil { // already signed out: nothing happened
			return nil
		}
		return eventrecord.Record(ctx, uc.outbox, events.CustomerSessionEnded, session.ID, eventrecord.Data{
			"sales_channel_id": session.SalesChannelID, "customer_id": session.CustomerID,
		})
	})
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}
