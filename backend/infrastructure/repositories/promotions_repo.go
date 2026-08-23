package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type promotionModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	Name      string `gorm:"not null"`
	Type      string `gorm:"not null"`
	StartsAt  time.Time
	EndsAt    *time.Time
	Status    string `gorm:"not null;default:draft"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (promotionModel) TableName() string { return "promotions" }

type promotionSKUModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	PromotionID   string `gorm:"column:promotion_id;not null;type:uuid"`
	SKUID         string `gorm:"column:sku_id;not null;type:uuid"`
	DiscountType  string `gorm:"not null"`
	DiscountValue int64  `gorm:"not null"`
	CreatedAt     time.Time
}

func (promotionSKUModel) TableName() string { return "promotion_skus" }

// ── PromotionRepo ────────────────────────────────────────────────────────

type PromotionRepo struct{ db *gorm.DB }

func NewPromotionRepo(db *gorm.DB) *PromotionRepo { return &PromotionRepo{db: db} }

func (r *PromotionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("promotions: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PromotionRepo) Create(ctx context.Context, p *promotionsDomain.Promotion) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := promotionModel{
		TenantID: p.TenantID, Name: p.Name, Type: string(p.Type),
		StartsAt: p.StartsAt, EndsAt: p.EndsAt, Status: string(p.Status),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("promotionRepo.Create: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *PromotionRepo) FindByID(ctx context.Context, id string) (*promotionsDomain.Promotion, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m promotionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, promotionsDomain.ErrPromotionNotFound
		}
		return nil, fmt.Errorf("promotionRepo.FindByID: %w", err)
	}
	return toPromotionDomain(m), nil
}

func (r *PromotionRepo) List(ctx context.Context) ([]promotionsDomain.Promotion, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []promotionModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("promotionRepo.List: %w", err)
	}
	promos := make([]promotionsDomain.Promotion, len(rows))
	for i, row := range rows {
		promos[i] = *toPromotionDomain(row)
	}
	return promos, nil
}

func (r *PromotionRepo) Update(ctx context.Context, p *promotionsDomain.Promotion) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&promotionModel{}).Where("id = ?", p.ID).
		Updates(map[string]any{
			"name":       p.Name,
			"type":       string(p.Type),
			"starts_at":  p.StartsAt,
			"ends_at":    p.EndsAt,
			"status":     string(p.Status),
			"updated_at": p.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("promotionRepo.Update: %w", err)
	}
	return nil
}

func (r *PromotionRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&promotionModel{}).Error; err != nil {
		return fmt.Errorf("promotionRepo.Delete: %w", err)
	}
	return nil
}

func toPromotionDomain(m promotionModel) *promotionsDomain.Promotion {
	return &promotionsDomain.Promotion{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Type: promotionsDomain.PromotionType(m.Type),
		StartsAt: m.StartsAt, EndsAt: m.EndsAt, Status: promotionsDomain.PromotionStatus(m.Status),
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── PromotionSKURepo ─────────────────────────────────────────────────────

type PromotionSKURepo struct{ db *gorm.DB }

func NewPromotionSKURepo(db *gorm.DB) *PromotionSKURepo { return &PromotionSKURepo{db: db} }

func (r *PromotionSKURepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("promotions: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PromotionSKURepo) Create(ctx context.Context, line *promotionsDomain.PromotionSKU) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := promotionSKUModel{
		TenantID: line.TenantID, PromotionID: line.PromotionID, SKUID: line.SKUID,
		DiscountType: string(line.DiscountType), DiscountValue: line.DiscountValue, CreatedAt: line.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("promotionSKURepo.Create: %w", err)
	}
	line.ID = m.ID
	return nil
}

func (r *PromotionSKURepo) ListByPromotion(ctx context.Context, promotionID string) ([]promotionsDomain.PromotionSKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []promotionSKUModel
	if err := tx.Where("promotion_id = ?", promotionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("promotionSKURepo.ListByPromotion: %w", err)
	}
	lines := make([]promotionsDomain.PromotionSKU, len(rows))
	for i, row := range rows {
		lines[i] = promotionsDomain.PromotionSKU{
			ID: row.ID, TenantID: row.TenantID, PromotionID: row.PromotionID, SKUID: row.SKUID,
			DiscountType: promotionsDomain.PromotionType(row.DiscountType), DiscountValue: row.DiscountValue,
			CreatedAt: row.CreatedAt,
		}
	}
	return lines, nil
}

func (r *PromotionSKURepo) DeleteByPromotion(ctx context.Context, promotionID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("promotion_id = ?", promotionID).Delete(&promotionSKUModel{}).Error; err != nil {
		return fmt.Errorf("promotionSKURepo.DeleteByPromotion: %w", err)
	}
	return nil
}
