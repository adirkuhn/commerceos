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
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestExportMyParcelOrder_happyPath_createsAndPersistsAndRecordsActivity(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	accountID, shopID := 111, 222
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{MyParcelAccountID: &accountID, MyParcelShopID: &shopID}, nil
		},
	}
	var gotCreateOrderInput integrationsDomain.CreateOrderInput
	client := &fakeMyParcelClient{
		createOrderFn: func(_ context.Context, in integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
			gotCreateOrderInput = in
			return integrationsDomain.OrderResult{UUID: "order-uuid-1", Status: "open"}, nil
		},
	}
	var created *shippingDomain.MyParcelFulfilmentOrder
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{
		findByConnectionAndSalesOrderFn: func(_ context.Context, _, _ string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return nil, nil },
		createFn: func(_ context.Context, o *shippingDomain.MyParcelFulfilmentOrder) error {
			o.ID = "fulfilment-1"
			created = o
			return nil
		},
	}
	activity := &fakeRecordActivity{}

	uc := appshipping.NewExportMyParcelOrderUseCase(newMockGormDB(t, 2), connRepo, settingsRepo, fulfilmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, activity)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.ExportMyParcelOrderInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "fulfilment-1", result.ID)
	assert.Equal(t, "order-uuid-1", result.MyParcelUUID)

	assert.Equal(t, "order-1", gotCreateOrderInput.ExternalIdentifier)
	assert.Equal(t, 111, gotCreateOrderInput.AccountID)
	assert.Equal(t, 222, gotCreateOrderInput.ShopID)

	require.NotNil(t, created)
	assert.Equal(t, "tenant-1", created.TenantID)

	require.Len(t, activity.calls, 1)
	assert.Equal(t, "myparcel_order_exported", activity.calls[0].ActivityType)
	assert.Equal(t, "user-1", activity.calls[0].UserID)
}

func TestExportMyParcelOrder_alreadyExported_returnsExistingWithoutCallingMyParcel(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}
	existing := &shippingDomain.MyParcelFulfilmentOrder{ID: "fulfilment-1", MyParcelUUID: "order-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{
		findByConnectionAndSalesOrderFn: func(_ context.Context, _, _ string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
			return existing, nil
		},
	}
	createOrderCalled := false
	client := &fakeMyParcelClient{
		createOrderFn: func(_ context.Context, _ integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
			createOrderCalled = true
			return integrationsDomain.OrderResult{}, nil
		},
	}
	uc := appshipping.NewExportMyParcelOrderUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, fulfilmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.ExportMyParcelOrderInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	require.NoError(t, err)
	assert.Equal(t, "fulfilment-1", result.ID)
	assert.False(t, createOrderCalled, "an already-exported order must not be re-exported to MyParcel")
}

func TestExportMyParcelOrder_incompleteRecipient_returnsErrMissingRecipient(t *testing.T) {
	uc := appshipping.NewExportMyParcelOrderUseCase(newMockGormDB(t, 0), nil, nil, nil, nil, testEncSecret, nil)

	_, err := uc.Execute(context.Background(), appshipping.ExportMyParcelOrderInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Recipient: appshipping.RecipientInput{},
	})

	assert.ErrorIs(t, err, shippingDomain.ErrMissingRecipient)
}

func TestExportMyParcelOrder_myParcelAPIError_recordsFailureActivity(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
	}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{
		findByConnectionAndSalesOrderFn: func(_ context.Context, _, _ string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return nil, nil },
	}
	client := &fakeMyParcelClient{
		createOrderFn: func(_ context.Context, _ integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
			return integrationsDomain.OrderResult{}, assert.AnError
		},
	}
	activity := &fakeRecordActivity{}
	uc := appshipping.NewExportMyParcelOrderUseCase(newMockGormDB(t, 1), connRepo, settingsRepo, fulfilmentRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, activity)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.ExportMyParcelOrderInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
	})

	require.Error(t, err)
	require.Len(t, activity.calls, 1)
	assert.Equal(t, "myparcel_error", activity.calls[0].ActivityType)
}
