package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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

// notInsuredSalesOrderRepo has no findByIDFn -- panics if CreateShipmentUseCase
// ever calls it, proving the order lookup is skipped entirely when the
// resolved options don't call for insurance.
func notInsuredSalesOrderRepo() *fakeSalesOrderTotalGetter { return &fakeSalesOrderTotalGetter{} }

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

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, overrideRepo, shipmentRepo, factory, testEncSecret,
		&fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

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
	assert.Nil(t, gotCreateInput.Options.InsuranceAmountMinorEUR, "not insured by default -- no sales order lookup, no amount")

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
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

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

// TestCreateShipment_insuredByCarrierOverride_computesAmountFromOrderTotal
// proves the insured amount is never a caller-supplied or pre-configured
// figure -- it's this specific order's own TotalMinorUnits, looked up only
// because the carrier override's InsureByDefault says to insure.
func TestCreateShipment_insuredByCarrierOverride_computesAmountFromOrderTotal(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage}, nil
		},
	}
	yes := true
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return &shippingDomain.ShippingCarrierOverride{Carrier: shippingDomain.CarrierPostNL, InsureByDefault: &yes}, nil
		},
	}
	var gotCreateInput integrationsDomain.CreateShipmentInput
	client := &fakeMyParcelClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.ShipmentResult{ShipmentID: 1, Barcode: "b"}, nil
		},
	}
	var created *shippingDomain.Shipment
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; created = s; return nil }}
	salesOrderRepo := &fakeSalesOrderTotalGetter{findByIDFn: func(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
		assert.Equal(t, "order-1", id)
		return &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 50000}, nil
	}}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), salesOrderRepo)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	require.NotNil(t, gotCreateInput.Options.InsuranceAmountMinorEUR)
	assert.Equal(t, 50000, *gotCreateInput.Options.InsuranceAmountMinorEUR, "insured for the order's own value, not a fixed configured amount")
	assert.True(t, gotCreateInput.Options.SignatureRequired, "MyParcel requires signature whenever a shipment is insured")
	assert.True(t, gotCreateInput.Options.OnlyRecipient, "MyParcel requires only_recipient whenever a shipment is insured")
	require.NotNil(t, created)
	require.NotNil(t, created.InsuranceAmountMinorEUR)
	assert.Equal(t, 50000, *created.InsuranceAmountMinorEUR, "the persisted shipment record reflects the computed amount too")
}

// TestCreateShipment_insuredByConnectionDefault_carrierOverrideCanTurnItOff
// proves a non-nil override always wins, including to turn insurance off
// even though the connection default has it on.
func TestCreateShipment_insuredByConnectionDefault_carrierOverrideCanTurnItOff(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage, InsureByDefault: true}, nil
		},
	}
	no := false
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return &shippingDomain.ShippingCarrierOverride{Carrier: shippingDomain.CarrierPostNL, InsureByDefault: &no}, nil
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
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Nil(t, gotCreateInput.Options.InsuranceAmountMinorEUR, "carrier override turns insurance off despite the connection default")
	assert.False(t, gotCreateInput.Options.SignatureRequired)
	assert.False(t, gotCreateInput.Options.OnlyRecipient)
}

// TestCreateShipment_usesTenantDefaultCurrencyForInsurance proves the
// currency sent to MyParcel's insurance option comes from Settings ->
// Localization -> Default Currency (settingsDomain.TenantSettings), not a
// hardcoded value -- MyParcel requires an explicit currency per shipment
// (docs section 6.A.3), it does not infer one from the account.
func TestCreateShipment_usesTenantDefaultCurrencyForInsurance(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultPackageType: shippingDomain.PackageTypePackage, InsureByDefault: true}, nil
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
	salesOrderRepo := &fakeSalesOrderTotalGetter{findByIDFn: func(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
		return &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 50000}, nil
	}}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("USD"), salesOrderRepo)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", gotCreateInput.Options.Currency)
}

func TestCreateShipment_incompleteRecipient_returnsErrMissingRecipient(t *testing.T) {
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 0), &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{},
		&fakeShippingCarrierOverrideRepo{}, &fakeShipmentRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, &fakeRecordActivity{},
		tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

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
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.Error(t, err)
	assert.False(t, persisted, "a shipment must never be persisted locally when MyParcel itself rejected it")
}

func TestCreateShipment_recordsActivityOnSuccessAndFailure(t *testing.T) {
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

	t.Run("success", func(t *testing.T) {
		client := &fakeMyParcelClient{
			createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
				return integrationsDomain.ShipmentResult{ShipmentID: 1, Barcode: "3SMYPA1"}, nil
			},
		}
		shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }}
		activity := &fakeRecordActivity{}
		uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, overrideRepo, shipmentRepo,
			&fakeMyParcelClientFactory{client: client}, testEncSecret, activity, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

		ctx := tenant.WithTenantID(context.Background(), "tenant-1")
		_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
			SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
		})

		require.NoError(t, err)
		require.Len(t, activity.calls, 1)
		assert.Equal(t, "myparcel_shipment_created", activity.calls[0].ActivityType)
		assert.Equal(t, "user-1", activity.calls[0].UserID)
	})

	t.Run("failure", func(t *testing.T) {
		client := &fakeMyParcelClient{
			createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
				return integrationsDomain.ShipmentResult{}, assert.AnError
			},
		}
		shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { return nil }}
		activity := &fakeRecordActivity{}
		uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, overrideRepo, shipmentRepo,
			&fakeMyParcelClientFactory{client: client}, testEncSecret, activity, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo())

		ctx := tenant.WithTenantID(context.Background(), "tenant-1")
		_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
			SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
		})

		require.Error(t, err)
		require.Len(t, activity.calls, 1)
		assert.Equal(t, "myparcel_error", activity.calls[0].ActivityType)
	})
}
