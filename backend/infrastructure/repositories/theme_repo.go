package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

func themeTx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("theme: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

type themeModel struct {
	ID                 string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           string `gorm:"type:uuid"`
	SalesChannelID     string `gorm:"type:uuid"`
	Name               string
	PresetKey          string
	PublishedVersionID *string `gorm:"type:uuid"`
	IsLive             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (themeModel) TableName() string { return "themes" }

func (m themeModel) toDomain() theme.Theme {
	return theme.Theme{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, Name: m.Name, PresetKey: m.PresetKey,
		PublishedVersionID: m.PublishedVersionID, IsLive: m.IsLive, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type ThemeRepo struct{ db *gorm.DB }

func NewThemeRepo(db *gorm.DB) *ThemeRepo { return &ThemeRepo{db: db} }

var _ theme.ThemeRepository = (*ThemeRepo)(nil)

func (r *ThemeRepo) Create(ctx context.Context, t *theme.Theme) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := themeModel{
		TenantID: t.TenantID, SalesChannelID: t.SalesChannelID, Name: t.Name, PresetKey: t.PresetKey,
		PublishedVersionID: t.PublishedVersionID, IsLive: t.IsLive, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("themeRepo.Create: %w", err)
	}
	t.ID = m.ID
	return nil
}

func (r *ThemeRepo) FindByID(ctx context.Context, id string) (*theme.Theme, error) {
	return r.find(ctx, id, false)
}

func (r *ThemeRepo) FindByIDForUpdate(ctx context.Context, id string) (*theme.Theme, error) {
	return r.find(ctx, id, true)
}

func (r *ThemeRepo) find(ctx context.Context, id string, forUpdate bool) (*theme.Theme, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	if forUpdate {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var m themeModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, theme.ErrThemeNotFound
		}
		return nil, fmt.Errorf("themeRepo.find: %w", err)
	}
	t := m.toDomain()
	return &t, nil
}

func (r *ThemeRepo) FindLiveByChannel(ctx context.Context, salesChannelID string) (*theme.Theme, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []themeModel
	if err := tx.Where("sales_channel_id = ? AND is_live", salesChannelID).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("themeRepo.FindLiveByChannel: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	t := rows[0].toDomain()
	return &t, nil
}

func (r *ThemeRepo) ListByChannel(ctx context.Context, salesChannelID string, limit, offset int) ([]theme.Theme, int64, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&themeModel{}).Where("sales_channel_id = ?", salesChannelID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("themeRepo.ListByChannel count: %w", err)
	}
	var rows []themeModel
	if err := q.Order("is_live DESC, updated_at DESC, id").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("themeRepo.ListByChannel: %w", err)
	}
	out := make([]theme.Theme, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, total, nil
}

func (r *ThemeRepo) Update(ctx context.Context, t *theme.Theme) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&themeModel{}).Where("id = ?", t.ID).Updates(map[string]any{
		"name": t.Name, "published_version_id": t.PublishedVersionID, "is_live": t.IsLive, "updated_at": t.UpdatedAt,
	}).Error; err != nil {
		return fmt.Errorf("themeRepo.Update: %w", err)
	}
	return nil
}

func (r *ThemeRepo) ClearLive(ctx context.Context, salesChannelID, exceptThemeID string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&themeModel{}).
		Where("sales_channel_id = ? AND id <> ? AND is_live", salesChannelID, exceptThemeID).
		Update("is_live", false).Error; err != nil {
		return fmt.Errorf("themeRepo.ClearLive: %w", err)
	}
	return nil
}

func (r *ThemeRepo) Delete(ctx context.Context, id string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&themeModel{}).Error; err != nil {
		return fmt.Errorf("themeRepo.Delete: %w", err)
	}
	return nil
}

// ── Sessions ────────────────────────────────────────────────────────────────

type themeSessionModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"type:uuid"`
	ThemeID   string `gorm:"type:uuid"`
	UserID    string
	Role      string
	Purpose   string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (themeSessionModel) TableName() string { return "theme_sessions" }

type ThemeSessionRepo struct{ db *gorm.DB }

func NewThemeSessionRepo(db *gorm.DB) *ThemeSessionRepo { return &ThemeSessionRepo{db: db} }

var _ theme.SessionRepository = (*ThemeSessionRepo)(nil)

func (r *ThemeSessionRepo) Create(ctx context.Context, s *theme.Session) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := themeSessionModel{
		TenantID: s.TenantID, ThemeID: s.ThemeID, UserID: s.UserID, Role: s.Role, Purpose: string(s.Purpose),
		TokenHash: s.TokenHash, ExpiresAt: s.ExpiresAt, RevokedAt: s.RevokedAt, CreatedAt: s.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("themeSessionRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *ThemeSessionRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*theme.Session, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []themeSessionModel
	if err := tx.Where("token_hash = ?", tokenHash).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("themeSessionRepo.FindByTokenHash: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	m := rows[0]
	return &theme.Session{
		ID: m.ID, TenantID: m.TenantID, ThemeID: m.ThemeID, UserID: m.UserID, Role: m.Role,
		Purpose: theme.SessionPurpose(m.Purpose), TokenHash: m.TokenHash, ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt, CreatedAt: m.CreatedAt,
	}, nil
}
