package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// The event is part of the write: when it can't be recorded, the use case
// fails instead of leaving an unannounced change behind.

var errOutboxDown = errors.New("outbox down")

func shippingConnRepo(t *testing.T) *mocks.ConnectionRepo {
	conn := &integrationsDomain.Connection{ID: "conn-1", TenantID: "tenant-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	return &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
}

func TestCreateShipment_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{
				DefaultPackageType: shippingDomain.PackageTypePackage, DefaultLabelFormat: shippingDomain.LabelFormatPDF,
			}, nil
		},
	}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: func(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
			return nil, nil
		},
	}
	client := &fakeCarrierClient{
		createShipmentFn: func(_ context.Context, _ integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
			return integrationsDomain.CarrierShipmentResult{ExternalShipmentID: "555", Barcode: "3SMYPA9"}, nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{
		createFn:             func(_ context.Context, s *shippingDomain.Shipment) error { s.ID = "shipment-1"; return nil },
		findBySalesOrderIDFn: noExistingShipments,
	}
	uc := appshipping.NewCreateShipmentUseCase(newMockGormDB(t, 3), shippingConnRepo(t), settingsRepo, overrideRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, tenantSettingsRepoWithCurrency("EUR"),
		notInsuredSalesOrderRepo(), &fakeOrderExporter{}, &fakeSalesOrderLineItemLister{}, &fakeSKURepo{}, &eventrecordtest.Outbox{Err: errOutboxDown}, &fakeCompanyProfiles{}, &fakeShipmentLabelRepo{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.CreateShipmentInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(),
	})

	assert.ErrorIs(t, err, errOutboxDown)
}

func TestExportMyParcelOrder_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	accountID, shopID := 111, 222
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{MyParcelAccountID: &accountID, MyParcelShopID: &shopID}, nil
		},
	}
	client := &fakeMyParcelClient{
		createOrderFn: func(_ context.Context, _ integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
			return integrationsDomain.OrderResult{UUID: "order-uuid-1", Status: "open"}, nil
		},
	}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{
		findByConnectionAndSalesOrderFn: func(_ context.Context, _, _ string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return nil, nil },
		createFn: func(_ context.Context, o *shippingDomain.MyParcelFulfilmentOrder) error {
			o.ID = "fulfilment-1"
			return nil
		},
	}
	salesOrderRepo := &fakeSalesOrderTotalGetter{findByIDFn: func(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
		return &omsDomain.SalesOrder{ID: "order-1", Number: "1001", TotalMinorUnits: 4000, TaxTotalMinorUnits: 700}, nil
	}}
	lineItemRepo := &fakeSalesOrderLineItemLister{listBySalesOrderFn: func(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
		return []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUCode: "WIDGET-1", Name: "Widget", Quantity: 2, TotalMinorUnits: 2000}}, nil
	}}
	uc := appshipping.NewExportMyParcelOrderUseCase(newMockGormDB(t, 2), shippingConnRepo(t), settingsRepo, fulfilmentRepo,
		salesOrderRepo, lineItemRepo, &fakeMyParcelClientFactory{client: client}, testEncSecret, &fakeRecordActivity{},
		&eventrecordtest.Outbox{Err: errOutboxDown})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.ExportMyParcelOrderInput{
		SalesOrderID: "order-1", ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, Recipient: validRecipient(), UserID: "user-1",
	})

	assert.ErrorIs(t, err, errOutboxDown)
}

func TestRetryWebhookEventDispatch_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "shipment_status_change", EventID: "evt-1",
			Payload: myParcelHookPayload(555, 3, "3SMYPA1"), Status: webhooksDomain.WebhookEventStatusFailed, DispatchError: "myparcel unreachable"},
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", ExternalShipmentID: "555", Status: 1, StatusMain: "registered"}
	shipmentRepo := &fakeShipmentRepo{
		findByExternalShipmentIDFn: func(context.Context, string) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(context.Context, string, int, string, string, string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(context.Context, *shippingDomain.ShipmentStatusEvent) error { return nil }}
	client := &fakeCarrierClient{getTrackTraceFn: func(context.Context, string) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "handed_to_carrier"}, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, shippingConnRepo(t), salesOrderRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, nil, &fakeSalesOrderStatusUpdater{}, nil, &stubOutboxRepo{err: errOutboxDown}, nil)

	_, err := uc.Execute(context.Background(), "evt-row-1")

	assert.ErrorIs(t, err, errOutboxDown)
}

func TestHandleOrderStatusWebhook_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(_ context.Context, _ string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		return &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}, nil
	}}
	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 2), fulfilmentRepo, &stubOutboxRepo{err: errOutboxDown})

	err := uc.Execute(context.Background(), webhooksDomain.Envelope{
		TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed"),
	})

	assert.ErrorIs(t, err, errOutboxDown)
}
