package apikeys

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type RevokeAPIKeyUseCase struct {
	db     *gorm.DB
	repo   apikeysDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewRevokeAPIKeyUseCase(db *gorm.DB, repo apikeysDomain.Repository, outbox eventrecord.Enqueuer) *RevokeAPIKeyUseCase {
	return &RevokeAPIKeyUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *RevokeAPIKeyUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Revoke(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.APIKeyRevoked, id)
	})
	if err != nil {
		return fmt.Errorf("revokeApiKey: %w", err)
	}
	return nil
}
