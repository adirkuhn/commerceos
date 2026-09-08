package coupons

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpsertCouponCurrencyAmountInput struct {
	CouponID         string
	CurrencyCode     string
	AmountMinorUnits int64
}

type UpsertCouponCurrencyAmountUseCase struct {
	db         *gorm.DB
	couponRepo couponsDomain.Repository
	repo       couponsDomain.CouponCurrencyAmountRepository
}

func NewUpsertCouponCurrencyAmountUseCase(db *gorm.DB, couponRepo couponsDomain.Repository, repo couponsDomain.CouponCurrencyAmountRepository) *UpsertCouponCurrencyAmountUseCase {
	return &UpsertCouponCurrencyAmountUseCase{db: db, couponRepo: couponRepo, repo: repo}
}

func (uc *UpsertCouponCurrencyAmountUseCase) Execute(ctx context.Context, in UpsertCouponCurrencyAmountInput) (*couponsDomain.CouponCurrencyAmount, error) {
	if !fxDomain.ValidCurrencyCode(in.CurrencyCode) {
		return nil, fmt.Errorf("upsertCouponCurrencyAmount: %w", fxDomain.ErrInvalidCurrencyCode)
	}
	if in.AmountMinorUnits < 0 {
		return nil, fmt.Errorf("upsertCouponCurrencyAmount: %w", couponsDomain.ErrInvalidDiscountValue)
	}

	var amount couponsDomain.CouponCurrencyAmount
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		coupon, err := uc.couponRepo.FindByID(ctx, in.CouponID)
		if err != nil {
			return err
		}
		amount = couponsDomain.CouponCurrencyAmount{
			TenantID: tenant.IDFromContext(ctx), CouponID: coupon.ID, CurrencyCode: in.CurrencyCode, AmountMinorUnits: in.AmountMinorUnits,
		}
		return uc.repo.Upsert(ctx, &amount)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertCouponCurrencyAmount: %w", err)
	}
	return &amount, nil
}
