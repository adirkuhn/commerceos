package coupons_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
)

func TestDeleteCoupon_success(t *testing.T) {
	repo := &fakeCouponRepo{}
	uc := appcoupons.NewDeleteCouponUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "coupon-1")

	require.NoError(t, err)
	require.Equal(t, "coupon-1", repo.deletedID)
}
