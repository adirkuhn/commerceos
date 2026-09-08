package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// themeBlocks is JSONB-backed, same driver.Valuer/sql.Scanner + whole-
// document-replace convention as whatsapp_conversation_repo.go's own
// cartItems type.
type themeBlocks []themeDomain.Block

func (b themeBlocks) Value() (driver.Value, error) {
	if len(b) == 0 {
		return "[]", nil
	}
	out, err := json.Marshal([]themeDomain.Block(b))
	return string(out), err
}

func (b *themeBlocks) Scan(value any) error {
	if value == nil {
		*b = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("themeBlocks.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(raw, (*[]themeDomain.Block)(b))
}

type themePageModel struct {
	ID             string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string      `gorm:"not null;type:uuid"`
	SalesChannelID string      `gorm:"column:sales_channel_id;not null;type:uuid"`
	Slug           string      `gorm:"not null"`
	CanvasWidth    int         `gorm:"column:canvas_width;not null"`
	Blocks         themeBlocks `gorm:"type:jsonb;not null;default:'[]'"`
	CustomCSS      string      `gorm:"column:custom_css;not null;default:''"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (themePageModel) TableName() string { return "theme_pages" }

type ThemePageRepo struct{ db *gorm.DB }

func NewThemePageRepo(db *gorm.DB) *ThemePageRepo { return &ThemePageRepo{db: db} }

func (r *ThemePageRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("themePageRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ThemePageRepo) FindBySlug(ctx context.Context, salesChannelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m themePageModel
	err = tx.Where("sales_channel_id = ? AND slug = ?", salesChannelID, string(slug)).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("themePageRepo.FindBySlug: %w", err)
	}
	return toThemePageDomain(m), nil
}

// Upsert inserts a new row, or replaces Blocks/CanvasWidth/UpdatedAt
// wholesale when (sales_channel_id, slug) already exists -- one page per
// channel per slug, same "at most one row for this natural key" pattern as
// channels.SalesChannelProductOverrideRepository.Upsert. Deliberately
// doesn't try to populate p.ID/CreatedAt back from the write (Postgres's
// plain ON CONFLICT DO UPDATE doesn't reliably hand those back the way a
// fresh INSERT does) -- same convention as that override repo's own
// Upsert; a caller needing the full row back calls FindBySlug afterward
// (see application/theme.SaveThemePageUseCase).
func (r *ThemePageRepo) Upsert(ctx context.Context, p *themeDomain.ThemePage) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "sales_channel_id"}, {Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"canvas_width", "blocks", "custom_css", "updated_at"}),
	}).Create(&themePageModel{
		TenantID: p.TenantID, SalesChannelID: p.SalesChannelID, Slug: string(p.Slug),
		CanvasWidth: p.CanvasWidth, Blocks: themeBlocks(p.Blocks), CustomCSS: p.CustomCSS,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("themePageRepo.Upsert: %w", err)
	}
	return nil
}

func toThemePageDomain(m themePageModel) *themeDomain.ThemePage {
	return &themeDomain.ThemePage{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, Slug: themeDomain.PageSlug(m.Slug),
		CanvasWidth: m.CanvasWidth, Blocks: []themeDomain.Block(m.Blocks), CustomCSS: m.CustomCSS,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
