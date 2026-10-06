package storefront

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/domain/events"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ApplyCouponUseCase struct {
	db                 *gorm.DB
	cartRepo           storefrontDomain.CartRepository
	couponRepo         couponsDomain.Repository
	couponCurrencyRepo couponsDomain.CouponCurrencyAmountRepository
	fxRepo             fxDomain.ExchangeRateRepository
	outbox             eventrecord.Enqueuer
}

func NewApplyCouponUseCase(
	db *gorm.DB, cartRepo storefrontDomain.CartRepository, couponRepo couponsDomain.Repository,
	couponCurrencyRepo couponsDomain.CouponCurrencyAmountRepository, fxRepo fxDomain.ExchangeRateRepository,
	outbox eventrecord.Enqueuer,
) *ApplyCouponUseCase {
	return &ApplyCouponUseCase{db: db, cartRepo: cartRepo, couponRepo: couponRepo, couponCurrencyRepo: couponCurrencyRepo, fxRepo: fxRepo, outbox: outbox}
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

		previousCode := found.CouponCode
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
			effective, err := appcoupons.EffectiveCoupon(ctx, *coupon, found.CurrencyCode(), uc.couponCurrencyRepo, uc.fxRepo)
			if err != nil {
				return err
			}
			discount = effective.DiscountForSubtotal(subtotal)
		}
		found.UpdatedAt = time.Now()
		if err := uc.cartRepo.Update(ctx, found); err != nil {
			return err
		}
		cart = found
		changes := eventrecord.NewChanges().Set("coupon_code", previousCode, found.CouponCode)
		if found.CouponCode == nil {
			return eventrecord.Record(ctx, uc.outbox, events.CartCouponRemoved, found.ID, changes)
		}
		return eventrecord.Record(ctx, uc.outbox, events.CartCouponApplied, found.ID, changes,
			eventrecord.Data{"coupon_code": *found.CouponCode, "discount_minor_units": discount})
	})
	if err != nil {
		return nil, fmt.Errorf("applyCoupon: %w", err)
	}
	return &CartResult{Cart: cart, DiscountMinorUnits: discount}, nil
}
