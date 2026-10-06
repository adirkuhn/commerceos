package channels

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeleteSalesChannelUseCase only removes the channel's enrollment
// config -- it never touches ProductChannelListing rows, so any product
// already pushed to the channel's connection stays listed there (see
// SalesChannel's doc comment: this module never auto-delists).
type DeleteSalesChannelUseCase struct {
	db     *gorm.DB
	repo   channelsDomain.SalesChannelRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteSalesChannelUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository, outbox eventrecord.Enqueuer) *DeleteSalesChannelUseCase {
	return &DeleteSalesChannelUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteSalesChannelUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SalesChannelDeleted, id, eventrecord.Data{
			"name": existing.Name, "type": string(existing.Type),
		})
	})
	if err != nil {
		return fmt.Errorf("deleteSalesChannel: %w", err)
	}
	return nil
}
