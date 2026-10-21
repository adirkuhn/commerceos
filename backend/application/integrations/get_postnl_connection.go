package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetPostNLConnectionUseCase loads a PostNL connection for its settings
// page; ErrNotPostNLConnection for any other provider.
type GetPostNLConnectionUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
}

func NewGetPostNLConnectionUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository) *GetPostNLConnectionUseCase {
	return &GetPostNLConnectionUseCase{db: db, connRepo: connRepo}
}

func (uc *GetPostNLConnectionUseCase) Execute(ctx context.Context, connectionID string) (*integrationsDomain.Connection, error) {
	var conn *integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if c.Provider != integrationsDomain.ProviderPostNL {
			return integrationsDomain.ErrNotPostNLConnection
		}
		conn = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPostNLConnection: %w", err)
	}
	return conn, nil
}
