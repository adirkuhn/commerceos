package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestListShipmentsForOrder_returnsRepoResult(t *testing.T) {
	want := []shippingDomain.Shipment{{ID: "s-1"}, {ID: "s-2"}}
	repo := &fakeShipmentRepo{findBySalesOrderIDFn: func(_ context.Context, id string) ([]shippingDomain.Shipment, error) {
		assert.Equal(t, "order-1", id)
		return want, nil
	}}
	uc := appshipping.NewListShipmentsForOrderUseCase(newMockGormDB(t, 1), repo)

	got, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}
