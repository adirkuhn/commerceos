package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestGetShippingOptions_returnsPricedOptionsForAValidNLAddress(t *testing.T) {
	connRepo := &fakeConnectionRepo{byTenant: []integrationsDomain.Connection{{ID: "conn-1", Provider: integrationsDomain.ProviderMyParcel}}}
	settingsRepo := &fakeShippingSettingsRepo{out: &shippingDomain.ShippingConnectionSettings{ConnectionID: "conn-1", DefaultCarrier: shippingDomain.CarrierPostNL}}
	estimator := &fakeDeliveryPriceEstimator{out: []shippingDomain.DeliveryPriceOption{
		{PriceComment: "Standard", AmountMinorUnits: 495, Currency: "EUR"},
		{PriceComment: "Evening", AmountMinorUnits: 895, Currency: "EUR"},
	}}
	uc := appstorefront.NewGetShippingOptionsUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, estimator)

	options := uc.Execute(context.Background(), appstorefront.CheckoutAddressInput{Address1: "Damrak 1", Postcode: "1012 LG", Country: "NL"})

	require.Len(t, options, 2)
	require.Equal(t, "Standard", options[0].Title)
	require.Equal(t, int64(495), options[0].AmountMinorUnits)
	require.Equal(t, "EUR", options[0].CurrencyCode)
	require.Equal(t, "conn-1", estimator.gotInput.ConnectionID)
}

func TestGetShippingOptions_returnsNoOptionsForAnUnsupportedCountry(t *testing.T) {
	uc := appstorefront.NewGetShippingOptionsUseCase(newMockGormDB(t, 0), &fakeConnectionRepo{}, &fakeShippingSettingsRepo{}, &fakeDeliveryPriceEstimator{})

	options := uc.Execute(context.Background(), appstorefront.CheckoutAddressInput{Address1: "Main St 1", Postcode: "10115", Country: "DE"})

	require.Empty(t, options)
}

func TestGetShippingOptions_returnsNoOptionsWithNoHouseNumber(t *testing.T) {
	uc := appstorefront.NewGetShippingOptionsUseCase(newMockGormDB(t, 0), &fakeConnectionRepo{}, &fakeShippingSettingsRepo{}, &fakeDeliveryPriceEstimator{})

	options := uc.Execute(context.Background(), appstorefront.CheckoutAddressInput{Address1: "Damrak", Postcode: "1012 LG", Country: "NL"})

	require.Empty(t, options)
}

func TestGetShippingOptions_returnsNoOptionsWithNoMyParcelConnection(t *testing.T) {
	uc := appstorefront.NewGetShippingOptionsUseCase(newMockGormDB(t, 1), &fakeConnectionRepo{}, &fakeShippingSettingsRepo{}, &fakeDeliveryPriceEstimator{out: []shippingDomain.DeliveryPriceOption{{PriceComment: "Standard", AmountMinorUnits: 495}}})

	options := uc.Execute(context.Background(), appstorefront.CheckoutAddressInput{Address1: "Damrak 1", Postcode: "1012 LG", Country: "NL"})

	require.Empty(t, options)
}
