package coupons_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
)

func TestDeleteCouponCurrencyAmountUseCase_Execute(t *testing.T) {
	repo := &fakeCouponCurrencyAmountRepo{}
	uc := appcoupons.NewDeleteCouponCurrencyAmountUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "coupon-1", "USD")
	require.NoError(t, err)
	require.Equal(t, "coupon-1", repo.deletedCouponID)
	require.Equal(t, "USD", repo.deletedCcy)
}
