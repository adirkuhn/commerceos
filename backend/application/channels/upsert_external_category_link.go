package channels

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpsertExternalCategoryLinkInput struct {
	ConnectionID string
	ExternalID   string
	CategoryID   string
}

// UpsertExternalCategoryLinkUseCase is the manual counterpart to
// ResolveCategoryLinkUseCase's auto-match-on-push -- lets a user create or
// repoint a link directly from the Settings UI, for a category whose slug
// doesn't happen to match any Woo category name.
type UpsertExternalCategoryLinkUseCase struct {
	db           *gorm.DB
	linkRepo     channelsDomain.ExternalCategoryLinkRepository
	categoryRepo pmsDomain.CategoryRepository
	outbox       eventrecord.Enqueuer
}

func NewUpsertExternalCategoryLinkUseCase(db *gorm.DB, linkRepo channelsDomain.ExternalCategoryLinkRepository, categoryRepo pmsDomain.CategoryRepository, outbox eventrecord.Enqueuer) *UpsertExternalCategoryLinkUseCase {
	return &UpsertExternalCategoryLinkUseCase{db: db, linkRepo: linkRepo, categoryRepo: categoryRepo, outbox: outbox}
}

func (uc *UpsertExternalCategoryLinkUseCase) Execute(ctx context.Context, in UpsertExternalCategoryLinkInput) (*channelsDomain.ExternalCategoryLink, error) {
	var link channelsDomain.ExternalCategoryLink
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.categoryRepo.FindByID(ctx, in.CategoryID); err != nil {
			return err
		}
		now := time.Now()
		link = channelsDomain.ExternalCategoryLink{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: in.ConnectionID,
			Provider:   string(integrationsDomain.ProviderWooCommerce),
			ExternalID: in.ExternalID, CategoryID: in.CategoryID,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.linkRepo.Upsert(ctx, &link); err != nil {
			return err
		}
		return recordCategoryLinkSaved(ctx, uc.outbox, link, CategoryLinkSourceManual)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertExternalCategoryLink: %w", err)
	}
	return &link, nil
}
