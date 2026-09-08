package coupons_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

type fakeCouponRepo struct {
	byID      map[string]couponsDomain.Coupon
	byCode    map[string]couponsDomain.Coupon
	created   *couponsDomain.Coupon
	updated   *couponsDomain.Coupon
	deletedID string
	createErr error
	updateErr error
	findErr   error
}

func (f *fakeCouponRepo) Create(_ context.Context, c *couponsDomain.Coupon) error {
	if f.createErr != nil {
		return f.createErr
	}
	c.ID = "coupon-new"
	f.created = c
	return nil
}

func (f *fakeCouponRepo) FindByID(_ context.Context, id string) (*couponsDomain.Coupon, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if c, ok := f.byID[id]; ok {
		cp := c
		return &cp, nil
	}
	return nil, couponsDomain.ErrCouponNotFound
}

func (f *fakeCouponRepo) FindByCode(_ context.Context, code string) (*couponsDomain.Coupon, error) {
	if c, ok := f.byCode[code]; ok {
		cp := c
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeCouponRepo) ListByTenant(_ context.Context) ([]couponsDomain.Coupon, error) {
	out := make([]couponsDomain.Coupon, 0, len(f.byID))
	for _, c := range f.byID {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeCouponRepo) Update(_ context.Context, c *couponsDomain.Coupon) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = c
	return nil
}

func (f *fakeCouponRepo) Delete(_ context.Context, id string) error {
	f.deletedID = id
	return nil
}

func (f *fakeCouponRepo) IncrementTimesUsed(_ context.Context, id string) error {
	return nil
}

type fakeCouponCurrencyAmountRepo struct {
	byKey                       map[string]couponsDomain.CouponCurrencyAmount
	upserted                    *couponsDomain.CouponCurrencyAmount
	deletedCouponID, deletedCcy string
}

func couponCurrencyKey(couponID, currencyCode string) string { return couponID + "|" + currencyCode }

func (f *fakeCouponCurrencyAmountRepo) FindByCouponAndCurrency(_ context.Context, couponID, currencyCode string) (*couponsDomain.CouponCurrencyAmount, error) {
	if a, ok := f.byKey[couponCurrencyKey(couponID, currencyCode)]; ok {
		cp := a
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeCouponCurrencyAmountRepo) ListByCoupon(_ context.Context, couponID string) ([]couponsDomain.CouponCurrencyAmount, error) {
	var out []couponsDomain.CouponCurrencyAmount
	for _, a := range f.byKey {
		if a.CouponID == couponID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeCouponCurrencyAmountRepo) Upsert(_ context.Context, a *couponsDomain.CouponCurrencyAmount) error {
	a.ID = "coupon-currency-amount-1"
	f.upserted = a
	return nil
}

func (f *fakeCouponCurrencyAmountRepo) Delete(_ context.Context, couponID, currencyCode string) error {
	f.deletedCouponID, f.deletedCcy = couponID, currencyCode
	return nil
}

type fakeExchangeRateRepoCoupons struct {
	byCurrency map[string]fxDomain.ExchangeRate
}

func (f *fakeExchangeRateRepoCoupons) FindByCurrency(_ context.Context, currencyCode string) (*fxDomain.ExchangeRate, error) {
	if r, ok := f.byCurrency[currencyCode]; ok {
		cp := r
		return &cp, nil
	}
	return nil, nil
}
func (f *fakeExchangeRateRepoCoupons) ListByTenant(_ context.Context) ([]fxDomain.ExchangeRate, error) {
	return nil, nil
}
func (f *fakeExchangeRateRepoCoupons) Upsert(_ context.Context, _ *fxDomain.ExchangeRate) error {
	return nil
}
func (f *fakeExchangeRateRepoCoupons) Delete(_ context.Context, _ string) error { return nil }
