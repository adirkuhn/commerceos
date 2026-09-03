package apikeys

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type RevokeAPIKeyUseCase struct {
	db   *gorm.DB
	repo apikeysDomain.Repository
}

func NewRevokeAPIKeyUseCase(db *gorm.DB, repo apikeysDomain.Repository) *RevokeAPIKeyUseCase {
	return &RevokeAPIKeyUseCase{db: db, repo: repo}
}

func (uc *RevokeAPIKeyUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Revoke(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("revokeApiKey: %w", err)
	}
	return nil
}
