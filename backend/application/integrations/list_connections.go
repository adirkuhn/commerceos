package integrations

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListConnectionsUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewListConnectionsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *ListConnectionsUseCase {
	return &ListConnectionsUseCase{db: db, connRepo: connRepo}
}

func (uc *ListConnectionsUseCase) Execute(ctx context.Context) ([]integrationsDomain.Connection, error) {
	var conns []integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		conns = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listConnections: %w", err)
	}
	return conns, nil
}
