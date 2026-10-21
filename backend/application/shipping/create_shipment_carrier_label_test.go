package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type createShipmentFixture struct {
	gotRequest integrationsDomain.CreateCarrierShipmentRequest
	result     integrationsDomain.CarrierShipmentResult
	created    *shippingDomain.Shipment
	labels     *fakeShipmentLabelRepo
	profiles   *fakeCompanyProfiles
	settings   *shippingDomain.ShippingConnectionSettings
	activity   fakeRecordActivity
	prefs      *fakeShippingPreferencesRepo
}

func (f *createShipmentFixture) useCase(t *testing.T) *appshipping.CreateShipmentUseCase {
	t.Helper()
	conn := &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderPostNL, ConsumerSecretEncrypted: encryptedFor(t, "k")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(context.Context, string) (*shippingDomain.ShippingConnectionSettings, error) {
			return f.settings, nil
		},
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(context.Context, string, shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			f.gotRequest = in
			return f.result, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{
		createFn: func(_ context.Context, s *shippingDomain.Shipment) error {
			s.ID = "shipment-1"
			f.created = s
			return nil
		},
		findBySalesOrderIDFn: noExistingShipments,
	}
	if f.labels == nil {
		f.labels = &fakeShipmentLabelRepo{}
	}
	if f.profiles == nil {
		f.profiles = &fakeCompanyProfiles{}
	}
	if f.prefs == nil {
		f.prefs = &fakeShippingPreferencesRepo{}
	}
	return appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &f.activity, tenantSettingsRepoWithCurrency("EUR"),
		notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{},
		appshipping.NewShipmentAddressResolver(f.prefs, f.profiles), f.labels)
}

func createPostNLShipment(t *testing.T, uc *appshipping.CreateShipmentUseCase) *shippingDomain.Shipment {
	t.Helper()
	s, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})
	require.NoError(t, err)
	return s
}

func TestCreateShipment_sendsCompanyAddressAsSender(t *testing.T) {
	f := &createShipmentFixture{
		result: integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "ref", Barcode: "3SDEVC1"},
		profiles: &fakeCompanyProfiles{profile: &settingsDomain.CompanyProfile{
			TenantID: "tenant-1", LegalName: "Cantinho BV", TradingName: "Cantinho Brasileiro", Email: "shop@example.com", Phone: "+31612345678",
			Address: settingsDomain.CompanyAddress{Street: "Damrak", HouseNumber: "1A", PostalCode: "1012AB", City: "Amsterdam", Country: "NL"},
		}},
	}

	createPostNLShipment(t, f.useCase(t))

	require.NotNil(t, f.gotRequest.Sender)
	assert.Equal(t, integrationsDomain.CarrierAddress{
		Company: "Cantinho Brasileiro", Street: "Damrak", Number: "1A", PostalCode: "1012AB", City: "Amsterdam", CC: "NL",
		Email: "shop@example.com", Phone: "+31612345678",
	}, *f.gotRequest.Sender)
}

func TestCreateShipment_noCompanyAddressSendsNoSender(t *testing.T) {
	f := &createShipmentFixture{result: integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "ref"}}

	createPostNLShipment(t, f.useCase(t))

	assert.Nil(t, f.gotRequest.Sender)
}

func TestCreateShipment_passesResolvedLabelFormat(t *testing.T) {
	f := &createShipmentFixture{
		result:   integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "ref"},
		settings: &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage, DefaultLabelFormat: shippingDomain.LabelFormatZPL},
	}

	createPostNLShipment(t, f.useCase(t))

	assert.Equal(t, shippingDomain.LabelFormatZPL, f.gotRequest.LabelFormat)
}

func TestCreateShipment_storesLabelReturnedAtCreation(t *testing.T) {
	f := &createShipmentFixture{result: integrationsDomain.CarrierShipmentResult{
		ExternalShipmentID: "ref", Barcode: "3SDEVC1",
		Label: &integrationsDomain.LabelResult{Data: []byte("%PDF"), ContentType: "application/pdf"},
	}}

	s := createPostNLShipment(t, f.useCase(t))

	require.NotNil(t, f.labels.saved)
	assert.Equal(t, s.ID, f.labels.saved.ShipmentID)
	assert.Equal(t, "tenant-1", f.labels.saved.TenantID)
	assert.Equal(t, []byte("%PDF"), f.labels.saved.Data)
	assert.Equal(t, "application/pdf", f.labels.saved.ContentType)
}

func TestCreateShipment_carrierWithoutLabelAtCreationStoresNothing(t *testing.T) {
	f := &createShipmentFixture{result: integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "555"}}

	createPostNLShipment(t, f.useCase(t))

	assert.Nil(t, f.labels.saved)
}

func TestCreateShipment_activityNamesTheProvider(t *testing.T) {
	f := &createShipmentFixture{result: integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "ref", Barcode: "3SDEVC1"}}

	createPostNLShipment(t, f.useCase(t))

	require.Len(t, f.activity.calls, 1)
	assert.Equal(t, "postnl", f.activity.calls[0].Metadata["provider"])
	assert.Equal(t, "Shipment created via postnl", f.activity.calls[0].Summary)
}

func TestCreateShipment_sendsTenantSenderAndReturnAddress(t *testing.T) {
	sender := &shippingDomain.ShippingAddress{Company: "Warehouse", Street: "Kade", Number: "9", PostalCode: "3011AA", City: "Rotterdam", CC: "NL"}
	ret := &shippingDomain.ShippingAddress{Company: "Returns", Street: "Laan", Number: "2", PostalCode: "3511AA", City: "Utrecht", CC: "NL"}
	f := &createShipmentFixture{
		result: integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "ref"},
		prefs:  &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{SenderAddress: sender, ReturnAddress: ret}},
	}

	createPostNLShipment(t, f.useCase(t))

	require.NotNil(t, f.gotRequest.Sender)
	assert.Equal(t, "Warehouse", f.gotRequest.Sender.Company)
	require.NotNil(t, f.gotRequest.ReturnAddress)
	assert.Equal(t, "Returns", f.gotRequest.ReturnAddress.Company)
}
