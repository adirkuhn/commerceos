package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func companyWithAddress() *fakeCompanyProfiles {
	return &fakeCompanyProfiles{profile: &settingsDomain.CompanyProfile{
		TenantID: "tenant-1", LegalName: "Cantinho BV", TradingName: "Cantinho Brasileiro", Email: "shop@example.com", Phone: "+31612345678",
		Address: settingsDomain.CompanyAddress{Street: "Damrak", HouseNumber: "1A", PostalCode: "1012AB", City: "Amsterdam", Country: "NL"},
	}}
}

func TestShipmentAddressResolver_tenantSenderAndReturnAddressWin(t *testing.T) {
	sender := &shippingDomain.ShippingAddress{Company: "Warehouse", Street: "Kade", Number: "9", NumberSuffix: "B", PostalCode: "3011AA", City: "Rotterdam", CC: "NL", Email: "wh@example.com"}
	ret := &shippingDomain.ShippingAddress{Company: "Returns", Street: "Laan", Number: "2", PostalCode: "3511AA", City: "Utrecht", CC: "NL"}
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{SenderAddress: sender, ReturnAddress: ret}}

	got, err := appshipping.NewShipmentAddressResolver(prefs, companyWithAddress()).Resolve(context.Background())

	require.NoError(t, err)
	require.NotNil(t, got.Sender)
	assert.Equal(t, integrationsDomain.CarrierAddress{Company: "Warehouse", Street: "Kade", Number: "9", NumberSuffix: "B", PostalCode: "3011AA", City: "Rotterdam", CC: "NL", Email: "wh@example.com"}, *got.Sender)
	require.NotNil(t, got.Return)
	assert.Equal(t, "Returns", got.Return.Company)
}

func TestShipmentAddressResolver_noTenantSenderFallsBackToCompanyProfile(t *testing.T) {
	got, err := appshipping.NewShipmentAddressResolver(&fakeShippingPreferencesRepo{}, companyWithAddress()).Resolve(context.Background())

	require.NoError(t, err)
	require.NotNil(t, got.Sender)
	assert.Equal(t, integrationsDomain.CarrierAddress{
		Company: "Cantinho Brasileiro", Street: "Damrak", Number: "1A", PostalCode: "1012AB", City: "Amsterdam", CC: "NL",
		Email: "shop@example.com", Phone: "+31612345678",
	}, *got.Sender)
	assert.Nil(t, got.Return)
}

func TestShipmentAddressResolver_noAddressAnywhere(t *testing.T) {
	got, err := appshipping.NewShipmentAddressResolver(&fakeShippingPreferencesRepo{}, &fakeCompanyProfiles{}).Resolve(context.Background())

	require.NoError(t, err)
	assert.Nil(t, got.Sender)
	assert.Nil(t, got.Return)
}
