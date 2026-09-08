package coupons

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListCouponCurrencyAmountsUseCase struct {
	db   *gorm.DB
	repo couponsDomain.CouponCurrencyAmountRepository
}

func NewListCouponCurrencyAmountsUseCase(db *gorm.DB, repo couponsDomain.CouponCurrencyAmountRepository) *ListCouponCurrencyAmountsUseCase {
	return &ListCouponCurrencyAmountsUseCase{db: db, repo: repo}
}

func (uc *ListCouponCurrencyAmountsUseCase) Execute(ctx context.Context, couponID string) ([]couponsDomain.CouponCurrencyAmount, error) {
	var amounts []couponsDomain.CouponCurrencyAmount
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		a, err := uc.repo.ListByCoupon(ctx, couponID)
		if err != nil {
			return err
		}
		amounts = a
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listCouponCurrencyAmounts: %w", err)
	}
	return amounts, nil
}
