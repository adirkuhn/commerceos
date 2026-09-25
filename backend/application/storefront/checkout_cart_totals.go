package storefront

import (
	"context"
	"fmt"
	"strings"
	"time"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// checkoutTotals is computeTotals' full result -- carries everything both
// Execute (to build the order) and EstimateTotal (to quote a
// PaymentIntent) need, computed exactly once per call so the two can never
// disagree on what a cart is about to cost.
type checkoutTotals struct {
	LineItems           []appoms.CreateSalesOrderLineItemInput
	SubtotalMinorUnits  int64
	TaxTotalMinorUnits  int64
	ShippingMinorUnits  int64
	ShippingMethodTitle string
	DiscountMinorUnits  int64
	CouponCode          string
	CouponID            string
	TotalMinorUnits     int64
	CurrencyCode        string
}

func (uc *CheckoutCartUseCase) computeTotals(ctx context.Context, cart *storefrontDomain.Cart, shipping CheckoutAddressInput, wantShippingTitle string) (*checkoutTotals, error) {
	classes, rates, err := uc.loadTaxConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load tax config: %w", err)
	}

	lineItems := make([]appoms.CreateSalesOrderLineItemInput, len(cart.Items))
	var subtotal, taxTotal int64
	currency := ""
	for i, item := range cart.Items {
		lineSubtotal := item.Quantity * item.PriceMinorUnits
		lineTax := computeLineTax(lineSubtotal, item.TaxClassID, shipping.Country, classes, rates)
		lineItems[i] = appoms.CreateSalesOrderLineItemInput{
			SKUID: item.SKUID, Name: item.ProductName, SKUCode: item.SKUCode, Quantity: item.Quantity,
			PriceMinorUnits: item.PriceMinorUnits, SubtotalMinorUnits: lineSubtotal, SubtotalTaxMinorUnits: lineTax,
			TotalMinorUnits: lineSubtotal, TotalTaxMinorUnits: lineTax,
		}
		subtotal += lineSubtotal
		taxTotal += lineTax
		currency = item.CurrencyCode
	}
	// Best-effort: NL/BE only (MyParcel's own API constraint), and only
	// when a MyParcel connection + a house number are both available --
	// see estimateShipping's own doc comment. Every other case (no
	// connection, unsupported country, address we can't parse a number
	// out of, a failed MyParcel call) falls back to 0, never blocking
	// checkout.
	shippingAmount, shippingMethodTitle := uc.resolveShipping(ctx, shipping, wantShippingTitle)

	// Unlike shipping, a coupon the shopper explicitly attached is never
	// best-effort: if it's no longer redeemable (or the cart no longer
	// meets its minimum -- e.g. an item was removed after applying it),
	// checkout fails with a clear error rather than silently dropping the
	// discount the shopper is expecting.
	discountAmount, couponID, err := uc.resolveCouponDiscount(ctx, cart.CouponCode, subtotal, currency)
	if err != nil {
		return nil, err
	}
	couponCode := ""
	if cart.CouponCode != nil {
		couponCode = *cart.CouponCode
	}

	total := subtotal + taxTotal + shippingAmount - discountAmount

	return &checkoutTotals{
		LineItems: lineItems, SubtotalMinorUnits: subtotal, TaxTotalMinorUnits: taxTotal,
		ShippingMinorUnits: shippingAmount, ShippingMethodTitle: shippingMethodTitle,
		DiscountMinorUnits: discountAmount, CouponCode: couponCode, CouponID: couponID,
		TotalMinorUnits: total, CurrencyCode: currency,
	}, nil
}

// resolveCouponDiscount returns (0, "", nil) when couponCode is nil (no
// coupon attached to this cart). Otherwise re-loads the coupon fresh (see
// Cart.CouponCode's own doc comment on why the cart never caches a
// discount amount) and re-checks both IsRedeemable and MeetsMinSubtotal --
// either failing returns an error that Execute propagates, never silently
// drops the discount. Returns the coupon's ID (for IncrementTimesUsed)
// alongside the amount.
func (uc *CheckoutCartUseCase) resolveCouponDiscount(ctx context.Context, couponCode *string, subtotal int64, channelCurrency string) (int64, string, error) {
	if couponCode == nil {
		return 0, "", nil
	}
	var coupon *couponsDomain.Coupon
	var effective couponsDomain.Coupon
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		coupon, err = uc.couponRepo.FindByCode(ctx, *couponCode)
		if err != nil {
			return err
		}
		if coupon == nil {
			return nil
		}
		effective, err = appcoupons.EffectiveCoupon(ctx, *coupon, channelCurrency, uc.couponCurrencyRepo, uc.fxRepo)
		return err
	})
	if err != nil {
		return 0, "", err
	}
	if coupon == nil || !coupon.IsRedeemable(time.Now()) {
		return 0, "", couponsDomain.ErrCouponNotRedeemable
	}
	if !coupon.MeetsMinSubtotal(subtotal) {
		return 0, "", couponsDomain.ErrCouponMinSubtotalNotMet
	}
	return effective.DiscountForSubtotal(subtotal), coupon.ID, nil
}

func (uc *CheckoutCartUseCase) loadTaxConfig(ctx context.Context) ([]taxDomain.TaxClass, []taxDomain.TaxRate, error) {
	var classes []taxDomain.TaxClass
	var rates []taxDomain.TaxRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if classes, err = uc.taxClassRepo.List(ctx); err != nil {
			return err
		}
		rates, err = uc.taxRateRepo.List(ctx)
		return err
	})
	return classes, rates, err
}

// computeLineTax resolves the effective tax class (the line's own, else
// the tenant's default) and country rate, applying it to lineSubtotal --
// 0 when no matching rate is configured, same "unset means unset, never an
// error" convention as settings.TenantSettings' zero-value fields.
func computeLineTax(lineSubtotal int64, taxClassID *string, country string, classes []taxDomain.TaxClass, rates []taxDomain.TaxRate) int64 {
	classID := ""
	if taxClassID != nil {
		classID = *taxClassID
	} else {
		for _, c := range classes {
			if c.IsDefault {
				classID = c.ID
				break
			}
		}
	}
	if classID == "" {
		return 0
	}
	for _, r := range rates {
		if r.TaxClassID == classID && strings.EqualFold(r.Country, country) {
			return lineSubtotal * int64(r.RateBasisPoints) / 10000
		}
	}
	return 0
}
