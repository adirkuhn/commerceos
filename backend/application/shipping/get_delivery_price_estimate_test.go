package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func newGetDeliveryPriceEstimateUseCase(t *testing.T, n int, connRepo *mocks.ConnectionRepo, factory *fakeCarrierClientFactory) *appshipping.GetDeliveryPriceEstimateUseCase {
	t.Helper()
	return appshipping.NewGetDeliveryPriceEstimateUseCase(newMockGormDB(t, n), connRepo, factory, testEncSecret)
}

func TestGetDeliveryPriceEstimate_happyPath_returnsMyParcelsPricedOptions(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", TenantID: "tenant-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
		assert.Equal(t, "conn-1", id)
		return conn, nil
	}}

	var gotInput integrationsDomain.DeliveryPriceEstimateInput
	client := &fakeCarrierClient{
		getDeliveryPriceEstimateFn: func(_ context.Context, in integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
			gotInput = in
			return []integrationsDomain.DeliveryPriceOption{
				{PriceComment: "standard", AmountMinorUnits: 1000, Currency: "EUR"},
				{PriceComment: "morning", AmountMinorUnits: 1500, Currency: "EUR"},
			}, nil
		},
	}
	factory := &fakeCarrierClientFactory{client: client}
	uc := newGetDeliveryPriceEstimateUseCase(t, 1, connRepo, factory)

	options, err := uc.Execute(context.Background(), appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: "conn-1", CC: "nl", PostalCode: "1234 AB", Number: "31", Carrier: shippingDomain.CarrierPostNL,
	})

	require.NoError(t, err)
	assert.Equal(t, "NL", gotInput.CC, "lowercase input is upper-cased before reaching MyParcel")
	assert.Equal(t, "1234 AB", gotInput.PostalCode)
	assert.Equal(t, "31", gotInput.Number)
	assert.Equal(t, 1, gotInput.Carrier)
	require.Len(t, options, 2)
	assert.Equal(t, shippingDomain.DeliveryPriceOption{PriceComment: "standard", AmountMinorUnits: 1000, Currency: "EUR"}, options[0])
}

func TestGetDeliveryPriceEstimate_rejectsUnsupportedCountryBeforeCallingMyParcel(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
		t.Fatal("must not look up the connection for an unsupported country -- reject before any I/O")
		return nil, nil
	}}
	client := &fakeCarrierClient{
		getDeliveryPriceEstimateFn: func(_ context.Context, _ integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
			t.Fatal("must not call MyParcel for an unsupported country")
			return nil, nil
		},
	}
	factory := &fakeCarrierClientFactory{client: client}
	uc := newGetDeliveryPriceEstimateUseCase(t, 0, connRepo, factory)

	_, err := uc.Execute(context.Background(), appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: "conn-1", CC: "DE", PostalCode: "10115", Number: "1", Carrier: shippingDomain.CarrierDHLEuroplus,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrDeliveryEstimateUnsupportedCountry)
}

func TestGetDeliveryPriceEstimate_propagatesMyParcelError(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeCarrierClient{
		getDeliveryPriceEstimateFn: func(_ context.Context, _ integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
			return nil, assert.AnError
		},
	}
	factory := &fakeCarrierClientFactory{client: client}
	uc := newGetDeliveryPriceEstimateUseCase(t, 1, connRepo, factory)

	_, err := uc.Execute(context.Background(), appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: "conn-1", CC: "BE", PostalCode: "1000", Number: "1", Carrier: shippingDomain.CarrierBpost,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}
