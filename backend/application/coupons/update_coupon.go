package coupons

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type UpdateCouponInput struct {
	ID                    string
	Code                  string
	DiscountType          couponsDomain.DiscountType
	DiscountValue         int
	MinSubtotalMinorUnits *int64
	UsageLimit            *int
	ExpiresAt             *time.Time
	Active                bool
}

type UpdateCouponUseCase struct {
	db   *gorm.DB
	repo couponsDomain.Repository
}

func NewUpdateCouponUseCase(db *gorm.DB, repo couponsDomain.Repository) *UpdateCouponUseCase {
	return &UpdateCouponUseCase{db: db, repo: repo}
}

func (uc *UpdateCouponUseCase) Execute(ctx context.Context, in UpdateCouponInput) (*couponsDomain.Coupon, error) {
	code := normalizeCode(in.Code)
	if err := validateCouponInput(code, in.DiscountType, in.DiscountValue, in.MinSubtotalMinorUnits, in.UsageLimit); err != nil {
		return nil, fmt.Errorf("updateCoupon: %w", err)
	}

	var coupon couponsDomain.Coupon
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Code = code
		existing.DiscountType = in.DiscountType
		existing.DiscountValue = in.DiscountValue
		existing.MinSubtotalMinorUnits = in.MinSubtotalMinorUnits
		existing.UsageLimit = in.UsageLimit
		existing.ExpiresAt = in.ExpiresAt
		existing.Active = in.Active
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		coupon = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateCoupon: %w", err)
	}
	return &coupon, nil
}
