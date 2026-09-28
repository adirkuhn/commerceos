package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/theme"
)

type themeVersionModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"type:uuid"`
	ThemeID   string `gorm:"type:uuid"`
	Number    int
	State     string
	Revision  int
	Settings  jsonb[themeSettingsJSON] `gorm:"type:jsonb"`
	Note      string
	CreatedBy string
	CreatedAt time.Time
	SealedAt  *time.Time
}

func (themeVersionModel) TableName() string { return "theme_versions" }

func (m themeVersionModel) toDomain() theme.Version {
	return theme.Version{
		ID: m.ID, TenantID: m.TenantID, ThemeID: m.ThemeID, Number: m.Number, State: theme.VersionState(m.State),
		Revision: m.Revision, Settings: m.Settings.V.toDomain(), Note: m.Note, CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt, SealedAt: m.SealedAt,
	}
}

type ThemeVersionRepo struct{ db *gorm.DB }

func NewThemeVersionRepo(db *gorm.DB) *ThemeVersionRepo { return &ThemeVersionRepo{db: db} }

var _ theme.VersionRepository = (*ThemeVersionRepo)(nil)

func (r *ThemeVersionRepo) Create(ctx context.Context, v *theme.Version) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := themeVersionModel{
		TenantID: v.TenantID, ThemeID: v.ThemeID, Number: v.Number, State: string(v.State), Revision: v.Revision,
		Settings: jsonb[themeSettingsJSON]{V: toSettingsJSON(v.Settings)}, Note: v.Note, CreatedBy: v.CreatedBy,
		CreatedAt: v.CreatedAt, SealedAt: v.SealedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("themeVersionRepo.Create: %w", err)
	}
	v.ID = m.ID
	return nil
}

func (r *ThemeVersionRepo) FindByID(ctx context.Context, id string) (*theme.Version, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var m themeVersionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, theme.ErrVersionNotFound
		}
		return nil, fmt.Errorf("themeVersionRepo.FindByID: %w", err)
	}
	v := m.toDomain()
	return &v, nil
}

func (r *ThemeVersionRepo) FindDraft(ctx context.Context, themeID string, forUpdate bool) (*theme.Version, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	if forUpdate {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var rows []themeVersionModel
	if err := tx.Where("theme_id = ? AND state = ?", themeID, theme.VersionStateDraft).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("themeVersionRepo.FindDraft: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	v := rows[0].toDomain()
	return &v, nil
}

func (r *ThemeVersionRepo) NextNumber(ctx context.Context, themeID string) (int, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return 0, err
	}
	var max *int
	if err := tx.Model(&themeVersionModel{}).Where("theme_id = ?", themeID).Select("MAX(number)").Scan(&max).Error; err != nil {
		return 0, fmt.Errorf("themeVersionRepo.NextNumber: %w", err)
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

func (r *ThemeVersionRepo) Update(ctx context.Context, v *theme.Version) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&themeVersionModel{}).Where("id = ?", v.ID).Updates(map[string]any{
		"state": string(v.State), "revision": v.Revision, "note": v.Note, "sealed_at": v.SealedAt,
		"settings": jsonb[themeSettingsJSON]{V: toSettingsJSON(v.Settings)},
	}).Error; err != nil {
		return fmt.Errorf("themeVersionRepo.Update: %w", err)
	}
	return nil
}

func (r *ThemeVersionRepo) ListByTheme(ctx context.Context, themeID string, limit, offset int) ([]theme.Version, int64, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&themeVersionModel{}).Where("theme_id = ?", themeID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("themeVersionRepo.ListByTheme count: %w", err)
	}
	var rows []themeVersionModel
	if err := q.Order("number DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("themeVersionRepo.ListByTheme: %w", err)
	}
	out := make([]theme.Version, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, total, nil
}

// PruneSealed: see theme.VersionRepository. Documents go with their
// version (ON DELETE CASCADE); the sealed-immutability triggers only
// guard UPDATE, so dropping a whole old version is allowed.
func (r *ThemeVersionRepo) PruneSealed(ctx context.Context, themeID string, keep int, protectID string) (int64, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return 0, err
	}
	res := tx.Exec(`
		DELETE FROM theme_versions
		WHERE theme_id = ? AND state = ? AND id <> ?
		  AND id NOT IN (
		    SELECT id FROM theme_versions WHERE theme_id = ? AND state = ? ORDER BY number DESC LIMIT ?
		  )`,
		themeID, string(theme.VersionStateSealed), protectID, themeID, string(theme.VersionStateSealed), keep)
	if res.Error != nil {
		return 0, fmt.Errorf("themeVersionRepo.PruneSealed: %w", res.Error)
	}
	return res.RowsAffected, nil
}
