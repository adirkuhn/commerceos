package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// categoryLinkResolver is the one method PushProductToWooUseCase/
// SyncProductToChannelsUseCase need from ResolveCategoryLinkUseCase (ISP).
// client is passed in rather than built internally -- the caller already
// has one open for this connection's push, so this avoids a second
// decrypt+HTTP-client-construction per product.
type categoryLinkResolver interface {
	Execute(ctx context.Context, client integrationsDomain.WooClient, connectionID, categoryID string) (string, error)
}

// ResolveCategoryLinkUseCase answers "what's this category's id on this Woo
// connection", so the push paths can set WooProduct.Categories instead of
// every pushed product landing in Woo's "Uncategorized". Three-step
// resolution, cheapest first:
//
//  1. An existing ExternalCategoryLink for this (connection, category) --
//     the common case once a category has been seen before.
//  2. A live slug match against the connection's Woo category list --
//     the same matching rule ImportWooProductsUseCase already uses in the
//     opposite direction. A match found this way is persisted as a link
//     so later pushes skip straight to step 1.
//  3. No match: "" (Woo shows the product Uncategorized, exactly today's
//     behavior), left for manual mapping via the Settings UI -- this use
//     case never creates a new category on Woo's side, and a store that's
//     temporarily unreachable degrades to "" rather than failing the
//     whole product push.
//
// The Woo API call (step 2) deliberately happens outside any
// tenant.WithTenant block -- external I/O has no place inside a DB
// transaction; each DB read/write here gets its own short transaction.
type ResolveCategoryLinkUseCase struct {
	db           *gorm.DB
	linkRepo     channelsDomain.ExternalCategoryLinkRepository
	categoryRepo pmsDomain.CategoryRepository
}

func NewResolveCategoryLinkUseCase(db *gorm.DB, linkRepo channelsDomain.ExternalCategoryLinkRepository, categoryRepo pmsDomain.CategoryRepository) *ResolveCategoryLinkUseCase {
	return &ResolveCategoryLinkUseCase{db: db, linkRepo: linkRepo, categoryRepo: categoryRepo}
}

func (uc *ResolveCategoryLinkUseCase) Execute(ctx context.Context, client integrationsDomain.WooClient, connectionID, categoryID string) (string, error) {
	var link *channelsDomain.ExternalCategoryLink
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		link, err = uc.linkRepo.FindByConnectionAndCategory(ctx, connectionID, categoryID)
		if err != nil && !errors.Is(err, channelsDomain.ErrExternalCategoryLinkNotFound) {
			return err
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("resolveCategoryLink: %w", err)
	}
	if link != nil {
		return link.ExternalID, nil
	}

	var category *pmsDomain.Category
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		category, err = uc.categoryRepo.FindByID(ctx, categoryID)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("resolveCategoryLink: %w", err)
	}

	wooCategories, err := client.ListCategories(ctx)
	if err != nil {
		return "", nil
	}

	var matchedExternalID string
	for _, wc := range wooCategories {
		if wc.Slug == category.Slug {
			matchedExternalID = wc.ExternalID
			break
		}
	}
	if matchedExternalID == "" {
		return "", nil
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		newLink := &channelsDomain.ExternalCategoryLink{
			TenantID: tenant.IDFromContext(ctx), ConnectionID: connectionID,
			Provider:   string(integrationsDomain.ProviderWooCommerce),
			ExternalID: matchedExternalID, CategoryID: categoryID,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.linkRepo.Upsert(ctx, newLink)
	})
	if err != nil {
		return "", fmt.Errorf("resolveCategoryLink: %w", err)
	}
	return matchedExternalID, nil
}
