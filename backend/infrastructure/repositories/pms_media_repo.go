package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── Media repo ───────────────────────────────────────────────────────────────

type mediaModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	ProductID *string
	VariantID *string
	URL       string `gorm:"not null"`
	Type      string `gorm:"not null"`
	Position  int    `gorm:"not null"`
	AltText   string
	CreatedAt time.Time
}

func (mediaModel) TableName() string { return "media" }

type MediaRepo struct{ db *gorm.DB }

func NewMediaRepo(db *gorm.DB) *MediaRepo { return &MediaRepo{db: db} }

func (r *MediaRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *MediaRepo) Attach(ctx context.Context, m *pmsDomain.Media) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	row := mediaModel{
		TenantID:  m.TenantID,
		ProductID: m.ProductID,
		VariantID: m.VariantID,
		URL:       m.URL,
		Type:      string(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("mediaRepo.Attach: %w", err)
	}
	m.ID = row.ID
	return nil
}

func (r *MediaRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Media, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var row mediaModel
	if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrMediaNotFound
		}
		return nil, fmt.Errorf("mediaRepo.FindByID: %w", err)
	}
	return toMediaDomain(row), nil
}

func (r *MediaRepo) ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Media, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []mediaModel
	if err := tx.Where("product_id = ?", productID).Order("position asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("mediaRepo.ListByProduct: %w", err)
	}
	out := make([]pmsDomain.Media, len(rows))
	for i, row := range rows {
		out[i] = *toMediaDomain(row)
	}
	return out, nil
}

// ListByProductIDs is ListByProduct's bulk counterpart -- one query for a
// whole page of products (see ListPOSProductCatalogUseCase) instead of one
// per product. Still ordered by position asc, so grouping the result by
// ProductID and taking the first entry per group is still "the featured
// image" (same convention frontend/src/pms/hooks/useProductMedia.ts's
// callers already use -- media[0]).
func (r *MediaRepo) ListByProductIDs(ctx context.Context, productIDs []string) ([]pmsDomain.Media, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []mediaModel
	if err := tx.Where("product_id IN ?", productIDs).Order("position asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("mediaRepo.ListByProductIDs: %w", err)
	}
	out := make([]pmsDomain.Media, len(rows))
	for i, row := range rows {
		out[i] = *toMediaDomain(row)
	}
	return out, nil
}

func (r *MediaRepo) Update(ctx context.Context, m *pmsDomain.Media) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&mediaModel{}).Where("id = ?", m.ID).
		Update("alt_text", m.AltText).Error; err != nil {
		return fmt.Errorf("mediaRepo.Update: %w", err)
	}
	return nil
}

func (r *MediaRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&mediaModel{}).Error; err != nil {
		return fmt.Errorf("mediaRepo.Delete: %w", err)
	}
	return nil
}

func toMediaDomain(m mediaModel) *pmsDomain.Media {
	return &pmsDomain.Media{
		ID:        m.ID,
		TenantID:  m.TenantID,
		ProductID: m.ProductID,
		VariantID: m.VariantID,
		URL:       m.URL,
		Type:      pmsDomain.MediaType(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
}
