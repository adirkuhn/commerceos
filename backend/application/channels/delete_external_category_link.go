package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteExternalCategoryLinkUseCase struct {
	db     *gorm.DB
	repo   channelsDomain.ExternalCategoryLinkRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteExternalCategoryLinkUseCase(db *gorm.DB, repo channelsDomain.ExternalCategoryLinkRepository, outbox eventrecord.Enqueuer) *DeleteExternalCategoryLinkUseCase {
	return &DeleteExternalCategoryLinkUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteExternalCategoryLinkUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ExternalCategoryLinkDeleted, id)
	})
	if err != nil {
		return fmt.Errorf("deleteExternalCategoryLink: %w", err)
	}
	return nil
}
