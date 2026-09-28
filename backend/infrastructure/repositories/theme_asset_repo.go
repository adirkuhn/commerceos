package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/theme"
)

type themeAssetModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"type:uuid"`
	URL         string `gorm:"column:url"`
	Filename    string
	ContentType string
	Bytes       int64
	Width       int
	Height      int
	Alt         string
	CreatedBy   string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

func (themeAssetModel) TableName() string { return "theme_assets" }

func (m themeAssetModel) toDomain() theme.Asset {
	return theme.Asset{
		ID: m.ID, TenantID: m.TenantID, URL: m.URL, Filename: m.Filename, ContentType: m.ContentType, Bytes: m.Bytes,
		Width: m.Width, Height: m.Height, Alt: m.Alt, CreatedBy: m.CreatedBy, CreatedAt: m.CreatedAt, DeletedAt: m.DeletedAt,
	}
}

type ThemeAssetRepo struct{ db *gorm.DB }

func NewThemeAssetRepo(db *gorm.DB) *ThemeAssetRepo { return &ThemeAssetRepo{db: db} }

var _ theme.AssetRepository = (*ThemeAssetRepo)(nil)

func (r *ThemeAssetRepo) Create(ctx context.Context, a *theme.Asset) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := themeAssetModel{
		TenantID: a.TenantID, URL: a.URL, Filename: a.Filename, ContentType: a.ContentType, Bytes: a.Bytes,
		Width: a.Width, Height: a.Height, Alt: a.Alt, CreatedBy: a.CreatedBy, CreatedAt: a.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("themeAssetRepo.Create: %w", err)
	}
	a.ID = m.ID
	return nil
}

func (r *ThemeAssetRepo) FindByID(ctx context.Context, id string) (*theme.Asset, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var m themeAssetModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, theme.ErrAssetNotFound
		}
		return nil, fmt.Errorf("themeAssetRepo.FindByID: %w", err)
	}
	a := m.toDomain()
	return &a, nil
}

func (r *ThemeAssetRepo) ListActive(ctx context.Context, limit, offset int) ([]theme.Asset, int64, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&themeAssetModel{}).Where("deleted_at IS NULL")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("themeAssetRepo.ListActive count: %w", err)
	}
	var rows []themeAssetModel
	if err := q.Order("created_at DESC, id").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("themeAssetRepo.ListActive: %w", err)
	}
	out := make([]theme.Asset, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, total, nil
}

func (r *ThemeAssetRepo) FindByIDs(ctx context.Context, ids []string) ([]theme.Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []themeAssetModel
	if err := tx.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("themeAssetRepo.FindByIDs: %w", err)
	}
	out := make([]theme.Asset, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, nil
}

func (r *ThemeAssetRepo) Update(ctx context.Context, a *theme.Asset) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&themeAssetModel{}).Where("id = ?", a.ID).
		Updates(map[string]any{"alt": a.Alt, "deleted_at": a.DeletedAt}).Error; err != nil {
		return fmt.Errorf("themeAssetRepo.Update: %w", err)
	}
	return nil
}
