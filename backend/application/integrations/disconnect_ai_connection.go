package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DisconnectAIConnectionUseCase removes the tenant's AI connection --
// idempotent, since "already disconnected" is not an error (no ID to look
// up first, unlike DisconnectConnectionUseCase's per-row WooCommerce case).
type DisconnectAIConnectionUseCase struct {
	db         *gorm.DB
	aiConnRepo integrationsDomain.AIConnectionRepository
	outbox     eventrecord.Enqueuer
}

func NewDisconnectAIConnectionUseCase(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository, outbox eventrecord.Enqueuer) *DisconnectAIConnectionUseCase {
	return &DisconnectAIConnectionUseCase{db: db, aiConnRepo: aiConnRepo, outbox: outbox}
}

func (uc *DisconnectAIConnectionUseCase) Execute(ctx context.Context) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.aiConnRepo.Get(ctx)
		if err != nil {
			return err
		}
		if err := uc.aiConnRepo.Delete(ctx); err != nil {
			return err
		}
		if existing == nil {
			return nil // already disconnected: nothing happened
		}
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionDisconnected, existing.ID)
	})
	if err != nil {
		return fmt.Errorf("disconnectAIConnection: %w", err)
	}
	return nil
}
