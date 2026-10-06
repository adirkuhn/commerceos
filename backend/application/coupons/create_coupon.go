// Package coupons holds the admin CRUD use cases for domain/coupons.Coupon
// -- create/update/list/delete, all tenant-scoped via tenant.WithTenant
// same as every other admin-managed resource in this codebase.
package coupons

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"strings"
	"time"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateCouponInput struct {
	Code                  string
	DiscountType          couponsDomain.DiscountType
	DiscountValue         int
	MinSubtotalMinorUnits *int64
	UsageLimit            *int
	ExpiresAt             *time.Time
}

// normalizeCode uppercases+trims a coupon code -- the one canonical form
// stored and matched against, so "summer10", "SUMMER10", and " Summer10 "
// are all the same code (see CouponRepo.FindByCode's own doc comment).
func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// validateCouponInput is shared by Create and Update -- same field rules
// apply whichever way a coupon got into this shape.
func validateCouponInput(code string, discountType couponsDomain.DiscountType, discountValue int, minSubtotal *int64, usageLimit *int) error {
	if code == "" {
		return couponsDomain.ErrCouponCodeRequired
	}
	if !discountType.Valid() {
		return couponsDomain.ErrInvalidDiscountType
	}
	if discountType == couponsDomain.DiscountTypePercent && (discountValue < 0 || discountValue > 10000) {
		return couponsDomain.ErrInvalidDiscountValue
	}
	if discountType == couponsDomain.DiscountTypeFixed && discountValue < 0 {
		return couponsDomain.ErrInvalidDiscountValue
	}
	if minSubtotal != nil && *minSubtotal < 0 {
		return couponsDomain.ErrInvalidMinSubtotal
	}
	if usageLimit != nil && *usageLimit <= 0 {
		return couponsDomain.ErrInvalidUsageLimit
	}
	return nil
}

type CreateCouponUseCase struct {
	db     *gorm.DB
	repo   couponsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewCreateCouponUseCase(db *gorm.DB, repo couponsDomain.Repository, outbox eventrecord.Enqueuer) *CreateCouponUseCase {
	return &CreateCouponUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *CreateCouponUseCase) Execute(ctx context.Context, in CreateCouponInput) (*couponsDomain.Coupon, error) {
	code := normalizeCode(in.Code)
	if err := validateCouponInput(code, in.DiscountType, in.DiscountValue, in.MinSubtotalMinorUnits, in.UsageLimit); err != nil {
		return nil, fmt.Errorf("createCoupon: %w", err)
	}

	var coupon couponsDomain.Coupon
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		coupon = couponsDomain.Coupon{
			TenantID: tenant.IDFromContext(ctx), Code: code, DiscountType: in.DiscountType, DiscountValue: in.DiscountValue,
			MinSubtotalMinorUnits: in.MinSubtotalMinorUnits, UsageLimit: in.UsageLimit, ExpiresAt: in.ExpiresAt,
			Active: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Create(ctx, &coupon); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CouponCreated, coupon.ID, eventrecord.Data{
			"code": coupon.Code, "discount_type": string(coupon.DiscountType), "discount_value": coupon.DiscountValue,
			"min_subtotal_minor_units": coupon.MinSubtotalMinorUnits, "usage_limit": coupon.UsageLimit, "expires_at": coupon.ExpiresAt,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createCoupon: %w", err)
	}
	return &coupon, nil
}
