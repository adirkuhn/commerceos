package coupons_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appCoupons "github.com/adirkuhn/commerceos/application/coupons"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateCoupon_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appCoupons.NewCreateCouponUseCase(newMockGormDB(t, 1), &fakeCouponRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appCoupons.CreateCouponInput{Code: " xmas10 ", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "coupons.coupon.created", e.Subject)
	assert.Equal(t, "coupon-new", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"code": "XMAS10", "discount_type": "percent", "discount_value": float64(1000),
		"min_subtotal_minor_units": nil, "usage_limit": nil, "expires_at": nil,
	}, e.Data())
}

func TestUpdateCoupon_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"c-1": {
		ID: "c-1", Code: "XMAS10", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000, Active: true,
	}}}
	uc := appCoupons.NewUpdateCouponUseCase(newMockGormDB(t, 1), repo, outbox)

	_, err := uc.Execute(context.Background(), appCoupons.UpdateCouponInput{
		ID: "c-1", Code: "XMAS10", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1500, Active: false,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "coupons.coupon.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"discount_value": {float64(1000), float64(1500)}, "active": {true, false}}, e.Changes())
}

func TestDeleteCoupon_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appCoupons.NewDeleteCouponUseCase(newMockGormDB(t, 1), &fakeCouponRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "c-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "coupons.coupon.deleted", e.Subject)
	assert.Equal(t, "c-1", e.Payload["entity_id"])
}

func TestUpsertCouponCurrencyAmount_RecordsSet(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"c-1": {ID: "c-1"}}}
	uc := appCoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 1), repo, &fakeCouponCurrencyAmountRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appCoupons.UpsertCouponCurrencyAmountInput{CouponID: "c-1", CurrencyCode: "EUR", AmountMinorUnits: 500})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "coupons.coupon_currency_amount.set", e.Subject)
	assert.Equal(t, "c-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"currency_code": "EUR", "amount_minor_units": float64(500)}, e.Data())
}

func TestDeleteCouponCurrencyAmount_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appCoupons.NewDeleteCouponCurrencyAmountUseCase(newMockGormDB(t, 1), &fakeCouponCurrencyAmountRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "c-1", "EUR"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "coupons.coupon_currency_amount.deleted", e.Subject)
	assert.Equal(t, "c-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"currency_code": "EUR"}, e.Data())
}

func TestCouponUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()
	found := func() *fakeCouponRepo {
		return &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"c-1": {ID: "c-1"}}}
	}

	_, err := appCoupons.NewCreateCouponUseCase(newMockGormDB(t, 1), &fakeCouponRepo{}, failing).
		Execute(ctx, appCoupons.CreateCouponInput{Code: "A", DiscountType: couponsDomain.DiscountTypePercent})
	assert.ErrorIs(t, err, boom, "create")
	_, err = appCoupons.NewUpdateCouponUseCase(newMockGormDB(t, 1), found(), failing).
		Execute(ctx, appCoupons.UpdateCouponInput{ID: "c-1", Code: "A", DiscountType: couponsDomain.DiscountTypePercent})
	assert.ErrorIs(t, err, boom, "update")
	assert.ErrorIs(t, appCoupons.NewDeleteCouponUseCase(newMockGormDB(t, 1), found(), failing).Execute(ctx, "c-1"), boom, "delete")
	_, err = appCoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 1), found(), &fakeCouponCurrencyAmountRepo{}, failing).
		Execute(ctx, appCoupons.UpsertCouponCurrencyAmountInput{CouponID: "c-1", CurrencyCode: "EUR"})
	assert.ErrorIs(t, err, boom, "upsert amount")
	assert.ErrorIs(t, appCoupons.NewDeleteCouponCurrencyAmountUseCase(newMockGormDB(t, 1), &fakeCouponCurrencyAmountRepo{}, failing).
		Execute(ctx, "c-1", "EUR"), boom, "delete amount")
}
