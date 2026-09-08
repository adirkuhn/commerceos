package theme

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type SaveThemePageInput struct {
	SalesChannelID string
	Slug           themeDomain.PageSlug
	CanvasWidth    int
	Blocks         []themeDomain.Block
}

type SaveThemePageUseCase struct {
	db          *gorm.DB
	repo        themeDomain.Repository
	channelRepo channelsDomain.SalesChannelRepository
}

func NewSaveThemePageUseCase(db *gorm.DB, repo themeDomain.Repository, channelRepo channelsDomain.SalesChannelRepository) *SaveThemePageUseCase {
	return &SaveThemePageUseCase{db: db, repo: repo, channelRepo: channelRepo}
}

// Execute replaces salesChannelID's whole page document for in.Slug --
// rejects an unknown PageSlug or BlockType up front (never silently drops
// a block the storefront-frontend wouldn't know how to render), and
// returns the freshly-saved page (re-fetched after the write -- see
// ThemePageRepository.Upsert's own doc comment on why it doesn't hand rows
// back itself).
func (uc *SaveThemePageUseCase) Execute(ctx context.Context, in SaveThemePageInput) (*themeDomain.ThemePage, error) {
	if !in.Slug.Valid() {
		return nil, fmt.Errorf("saveThemePage: %w", themeDomain.ErrInvalidPageSlug)
	}
	for _, b := range in.Blocks {
		if !b.Type.Valid() {
			return nil, fmt.Errorf("saveThemePage: %w", themeDomain.ErrInvalidBlockType)
		}
	}

	var saved *themeDomain.ThemePage
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.channelRepo.FindByID(ctx, in.SalesChannelID); err != nil {
			return err
		}
		now := time.Now()
		page := themeDomain.ThemePage{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: in.SalesChannelID, Slug: in.Slug,
			CanvasWidth: in.CanvasWidth, Blocks: in.Blocks, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Upsert(ctx, &page); err != nil {
			return err
		}
		found, err := uc.repo.FindBySlug(ctx, in.SalesChannelID, in.Slug)
		if err != nil {
			return err
		}
		saved = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("saveThemePage: %w", err)
	}
	return saved, nil
}
