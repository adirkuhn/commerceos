package storefront

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ApplyCouponUseCase struct {
	db         *gorm.DB
	cartRepo   storefrontDomain.CartRepository
	couponRepo couponsDomain.Repository
}

func NewApplyCouponUseCase(db *gorm.DB, cartRepo storefrontDomain.CartRepository, couponRepo couponsDomain.Repository) *ApplyCouponUseCase {
	return &ApplyCouponUseCase{db: db, cartRepo: cartRepo, couponRepo: couponRepo}
}

// Execute attaches code to token's cart, or removes any attached coupon
// when code is empty. Stores only the code (see Cart.CouponCode's doc
// comment on why no discount amount is ever cached on the cart) --
// re-validated for real at checkout time, since the cart's subtotal (and
// the coupon's own redeemability) can both change between now and then.
func (uc *ApplyCouponUseCase) Execute(ctx context.Context, token string, code string) (*CartResult, error) {
	var cart *storefrontDomain.Cart
	var discount int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		if found.IsCheckedOut() {
			return storefrontDomain.ErrCartAlreadyCheckedOut
		}

		trimmed := strings.ToUpper(strings.TrimSpace(code))
		if trimmed == "" {
			found.CouponCode = nil
		} else {
			coupon, err := uc.couponRepo.FindByCode(ctx, trimmed)
			if err != nil {
				return err
			}
			if coupon == nil || !coupon.IsRedeemable(time.Now()) {
				return couponsDomain.ErrCouponNotRedeemable
			}
			subtotal := found.SubtotalMinorUnits()
			if !coupon.MeetsMinSubtotal(subtotal) {
				return couponsDomain.ErrCouponMinSubtotalNotMet
			}
			found.CouponCode = &trimmed
			// Computed here, inside the same transaction that just
			// validated this exact coupon -- resolveDisplayDiscount would
			// otherwise re-look it up a second time right after.
			discount = coupon.DiscountForSubtotal(subtotal)
		}
		found.UpdatedAt = time.Now()
		if err := uc.cartRepo.Update(ctx, found); err != nil {
			return err
		}
		cart = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("applyCoupon: %w", err)
	}
	return &CartResult{Cart: cart, DiscountMinorUnits: discount}, nil
}
