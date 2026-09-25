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

type variantModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	ProductID       string `gorm:"not null;type:uuid"`
	Name            string `gorm:"not null"`
	Position        int
	FeaturedMediaID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (variantModel) TableName() string { return "variants" }

// ── VariantRepo ───────────────────────────────────────────────────────────────

type VariantRepo struct{ db *gorm.DB }

func NewVariantRepo(db *gorm.DB) *VariantRepo { return &VariantRepo{db: db} }

func (r *VariantRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *VariantRepo) Create(ctx context.Context, v *pmsDomain.Variant) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := variantModel{
		TenantID:        v.TenantID,
		ProductID:       v.ProductID,
		Name:            v.Name,
		Position:        v.Position,
		FeaturedMediaID: v.FeaturedMediaID,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("variantRepo.Create: %w", err)
	}
	v.ID = m.ID
	return nil
}

func (r *VariantRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m variantModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrVariantNotFound
		}
		return nil, fmt.Errorf("variantRepo.FindByID: %w", err)
	}
	return toVariantDomain(m), nil
}

func (r *VariantRepo) ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []variantModel
	if err := tx.Where("product_id = ?", productID).Order("position asc, name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("variantRepo.ListByProduct: %w", err)
	}
	variants := make([]pmsDomain.Variant, len(rows))
	for i, row := range rows {
		variants[i] = *toVariantDomain(row)
	}
	return variants, nil
}

// ListByProductIDs bulk-loads variants for multiple products in one query --
// avoids N+1 when assembling a paginated product list that also needs each
// product's variants (see application/pms.ListPOSProductCatalogUseCase).
func (r *VariantRepo) ListByProductIDs(ctx context.Context, productIDs []string) ([]pmsDomain.Variant, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if len(productIDs) == 0 {
		return nil, nil
	}
	var rows []variantModel
	if err := tx.Where("product_id IN ?", productIDs).Order("position asc, name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("variantRepo.ListByProductIDs: %w", err)
	}
	variants := make([]pmsDomain.Variant, len(rows))
	for i, row := range rows {
		variants[i] = *toVariantDomain(row)
	}
	return variants, nil
}

func (r *VariantRepo) Update(ctx context.Context, v *pmsDomain.Variant) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&variantModel{}).Where("id = ?", v.ID).
		Updates(map[string]any{
			"name":              v.Name,
			"position":          v.Position,
			"featured_media_id": v.FeaturedMediaID,
			"updated_at":        v.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("variantRepo.Update: %w", err)
	}
	return nil
}

func (r *VariantRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&variantModel{}).Error; err != nil {
		return fmt.Errorf("variantRepo.Delete: %w", err)
	}
	return nil
}

func toVariantDomain(m variantModel) *pmsDomain.Variant {
	return &pmsDomain.Variant{
		ID:              m.ID,
		TenantID:        m.TenantID,
		ProductID:       m.ProductID,
		Name:            m.Name,
		Position:        m.Position,
		FeaturedMediaID: m.FeaturedMediaID,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
