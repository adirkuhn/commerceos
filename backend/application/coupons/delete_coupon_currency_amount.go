package coupons

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteCouponCurrencyAmountUseCase struct {
	db   *gorm.DB
	repo couponsDomain.CouponCurrencyAmountRepository
}

func NewDeleteCouponCurrencyAmountUseCase(db *gorm.DB, repo couponsDomain.CouponCurrencyAmountRepository) *DeleteCouponCurrencyAmountUseCase {
	return &DeleteCouponCurrencyAmountUseCase{db: db, repo: repo}
}

func (uc *DeleteCouponCurrencyAmountUseCase) Execute(ctx context.Context, couponID, currencyCode string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.repo.Delete(ctx, couponID, currencyCode)
	})
	if err != nil {
		return fmt.Errorf("deleteCouponCurrencyAmount: %w", err)
	}
	return nil
}
