package coupons_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
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
