package shipping_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	return appshipping.RecipientInput{Person: "Jane Doe", Street: "Damrak", Number: "1", City: "Amsterdam", CC: "NL", Email: "jane@example.com"}
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

	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "555", Barcode: "3SMYPA9"}, nil
		},
	}
	factory := &fakeCarrierClientFactory{client: client}

	var created *shippingDomain.Shipment
	shipmentRepo := &fakeShipmentRepo{
		createFn: func(_ context.Context, s *shippingDomain.Shipment) error {
			s.ID = "shipment-1"
			created = s
			return nil
		},
		findBySalesOrderIDFn: noExistingShipments,
	}

	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo, factory, testEncSecret,
		&fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, outbox, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "shipment-1", result.ID)
	assert.Equal(t, "555", result.ExternalShipmentID)
	assert.Equal(t, "3SMYPA9", result.Barcode)
	assert.Equal(t, "registered", result.StatusMain)

	assert.Equal(t, "order-1", gotCreateInput.OrderReference, "must round-trip our own SalesOrder.ID for webhook correlation")
	assert.Equal(t, 1, gotCreateInput.Options.PackageType, "falls through to connection settings' default")
	assert.Nil(t, gotCreateInput.Options.InsuranceAmountMinorUnits, "not insured by default -- no sales order lookup, no amount")

	require.NotNil(t, created)
	assert.Equal(t, "tenant-1", created.TenantID)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.shipment.created", e.Subject)
	assert.Equal(t, "shipment-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"sales_order_id": "order-1", "connection_id": "conn-1", "carrier": float64(shippingDomain.CarrierPostNL),
		"external_shipment_id": "555", "barcode": "3SMYPA9", "package_type": float64(1), "insured": false,
	}, e.Data())
}

// TestCreateShipment_shipmentAlreadyExists_returnsExistingWithoutCallingMyParcel
// is a regression test for the duplicate-shipment bug: retrying
// "Create shipping label" for an order that already has a shipment on this
// connection must not create a real second MyParcel shipment.
func TestCreateShipment_shipmentAlreadyExists_returnsExistingWithoutCallingMyParcel(t *testing.T) {
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
	existing := shippingDomain.Shipment{ID: "shipment-existing", ConnectionID: "conn-1", SalesOrderID: "order-1", Barcode: "3SMYPA0"}
	createShipmentCalled := false
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			createShipmentCalled = true
			return integrationsDomain.CarrierShipmentResult{}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{
		findBySalesOrderIDFn: func(_ context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
			assert.Equal(t, "order-1", salesOrderID)
			return []shippingDomain.Shipment{existing}, nil
		},
	}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Equal(t, "shipment-existing", result.ID)
	assert.False(t, createShipmentCalled, "an existing shipment must not be re-created on MyParcel")
}

func TestCreateShipment_existingShipmentIsCancelled_createsNewOneInstead(t *testing.T) {
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
	cancelledAt := time.Now()
	cancelled := shippingDomain.Shipment{
		ID: "shipment-wrong-carrier", ConnectionID: "conn-1", SalesOrderID: "order-1",
		Barcode: "3SMYPA0", CancelledAt: &cancelledAt,
	}
	createShipmentCalled := false
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			createShipmentCalled = true
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "999", Barcode: "3SMYPA1"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{
		findBySalesOrderIDFn: func(_ context.Context, _ string) ([]shippingDomain.Shipment, error) {
			return []shippingDomain.Shipment{cancelled}, nil
		},
		createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "shipment-new"; return nil },
	}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Equal(t, "shipment-new", result.ID)
	assert.True(t, createShipmentCalled, "a cancelled shipment must not block creating a corrected one")
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	digitalStamp := shippingDomain.PackageTypeDigitalStamp
	sig := true
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
		PackageType: &digitalStamp, SignatureRequired: &sig,
	})

	require.NoError(t, err)
	assert.Equal(t, int(shippingDomain.PackageTypeDigitalStamp), gotCreateInput.Options.PackageType)
	// Only Mailbox package is confirmed to reject signature/insurance
	// (MyParcel 422 error 3705) -- Digital stamp isn't gated
	// (GateOptionsForPackageType), so the per-shipment signature request
	// must pass through unchanged.
	assert.True(t, gotCreateInput.Options.SignatureRequired)
}

// TestCreateShipment_perShipmentPackageTypeOverride_gatesOptionsFromConnectionDefault
// pins the real MyParcel 422 this codebase actually hit: a connection
// default with insurance (and therefore signature/only_recipient, see
// ResolveOptions) on, staff picking Mailbox package per-shipment with no
// other per-shipment override at all. ResolveOptions itself never sees
// PackageTypeMailboxPackage (its own gate only runs against the connection/
// carrier-layered PackageType) -- the request-level PackageType override is
// applied by CreateShipmentUseCase after ResolveOptions returns, so the
// gate must be re-run there too.
func TestCreateShipment_perShipmentPackageTypeOverride_gatesOptionsFromConnectionDefault(t *testing.T) {
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	mailbox := shippingDomain.PackageTypeMailboxPackage
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
		PackageType: &mailbox,
	})

	require.NoError(t, err)
	assert.Equal(t, int(shippingDomain.PackageTypeMailboxPackage), gotCreateInput.Options.PackageType)
	assert.False(t, gotCreateInput.Options.SignatureRequired)
	assert.Nil(t, gotCreateInput.Options.InsuranceAmountMinorUnits, "insurance must not be computed/sent for a non-Package box type")
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	var created *shippingDomain.Shipment
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; created = s; return nil }, findBySalesOrderIDFn: noExistingShipments}
	salesOrderRepo := &fakeSalesOrderTotalGetter{findByIDFn: func(_ context.Context, id string) (*omsDomain.SalesOrder, error) {
		assert.Equal(t, "order-1", id)
		return &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 50000}, nil
	}}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 4), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), salesOrderRepo, &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	require.NotNil(t, gotCreateInput.Options.InsuranceAmountMinorUnits)
	assert.Equal(t, 50000, *gotCreateInput.Options.InsuranceAmountMinorUnits, "insured for the order's own value, not a fixed configured amount")
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Nil(t, gotCreateInput.Options.InsuranceAmountMinorUnits, "carrier override turns insurance off despite the connection default")
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
	salesOrderRepo := &fakeSalesOrderTotalGetter{findByIDFn: func(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
		return &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 50000}, nil
	}}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 4), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("USD"), salesOrderRepo, &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", gotCreateInput.Options.Currency)
}

func TestCreateShipment_incompleteRecipient_returnsErrMissingRecipient(t *testing.T) {
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 0), &mocks.ConnectionRepo{}, &mocks.ShippingConnectionSettingsRepo{},
		&fakeShippingCarrierOverrideRepo{}, &fakeShipmentRepo{}, &fakeCarrierClientFactory{}, testEncSecret, &fakeRecordActivity{},
		tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

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
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			return integrationsDomain.CarrierShipmentResult{}, assert.AnError
		},
	}
	persisted := false
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { persisted = true; return nil }, findBySalesOrderIDFn: noExistingShipments}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

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
		client := &fakeCarrierClient{
			createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
				return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "3SMYPA1"}, nil
			},
		}
		shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
		activity := &fakeRecordActivity{}
		uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
			&fakeCarrierClientFactory{client: client}, testEncSecret, activity, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

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
		client := &fakeCarrierClient{
			createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
				return integrationsDomain.CarrierShipmentResult{}, assert.AnError
			},
		}
		shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { return nil }, findBySalesOrderIDFn: noExistingShipments}
		activity := &fakeRecordActivity{}
		uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, overrideRepo, shipmentRepo,
			&fakeCarrierClientFactory{client: client}, testEncSecret, activity, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

		ctx := tenant.WithTenantID(context.Background(), "tenant-1")
		_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
			SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
		})

		require.Error(t, err)
		require.Len(t, activity.calls, 1)
		assert.Equal(t, omsDomain.ActivityTypeShipmentFailed, activity.calls[0].ActivityType)
	})
}

func TestCreateShipment_exportOrderToMyParcelEnabled_alsoExportsOrder(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settings := &shippingDomain.ShippingConnectionSettings{ExportOrderToMyParcel: true, DefaultPackageType: shippingDomain.PackageTypePackage}
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
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "3SMYPA1"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}

	var gotExportIn appshipping.ExportMyParcelOrderInput
	exportCalled := false
	exporter := &fakeOrderExporter{executeFn: func(_ context.Context, in appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		exportCalled = true
		gotExportIn = in
		return &shippingDomain.MyParcelFulfilmentOrder{ID: "fo-1"}, nil
	}}

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), exporter, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
	})

	require.NoError(t, err)
	assert.True(t, exportCalled, "ExportOrderToMyParcel=true must trigger the order export")
	assert.Equal(t, "order-1", gotExportIn.SalesOrderID)
	assert.Equal(t, "conn-1", gotExportIn.ConnectionID)
}

func TestCreateShipment_orderExportFails_doesNotBlockShipmentCreation(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settings := &shippingDomain.ShippingConnectionSettings{ExportOrderToMyParcel: true, DefaultPackageType: shippingDomain.PackageTypePackage}
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
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "3SMYPA1"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}
	exporter := &fakeOrderExporter{executeFn: func(_ context.Context, _ appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		return nil, assert.AnError
	}}
	activity := &fakeRecordActivity{}

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, activity, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), exporter, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
	})

	require.NoError(t, err, "a failed order export must not fail shipment creation")
	require.NotNil(t, result)
	assert.Equal(t, "s-1", result.ID)
	found := false
	for _, c := range activity.calls {
		if c.ActivityType == "myparcel_order_export_failed" {
			found = true
		}
	}
	assert.True(t, found, "the export failure must still be recorded")
}

func TestCreateShipment_computesWeightFromLineItemSKUs(t *testing.T) {
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}

	sku1ID, sku2ID := "sku-1", "sku-2"
	weight1, weight2 := int64(250), int64(600)
	lineItemRepo := &fakeSalesOrderLineItemLister{listBySalesOrderFn: func(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
		return []omsDomain.SalesOrderLineItem{
			{ID: "li-1", SKUID: &sku1ID, Quantity: 2},
			{ID: "li-2", SKUID: &sku2ID, Quantity: 1},
		}, nil
	}}
	skuRepo := &fakeSKURepo{findByIDFn: func(_ context.Context, id string) (*pmsDomain.SKU, error) {
		switch id {
		case sku1ID:
			return &pmsDomain.SKU{ID: sku1ID, WeightGrams: &weight1}, nil
		case sku2ID:
			return &pmsDomain.SKU{ID: sku2ID, WeightGrams: &weight2}, nil
		default:
			t.Fatalf("unexpected SKU id %q", id)
			return nil, nil
		}
	}}

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, lineItemRepo, skuRepo, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	require.NotNil(t, gotCreateInput.WeightGrams, "2*250 + 1*600 = 1100g must be computed and sent")
	assert.Equal(t, 1100, *gotCreateInput.WeightGrams)
}

func TestCreateShipment_lineItemMissingSKUWeight_omitsWeightEntirely(t *testing.T) {
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
	var gotCreateInput integrationsDomain.CreateCarrierShipmentRequest
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			gotCreateInput = in
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "1", Barcode: "b"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{createFn: func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "s-1"; return nil }, findBySalesOrderIDFn: noExistingShipments}

	skuID := "sku-1"
	lineItemRepo := &fakeSalesOrderLineItemLister{listBySalesOrderFn: func(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
		return []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 1}}, nil
	}}
	skuRepo := &fakeSKURepo{findByIDFn: func(_ context.Context, _ string) (*pmsDomain.SKU, error) {
		return &pmsDomain.SKU{ID: skuID, WeightGrams: nil}, nil
	}}

	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), connRepo, settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"), notInsuredSalesOrderRepo(), &fakeOrderExporter{}, lineItemRepo, skuRepo, &eventrecordtest.Outbox{}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Nil(t, gotCreateInput.WeightGrams, "a SKU with no weight on file must not produce a wrong/partial total")
}
