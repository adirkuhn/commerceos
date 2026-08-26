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
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const testEncSecret = "test-encryption-secret-32bytes!"

func encryptedFor(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt(testEncSecret, plaintext)
	require.NoError(t, err)
	return enc
}

func validRecipient() appshipping.RecipientInput {
	return appshipping.RecipientInput{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL"}
}

func TestCreateShipment_happyPath_resolvesOptionsCreatesAndPersists(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", TenantID: "tenant-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) { return conn, nil }}

	settings := &shippingDomain.ShippingConnectionSettings{
		DefaultPackageType: shippingDomain.PackageTypePackage, DefaultLabelFormat: shippingDomain.LabelFormatPDF,
	}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return settings, nil
		},
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}

	var gotCreateInput integrationsDomain.CreateShipmentInput
	client := &fakeMyParcelClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.ShipmentResult{ShipmentID: 555, Barcode: "3SMYPA9"}, nil
		},
	}
	factory := &fakeMyParcelClientFactory{client: client}

	var created *shippingDomain.Shipment
	shipmentRepo := &fakeShipmentRepo{
		createFn: func(_ context.Context, s *shippingDomain.Shipment) error {
			s.ID = "shipment-1"
			created = s
			return nil
		},
	}

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, overrideRepo, shipmentRepo, factory, testEncSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "shipment-1", result.ID)
	assert.Equal(t, int64(555), result.MyParcelShipmentID)
	assert.Equal(t, "3SMYPA9", result.Barcode)
	assert.Equal(t, "registered", result.StatusMain)

	assert.Equal(t, "order-1", gotCreateInput.OrderReference, "must round-trip our own SalesOrder.ID for webhook correlation")
	assert.Equal(t, 1, gotCreateInput.Options.PackageType, "falls through to connection settings' default")

	require.NotNil(t, created)
	assert.Equal(t, "tenant-1", created.TenantID)
}

func TestCreateShipment_perShipmentOverride_winsOverConnectionDefault(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage}, nil
		},
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	var gotCreateInput integrationsDomain.CreateShipmentInput
	client := &fakeMyParcelClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.ShipmentResult{ShipmentID: 1, Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	digitalStamp := shippingDomain.PackageTypeDigitalStamp
	sig := true
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
		PackageType: &digitalStamp, SignatureRequired: &sig,
	})

	require.NoError(t, err)
	assert.Equal(t, int(shippingDomain.PackageTypeDigitalStamp), gotCreateInput.Options.PackageType)
	assert.True(t, gotCreateInput.Options.SignatureRequired)
}

func TestCreateShipment_insuranceOverride_setsInsuranceAndForcesSignatureOnlyRecipient(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage}, nil
		},
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	var gotCreateInput integrationsDomain.CreateShipmentInput
	client := &fakeMyParcelClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.ShipmentResult{ShipmentID: 1, Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	amount := 50000
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
		InsuranceAmountMinorUnits: &amount,
	})

	require.NoError(t, err)
	require.NotNil(t, gotCreateInput.Options.InsuranceAmountMinorEUR)
	assert.Equal(t, 50000, *gotCreateInput.Options.InsuranceAmountMinorEUR)
	assert.True(t, gotCreateInput.Options.SignatureRequired, "MyParcel requires signature whenever a shipment is insured")
	assert.True(t, gotCreateInput.Options.OnlyRecipient, "MyParcel requires only_recipient whenever a shipment is insured")
}

func TestCreateShipment_incompleteRecipient_returnsErrMissingRecipient(t *testing.T) {
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 0), &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{},
		&fakeShippingCarrierOverrideRepo{}, &fakeShipmentRepo{}, &fakeMyParcelClientFactory{}, testEncSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL,
		Recipient: appshipping.RecipientInput{Person: "Jane"}, // missing street/city/cc
	})

	require.ErrorIs(t, err, shippingDomain.ErrMissingRecipient)
}

func TestCreateShipment_myParcelAPIError_propagatesAndDoesNotPersist(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	client := &fakeMyParcelClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
			return integrationsDomain.ShipmentResult{}, assert.AnError
		},
	}
	persisted := false
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { persisted = true; return nil }}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.Error(t, err)
	assert.False(t, persisted, "a shipment must never be persisted locally when MyParcel itself rejected it")
}
