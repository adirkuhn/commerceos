package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetAIConnectionUseCase returns the tenant's configured AI provider, or nil
// (not an error) when none is set up yet -- matches
// AIConnectionRepository.Get's contract.
type GetAIConnectionUseCase struct {
	db         *gorm.DB
	aiConnRepo integrationsDomain.AIConnectionRepository
}

func NewGetAIConnectionUseCase(db *gorm.DB, aiConnRepo integrationsDomain.AIConnectionRepository) *GetAIConnectionUseCase {
	return &GetAIConnectionUseCase{db: db, aiConnRepo: aiConnRepo}
}

func (uc *GetAIConnectionUseCase) Execute(ctx context.Context) (*integrationsDomain.AIConnection, error) {
	var conn *integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.aiConnRepo.Get(ctx)
		if err != nil {
			return err
		}
		conn = c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getAIConnection: %w", err)
	}
	return conn, nil
}
