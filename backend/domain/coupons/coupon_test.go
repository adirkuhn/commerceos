package coupons_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/coupons"
)

func TestIsRedeemable_activeNotExpiredNoLimit(t *testing.T) {
	c := coupons.Coupon{Active: true}
	assert.True(t, c.IsRedeemable(time.Now()))
}

func TestIsRedeemable_inactiveIsNotRedeemable(t *testing.T) {
	c := coupons.Coupon{Active: false}
	assert.False(t, c.IsRedeemable(time.Now()))
}

func TestIsRedeemable_expiredIsNotRedeemable(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	c := coupons.Coupon{Active: true, ExpiresAt: &past}
	assert.False(t, c.IsRedeemable(time.Now()))
}

func TestIsRedeemable_notYetExpiredIsRedeemable(t *testing.T) {
	future := time.Now().Add(time.Hour)
	c := coupons.Coupon{Active: true, ExpiresAt: &future}
	assert.True(t, c.IsRedeemable(time.Now()))
}

func TestIsRedeemable_atUsageLimitIsNotRedeemable(t *testing.T) {
	limit := 5
	c := coupons.Coupon{Active: true, UsageLimit: &limit, TimesUsed: 5}
	assert.False(t, c.IsRedeemable(time.Now()))
}

func TestIsRedeemable_underUsageLimitIsRedeemable(t *testing.T) {
	limit := 5
	c := coupons.Coupon{Active: true, UsageLimit: &limit, TimesUsed: 4}
	assert.True(t, c.IsRedeemable(time.Now()))
}

func TestMeetsMinSubtotal_noMinimumAlwaysMet(t *testing.T) {
	c := coupons.Coupon{}
	assert.True(t, c.MeetsMinSubtotal(0))
}

func TestMeetsMinSubtotal_belowMinimumNotMet(t *testing.T) {
	min := int64(5000)
	c := coupons.Coupon{MinSubtotalMinorUnits: &min}
	assert.False(t, c.MeetsMinSubtotal(4999))
}

func TestMeetsMinSubtotal_atOrAboveMinimumMet(t *testing.T) {
	min := int64(5000)
	c := coupons.Coupon{MinSubtotalMinorUnits: &min}
	assert.True(t, c.MeetsMinSubtotal(5000))
	assert.True(t, c.MeetsMinSubtotal(6000))
}

func TestDiscountForSubtotal_percent(t *testing.T) {
	c := coupons.Coupon{DiscountType: coupons.DiscountTypePercent, DiscountValue: 2000} // 20%
	assert.Equal(t, int64(200), c.DiscountForSubtotal(1000))
}

func TestDiscountForSubtotal_fixed(t *testing.T) {
	c := coupons.Coupon{DiscountType: coupons.DiscountTypeFixed, DiscountValue: 300}
	assert.Equal(t, int64(300), c.DiscountForSubtotal(1000))
}

func TestDiscountForSubtotal_fixedNeverExceedsSubtotal(t *testing.T) {
	c := coupons.Coupon{DiscountType: coupons.DiscountTypeFixed, DiscountValue: 5000}
	assert.Equal(t, int64(1000), c.DiscountForSubtotal(1000))
}

func TestDiscountForSubtotal_percentRoundsDown(t *testing.T) {
	c := coupons.Coupon{DiscountType: coupons.DiscountTypePercent, DiscountValue: 1000} // 10%
	assert.Equal(t, int64(99), c.DiscountForSubtotal(999))                              // 99.9 -> 99, not 100
}
