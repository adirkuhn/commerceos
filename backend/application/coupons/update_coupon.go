package coupons

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	db     *gorm.DB
	repo   couponsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewUpdateCouponUseCase(db *gorm.DB, repo couponsDomain.Repository, outbox eventrecord.Enqueuer) *UpdateCouponUseCase {
	return &UpdateCouponUseCase{db: db, repo: repo, outbox: outbox}
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
		before := *existing
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
		return eventrecord.Record(ctx, uc.outbox, events.CouponUpdated, coupon.ID, eventrecord.NewChanges().
			Set("code", before.Code, coupon.Code).
			Set("discount_type", string(before.DiscountType), string(coupon.DiscountType)).
			Set("discount_value", before.DiscountValue, coupon.DiscountValue).
			Set("min_subtotal_minor_units", before.MinSubtotalMinorUnits, coupon.MinSubtotalMinorUnits).
			Set("usage_limit", before.UsageLimit, coupon.UsageLimit).
			Set("expires_at", before.ExpiresAt, coupon.ExpiresAt).
			Set("active", before.Active, coupon.Active))
	})
	if err != nil {
		return nil, fmt.Errorf("updateCoupon: %w", err)
	}
	return &coupon, nil
}
