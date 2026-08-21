package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteExternalCategoryLinkUseCase struct {
	db   *gorm.DB
	repo channelsDomain.ExternalCategoryLinkRepository
}

func NewDeleteExternalCategoryLinkUseCase(db *gorm.DB, repo channelsDomain.ExternalCategoryLinkRepository) *DeleteExternalCategoryLinkUseCase {
	return &DeleteExternalCategoryLinkUseCase{db: db, repo: repo}
}

func (uc *DeleteExternalCategoryLinkUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteExternalCategoryLink: %w", err)
	}
	return nil
}
