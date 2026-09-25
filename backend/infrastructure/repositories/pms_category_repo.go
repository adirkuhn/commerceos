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

type categoryModel struct {
	ID                      string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                string `gorm:"not null;type:uuid"`
	ParentID                *string
	Name                    string `gorm:"not null"`
	Slug                    string `gorm:"not null"`
	Position                int
	TargetMarkupBasisPoints *int `gorm:"column:target_markup_basis_points"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (categoryModel) TableName() string { return "categories" }

// ── CategoryRepo ──────────────────────────────────────────────────────────────

type CategoryRepo struct{ db *gorm.DB }

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CategoryRepo) Create(ctx context.Context, c *pmsDomain.Category) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := categoryModel{
		TenantID:                c.TenantID,
		ParentID:                c.ParentID,
		Name:                    c.Name,
		Slug:                    c.Slug,
		Position:                c.Position,
		TargetMarkupBasisPoints: c.TargetMarkupBasisPoints,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("categoryRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *CategoryRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m categoryModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("categoryRepo.FindByID: %w", err)
	}
	return toCategoryDomain(m), nil
}

func (r *CategoryRepo) FindBySlug(ctx context.Context, slug string) (*pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m categoryModel
	if err := tx.Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("categoryRepo.FindBySlug: %w", err)
	}
	return toCategoryDomain(m), nil
}

func (r *CategoryRepo) List(ctx context.Context) ([]pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []categoryModel
	if err := tx.Order("position asc, name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("categoryRepo.List: %w", err)
	}
	categories := make([]pmsDomain.Category, len(rows))
	for i, row := range rows {
		categories[i] = *toCategoryDomain(row)
	}
	return categories, nil
}

func (r *CategoryRepo) Update(ctx context.Context, c *pmsDomain.Category) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&categoryModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"parent_id":                  c.ParentID,
			"name":                       c.Name,
			"target_markup_basis_points": c.TargetMarkupBasisPoints,
			"updated_at":                 c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("categoryRepo.Update: %w", err)
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&categoryModel{}).Error; err != nil {
		return fmt.Errorf("categoryRepo.Delete: %w", err)
	}
	return nil
}

func toCategoryDomain(m categoryModel) *pmsDomain.Category {
	return &pmsDomain.Category{
		ID:                      m.ID,
		TenantID:                m.TenantID,
		ParentID:                m.ParentID,
		Name:                    m.Name,
		Slug:                    m.Slug,
		Position:                m.Position,
		TargetMarkupBasisPoints: m.TargetMarkupBasisPoints,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}
}
