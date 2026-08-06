package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListSalesChannelsUseCase struct {
	db   *gorm.DB
	repo channelsDomain.SalesChannelRepository
}

func NewListSalesChannelsUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository) *ListSalesChannelsUseCase {
	return &ListSalesChannelsUseCase{db: db, repo: repo}
}

func (uc *ListSalesChannelsUseCase) Execute(ctx context.Context) ([]channelsDomain.SalesChannel, error) {
	var channels []channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		channels, err = uc.repo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSalesChannels: %w", err)
	}
	return channels, nil
}
