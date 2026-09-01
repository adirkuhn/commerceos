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
	want := []shippingDomain.Shipment{{ID: "s-1", ConnectionID: "conn-1"}, {ID: "s-2", ConnectionID: "conn-1"}}
	repo := &fakeShipmentRepo{findBySalesOrderIDFn: func(_ context.Context, id string) ([]shippingDomain.Shipment, error) {
		assert.Equal(t, "order-1", id)
		return want, nil
	}}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(context.Context, string, shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	uc := appshipping.NewListShipmentsForOrderUseCase(newMockGormDB(t, 1), repo, overrideRepo)

	got, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestListShipmentsForOrder_appliesConfiguredCarrierTrackingTemplate(t *testing.T) {
	shipments := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1", RecipientPostalCode: "1251 SB", TrackingURL: "https://myparcel-default/x"},
	}
	repo := &fakeShipmentRepo{findBySalesOrderIDFn: func(context.Context, string) ([]shippingDomain.Shipment, error) {
		return shipments, nil
	}}
	tmpl := "https://postnl.nl/tracktrace/?B={barcode}&P={postcode}&D=NL&T=C&L=EN"
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			assert.Equal(t, "conn-1", connectionID)
			assert.Equal(t, shippingDomain.CarrierPostNL, carrier)
			return &shippingDomain.ShippingCarrierOverride{TrackingURLTemplate: &tmpl}, nil
		},
	}
	uc := appshipping.NewListShipmentsForOrderUseCase(newMockGormDB(t, 1), repo, overrideRepo)

	got, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "https://postnl.nl/tracktrace/?B=3SMYPA1&P=1251%20SB&D=NL&T=C&L=EN", got[0].TrackingURL, "configured template wins over MyParcel's own default link")
}

func TestListShipmentsForOrder_noOverride_keepsMyParcelDefaultLink(t *testing.T) {
	shipments := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierDPD, Barcode: "05162815629376", TrackingURL: "https://myparcel-default/x"},
	}
	repo := &fakeShipmentRepo{findBySalesOrderIDFn: func(context.Context, string) ([]shippingDomain.Shipment, error) {
		return shipments, nil
	}}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(context.Context, string, shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	uc := appshipping.NewListShipmentsForOrderUseCase(newMockGormDB(t, 1), repo, overrideRepo)

	got, err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "https://myparcel-default/x", got[0].TrackingURL)
}
