package apikeys

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListAPIKeysUseCase struct {
	db   *gorm.DB
	repo apikeysDomain.Repository
}

func NewListAPIKeysUseCase(db *gorm.DB, repo apikeysDomain.Repository) *ListAPIKeysUseCase {
	return &ListAPIKeysUseCase{db: db, repo: repo}
}

func (uc *ListAPIKeysUseCase) Execute(ctx context.Context) ([]apikeysDomain.APIKey, error) {
	var out []apikeysDomain.APIKey
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.repo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listApiKeys: %w", err)
	}
	return out, nil
}
