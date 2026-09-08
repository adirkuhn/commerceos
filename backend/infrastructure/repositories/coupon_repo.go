package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type couponModel struct {
	ID                    string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID              string `gorm:"not null;type:uuid"`
	Code                  string `gorm:"not null"`
	DiscountType          string `gorm:"column:discount_type;not null"`
	DiscountValue         int    `gorm:"column:discount_value;not null"`
	MinSubtotalMinorUnits *int64 `gorm:"column:min_subtotal_minor_units"`
	UsageLimit            *int   `gorm:"column:usage_limit"`
	TimesUsed             int    `gorm:"column:times_used;not null;default:0"`
	ExpiresAt             *time.Time
	Active                bool `gorm:"not null;default:true"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (couponModel) TableName() string { return "coupons" }

type CouponRepo struct{ db *gorm.DB }

func NewCouponRepo(db *gorm.DB) *CouponRepo { return &CouponRepo{db: db} }

func (r *CouponRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("couponRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CouponRepo) Create(ctx context.Context, c *couponsDomain.Coupon) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := couponModel{
		TenantID: c.TenantID, Code: c.Code, DiscountType: string(c.DiscountType), DiscountValue: c.DiscountValue,
		MinSubtotalMinorUnits: c.MinSubtotalMinorUnits, UsageLimit: c.UsageLimit, TimesUsed: c.TimesUsed,
		ExpiresAt: c.ExpiresAt, Active: c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return couponsDomain.ErrCouponCodeTaken
		}
		return fmt.Errorf("couponRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *CouponRepo) FindByID(ctx context.Context, id string) (*couponsDomain.Coupon, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m couponModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, couponsDomain.ErrCouponNotFound
		}
		return nil, fmt.Errorf("couponRepo.FindByID: %w", err)
	}
	return toCouponDomain(m), nil
}

// FindByCode is case-insensitive on the stored (already-uppercased, see
// application/coupons) code -- normalized the same way on lookup so a
// shopper's mixed-case input still matches.
func (r *CouponRepo) FindByCode(ctx context.Context, code string) (*couponsDomain.Coupon, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m couponModel
	err = tx.Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("couponRepo.FindByCode: %w", err)
	}
	return toCouponDomain(m), nil
}

func (r *CouponRepo) ListByTenant(ctx context.Context) ([]couponsDomain.Coupon, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []couponModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("couponRepo.ListByTenant: %w", err)
	}
	out := make([]couponsDomain.Coupon, len(rows))
	for i, row := range rows {
		out[i] = *toCouponDomain(row)
	}
	return out, nil
}

func (r *CouponRepo) Update(ctx context.Context, c *couponsDomain.Coupon) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Model(&couponModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"code": c.Code, "discount_type": string(c.DiscountType), "discount_value": c.DiscountValue,
		"min_subtotal_minor_units": c.MinSubtotalMinorUnits, "usage_limit": c.UsageLimit,
		"expires_at": c.ExpiresAt, "active": c.Active, "updated_at": c.UpdatedAt,
	}).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return couponsDomain.ErrCouponCodeTaken
		}
		return fmt.Errorf("couponRepo.Update: %w", err)
	}
	return nil
}

func (r *CouponRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&couponModel{}).Error; err != nil {
		return fmt.Errorf("couponRepo.Delete: %w", err)
	}
	return nil
}

func (r *CouponRepo) IncrementTimesUsed(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&couponModel{}).Where("id = ?", id).UpdateColumn("times_used", gorm.Expr("times_used + 1")).Error; err != nil {
		return fmt.Errorf("couponRepo.IncrementTimesUsed: %w", err)
	}
	return nil
}

func toCouponDomain(m couponModel) *couponsDomain.Coupon {
	return &couponsDomain.Coupon{
		ID: m.ID, TenantID: m.TenantID, Code: m.Code, DiscountType: couponsDomain.DiscountType(m.DiscountType),
		DiscountValue: m.DiscountValue, MinSubtotalMinorUnits: m.MinSubtotalMinorUnits, UsageLimit: m.UsageLimit,
		TimesUsed: m.TimesUsed, ExpiresAt: m.ExpiresAt, Active: m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
