package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListExternalCategoryLinksUseCase struct {
	db   *gorm.DB
	repo channelsDomain.ExternalCategoryLinkRepository
}

func NewListExternalCategoryLinksUseCase(db *gorm.DB, repo channelsDomain.ExternalCategoryLinkRepository) *ListExternalCategoryLinksUseCase {
	return &ListExternalCategoryLinksUseCase{db: db, repo: repo}
}

func (uc *ListExternalCategoryLinksUseCase) Execute(ctx context.Context, connectionID string) ([]channelsDomain.ExternalCategoryLink, error) {
	var links []channelsDomain.ExternalCategoryLink
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		links, err = uc.repo.ListByConnection(ctx, connectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listExternalCategoryLinks: %w", err)
	}
	return links, nil
}
