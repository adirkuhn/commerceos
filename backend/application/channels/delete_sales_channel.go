package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeleteSalesChannelUseCase only removes the channel's enrollment
// config -- it never touches ProductChannelListing rows, so any product
// already pushed to the channel's connection stays listed there (see
// SalesChannel's doc comment: this module never auto-delists).
type DeleteSalesChannelUseCase struct {
	db   *gorm.DB
	repo channelsDomain.SalesChannelRepository
}

func NewDeleteSalesChannelUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository) *DeleteSalesChannelUseCase {
	return &DeleteSalesChannelUseCase{db: db, repo: repo}
}

func (uc *DeleteSalesChannelUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteSalesChannel: %w", err)
	}
	return nil
}
