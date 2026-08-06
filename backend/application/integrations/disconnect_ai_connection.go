package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DisconnectAIConnectionUseCase removes the tenant's AI connection --
// idempotent, since "already disconnected" is not an error (no ID to look
// up first, unlike DisconnectConnectionUseCase's per-row WooCommerce case).
type DisconnectAIConnectionUseCase struct {
	db         *gorm.DB
	aiConnRepo integrationsDomain.AIConnectionRepository
}

func NewDisconnectAIConnectionUseCase(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository) *DisconnectAIConnectionUseCase {
	return &DisconnectAIConnectionUseCase{db: db, aiConnRepo: aiConnRepo}
}

func (uc *DisconnectAIConnectionUseCase) Execute(ctx context.Context) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.aiConnRepo.Delete(ctx)
	})
	if err != nil {
		return fmt.Errorf("disconnectAIConnection: %w", err)
	}
	return nil
}
