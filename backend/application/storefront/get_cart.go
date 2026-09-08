package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// CartResult is a cart plus its live-computed coupon discount (if any) --
// returned instead of a bare *storefrontDomain.Cart by every use case that
// hands a cart back to a caller who might display its total, so the
// discount is never stale relative to the cart's current items (see
// Cart.CouponCode's own doc comment on why the cart itself caches
// nothing).
type CartResult struct {
	Cart               *storefrontDomain.Cart
	DiscountMinorUnits int64
}

// resolveDisplayDiscount is best-effort -- for showing a shopper their
// cart, never for anything that blocks it: any failure (coupon deleted,
// no longer redeemable, cart no longer meets its minimum) just shows 0
// rather than an error. CheckoutCartUseCase.resolveCouponDiscount is the
// strict counterpart used at the one point this must never be silently
// wrong.
func resolveDisplayDiscount(ctx context.Context, db *gorm.DB, couponRepo couponsDomain.Repository, cart *storefrontDomain.Cart) int64 {
	if cart.CouponCode == nil || couponRepo == nil {
		return 0
	}
	var discount int64
	_ = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		coupon, err := couponRepo.FindByCode(ctx, *cart.CouponCode)
		if err != nil || coupon == nil {
			return nil
		}
		subtotal := cart.SubtotalMinorUnits()
		if !coupon.IsRedeemable(time.Now()) || !coupon.MeetsMinSubtotal(subtotal) {
			return nil
		}
		discount = coupon.DiscountForSubtotal(subtotal)
		return nil
	})
	return discount
}

type GetCartUseCase struct {
	db         *gorm.DB
	cartRepo   storefrontDomain.CartRepository
	couponRepo couponsDomain.Repository
}

func NewGetCartUseCase(db *gorm.DB, cartRepo storefrontDomain.CartRepository, couponRepo couponsDomain.Repository) *GetCartUseCase {
	return &GetCartUseCase{db: db, cartRepo: cartRepo, couponRepo: couponRepo}
}

func (uc *GetCartUseCase) Execute(ctx context.Context, token string) (*CartResult, error) {
	var cart *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		cart = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getCart: %w", err)
	}
	return &CartResult{Cart: cart, DiscountMinorUnits: resolveDisplayDiscount(ctx, uc.db, uc.couponRepo, cart)}, nil
}
