package integrations

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DisconnectConnectionUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewDisconnectConnectionUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *DisconnectConnectionUseCase {
	return &DisconnectConnectionUseCase{db: db, connRepo: connRepo}
}

func (uc *DisconnectConnectionUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.connRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.connRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("disconnectConnection: %w", err)
	}
	return nil
}
