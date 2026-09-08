package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type couponCurrencyAmountModel struct {
	ID               string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         string `gorm:"not null;type:uuid"`
	CouponID         string `gorm:"column:coupon_id;not null;type:uuid"`
	CurrencyCode     string `gorm:"not null"`
	AmountMinorUnits int64  `gorm:"not null"`
}

func (couponCurrencyAmountModel) TableName() string { return "coupon_currency_amounts" }

type CouponCurrencyAmountRepo struct{ db *gorm.DB }

func NewCouponCurrencyAmountRepo(db *gorm.DB) *CouponCurrencyAmountRepo {
	return &CouponCurrencyAmountRepo{db: db}
}

func (r *CouponCurrencyAmountRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("coupons: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CouponCurrencyAmountRepo) FindByCouponAndCurrency(ctx context.Context, couponID, currencyCode string) (*couponsDomain.CouponCurrencyAmount, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m couponCurrencyAmountModel
	err = tx.Where("coupon_id = ? AND currency_code = ?", couponID, currencyCode).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("couponCurrencyAmountRepo.FindByCouponAndCurrency: %w", err)
	}
	a := toCouponCurrencyAmountDomain(m)
	return &a, nil
}

func (r *CouponCurrencyAmountRepo) ListByCoupon(ctx context.Context, couponID string) ([]couponsDomain.CouponCurrencyAmount, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []couponCurrencyAmountModel
	if err := tx.Where("coupon_id = ?", couponID).Order("currency_code ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("couponCurrencyAmountRepo.ListByCoupon: %w", err)
	}
	out := make([]couponsDomain.CouponCurrencyAmount, 0, len(models))
	for _, m := range models {
		out = append(out, toCouponCurrencyAmountDomain(m))
	}
	return out, nil
}

func (r *CouponCurrencyAmountRepo) Upsert(ctx context.Context, a *couponsDomain.CouponCurrencyAmount) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := couponCurrencyAmountModel{TenantID: a.TenantID, CouponID: a.CouponID, CurrencyCode: a.CurrencyCode, AmountMinorUnits: a.AmountMinorUnits}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "coupon_id"}, {Name: "currency_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount_minor_units"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("couponCurrencyAmountRepo.Upsert: %w", err)
	}
	a.ID = m.ID
	return nil
}

func (r *CouponCurrencyAmountRepo) Delete(ctx context.Context, couponID, currencyCode string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("coupon_id = ? AND currency_code = ?", couponID, currencyCode).Delete(&couponCurrencyAmountModel{}).Error; err != nil {
		return fmt.Errorf("couponCurrencyAmountRepo.Delete: %w", err)
	}
	return nil
}

func toCouponCurrencyAmountDomain(m couponCurrencyAmountModel) couponsDomain.CouponCurrencyAmount {
	return couponsDomain.CouponCurrencyAmount{ID: m.ID, TenantID: m.TenantID, CouponID: m.CouponID, CurrencyCode: m.CurrencyCode, AmountMinorUnits: m.AmountMinorUnits}
}
