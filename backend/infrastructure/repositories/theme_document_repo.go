package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/theme"
)

type themeDocumentModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"type:uuid"`
	VersionID string `gorm:"type:uuid"`
	Kind      string
	Key       string
	Title     string
	SEO       jsonb[themeSEOJSON]  `gorm:"column:seo;type:jsonb"`
	Tree      jsonb[themeNodeJSON] `gorm:"type:jsonb"`
	// Translations: title/SEO per language (see theme.Document.Translations).
	Translations jsonb[themeDocTranslationsJSON] `gorm:"type:jsonb"`
	UpdatedAt    time.Time
}

func (themeDocumentModel) TableName() string { return "theme_documents" }

func (m themeDocumentModel) toDomain() theme.Document {
	return theme.Document{
		ID: m.ID, TenantID: m.TenantID, VersionID: m.VersionID, Kind: theme.DocumentKind(m.Kind), Key: m.Key,
		Title: m.Title, SEO: theme.SEO{Title: m.SEO.V.Title, Description: m.SEO.V.Description}, Root: m.Tree.V.toDomain(),
		Translations: m.Translations.V.toDomain(),
	}
}

type ThemeDocumentRepo struct{ db *gorm.DB }

func NewThemeDocumentRepo(db *gorm.DB) *ThemeDocumentRepo { return &ThemeDocumentRepo{db: db} }

var _ theme.DocumentRepository = (*ThemeDocumentRepo)(nil)

func (r *ThemeDocumentRepo) Create(ctx context.Context, d *theme.Document) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := themeDocumentModel{
		TenantID: d.TenantID, VersionID: d.VersionID, Kind: string(d.Kind), Key: d.Key, Title: d.Title,
		SEO:  jsonb[themeSEOJSON]{V: themeSEOJSON{Title: d.SEO.Title, Description: d.SEO.Description}},
		Tree: jsonb[themeNodeJSON]{V: toNodeJSON(d.Root)}, Translations: jsonb[themeDocTranslationsJSON]{V: toDocTranslationsJSON(d.Translations)},
		UpdatedAt: time.Now(),
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("themeDocumentRepo.Create: %w", err)
	}
	d.ID = m.ID
	return nil
}

func (r *ThemeDocumentRepo) CopyAll(ctx context.Context, fromVersionID, toVersionID string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO theme_documents (tenant_id, version_id, kind, key, title, seo, translations, tree, updated_at)
		SELECT tenant_id, ?, kind, key, title, seo, translations, tree, NOW() FROM theme_documents WHERE version_id = ?`,
		toVersionID, fromVersionID).Error; err != nil {
		return fmt.Errorf("themeDocumentRepo.CopyAll: %w", err)
	}
	return nil
}

func (r *ThemeDocumentRepo) Find(ctx context.Context, versionID string, kind theme.DocumentKind, key string) (*theme.Document, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var m themeDocumentModel
	if err := tx.Where("version_id = ? AND kind = ? AND key = ?", versionID, string(kind), key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, theme.ErrDocumentNotFound
		}
		return nil, fmt.Errorf("themeDocumentRepo.Find: %w", err)
	}
	d := m.toDomain()
	return &d, nil
}

func (r *ThemeDocumentRepo) List(ctx context.Context, versionID string) ([]theme.Document, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []themeDocumentModel
	if err := tx.Select("id", "tenant_id", "version_id", "kind", "key", "title", "seo", "translations", "updated_at").
		Where("version_id = ?", versionID).Order("kind, key").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("themeDocumentRepo.List: %w", err)
	}
	out := make([]theme.Document, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, nil
}

func (r *ThemeDocumentRepo) Update(ctx context.Context, d *theme.Document) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&themeDocumentModel{}).Where("id = ?", d.ID).Updates(map[string]any{
		"title":        d.Title,
		"seo":          jsonb[themeSEOJSON]{V: themeSEOJSON{Title: d.SEO.Title, Description: d.SEO.Description}},
		"tree":         jsonb[themeNodeJSON]{V: toNodeJSON(d.Root)},
		"translations": jsonb[themeDocTranslationsJSON]{V: toDocTranslationsJSON(d.Translations)},
		"updated_at":   time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("themeDocumentRepo.Update: %w", err)
	}
	return nil
}

func (r *ThemeDocumentRepo) Delete(ctx context.Context, id string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&themeDocumentModel{}).Error; err != nil {
		return fmt.Errorf("themeDocumentRepo.Delete: %w", err)
	}
	return nil
}
