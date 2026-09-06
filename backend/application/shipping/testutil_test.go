package shipping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy n
// separate tenant.WithTenant Begin/set_config/Commit sequences -- same
// pattern as application/channels' own test helper.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

// --- MyParcelClient/MyParcelClientFactory fakes ---

type fakeMyParcelClient struct {
	createShipmentFn            func(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error)
	createOrderFn               func(ctx context.Context, in integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error)
	getLabelFn                  func(ctx context.Context, myParcelShipmentID int64, format, size string) (integrationsDomain.LabelResult, error)
	getTrackTraceFn             func(ctx context.Context, myParcelShipmentID int64) (integrationsDomain.TrackTraceResult, error)
	getShipmentBarcodeFn        func(ctx context.Context, myParcelShipmentID int64) (string, error)
	registerWebhookFn           func(ctx context.Context, callbackURL string) error
	listWebhookSubscriptionsFn  func(ctx context.Context) ([]integrationsDomain.WebhookSubscription, error)
	createWebhookSubscriptionFn func(ctx context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error)
	deleteWebhookSubscriptionFn func(ctx context.Context, id string) error
	deleteShipmentFn            func(ctx context.Context, myParcelShipmentID int64) error
	getDeliveryPriceEstimateFn  func(ctx context.Context, in integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error)
}

func (f *fakeMyParcelClient) CreateShipment(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
	return f.createShipmentFn(ctx, in)
}
func (f *fakeMyParcelClient) CreateOrder(ctx context.Context, in integrationsDomain.CreateOrderInput) (integrationsDomain.OrderResult, error) {
	return f.createOrderFn(ctx, in)
}
func (f *fakeMyParcelClient) GetLabel(ctx context.Context, id int64, format, size string) (integrationsDomain.LabelResult, error) {
	return f.getLabelFn(ctx, id, format, size)
}
func (f *fakeMyParcelClient) GetTrackTrace(ctx context.Context, id int64) (integrationsDomain.TrackTraceResult, error) {
	return f.getTrackTraceFn(ctx, id)
}
func (f *fakeMyParcelClient) GetShipmentBarcode(ctx context.Context, id int64) (string, error) {
	if f.getShipmentBarcodeFn == nil {
		return "", nil
	}
	return f.getShipmentBarcodeFn(ctx, id)
}
func (f *fakeMyParcelClient) RegisterWebhook(ctx context.Context, callbackURL string) error {
	return f.registerWebhookFn(ctx, callbackURL)
}
func (f *fakeMyParcelClient) ListWebhookSubscriptions(ctx context.Context) ([]integrationsDomain.WebhookSubscription, error) {
	if f.listWebhookSubscriptionsFn == nil {
		return nil, nil
	}
	return f.listWebhookSubscriptionsFn(ctx)
}
func (f *fakeMyParcelClient) CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (integrationsDomain.WebhookSubscription, error) {
	if f.createWebhookSubscriptionFn == nil {
		return integrationsDomain.WebhookSubscription{}, nil
	}
	return f.createWebhookSubscriptionFn(ctx, hook, deliveryURL)
}
func (f *fakeMyParcelClient) DeleteWebhookSubscription(ctx context.Context, id string) error {
	if f.deleteWebhookSubscriptionFn == nil {
		return nil
	}
	return f.deleteWebhookSubscriptionFn(ctx, id)
}
func (f *fakeMyParcelClient) DeleteShipment(ctx context.Context, myParcelShipmentID int64) error {
	if f.deleteShipmentFn == nil {
		return nil
	}
	return f.deleteShipmentFn(ctx, myParcelShipmentID)
}
func (f *fakeMyParcelClient) GetDeliveryPriceEstimate(ctx context.Context, in integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
	if f.getDeliveryPriceEstimateFn == nil {
		return nil, nil
	}
	return f.getDeliveryPriceEstimateFn(ctx, in)
}

type fakeMyParcelClientFactory struct {
	client       integrationsDomain.MyParcelClient
	newClientErr error
}

func (f *fakeMyParcelClientFactory) NewClient(apiKey string) (integrationsDomain.MyParcelClient, error) {
	if f.newClientErr != nil {
		return nil, f.newClientErr
	}
	return f.client, nil
}

// --- ShipmentRepository fake ---

// noExistingShipments is the common fakeShipmentRepo.findBySalesOrderIDFn
// for every CreateShipmentUseCase test that isn't specifically about the
// idempotency check -- CreateShipmentUseCase.Execute now always looks this
// up before creating (see TestCreateShipment_shipmentAlreadyExists_returns
// ExistingWithoutCallingMyParcel for the other branch).
func noExistingShipments(_ context.Context, _ string) ([]shippingDomain.Shipment, error) {
	return nil, nil
}

type fakeShipmentRepo struct {
	createFn                   func(ctx context.Context, s *shippingDomain.Shipment) error
	findByIDFn                 func(ctx context.Context, id string) (*shippingDomain.Shipment, error)
	findByMyParcelShipmentIDFn func(ctx context.Context, id int64) (*shippingDomain.Shipment, error)
	findBySalesOrderIDFn       func(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error)
	findStaleNonTerminalFn     func(ctx context.Context, olderThan time.Time, limit int) ([]shippingDomain.Shipment, error)
	updateStatusFn             func(ctx context.Context, id string, status int, statusMain, trackingURL, barcode string) error
	markCancelledFn            func(ctx context.Context, id string, cancelledAt time.Time, note string) error
}

func (f *fakeShipmentRepo) Create(ctx context.Context, s *shippingDomain.Shipment) error {
	return f.createFn(ctx, s)
}
func (f *fakeShipmentRepo) FindByID(ctx context.Context, id string) (*shippingDomain.Shipment, error) {
	return f.findByIDFn(ctx, id)
}
func (f *fakeShipmentRepo) FindByMyParcelShipmentID(ctx context.Context, id int64) (*shippingDomain.Shipment, error) {
	return f.findByMyParcelShipmentIDFn(ctx, id)
}
func (f *fakeShipmentRepo) FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	return f.findBySalesOrderIDFn(ctx, salesOrderID)
}
func (f *fakeShipmentRepo) FindStaleNonTerminal(ctx context.Context, olderThan time.Time, limit int) ([]shippingDomain.Shipment, error) {
	return f.findStaleNonTerminalFn(ctx, olderThan, limit)
}
func (f *fakeShipmentRepo) UpdateStatus(ctx context.Context, id string, status int, statusMain, trackingURL, barcode string) error {
	return f.updateStatusFn(ctx, id, status, statusMain, trackingURL, barcode)
}
func (f *fakeShipmentRepo) MarkCancelled(ctx context.Context, id string, cancelledAt time.Time, note string) error {
	if f.markCancelledFn == nil {
		return nil
	}
	return f.markCancelledFn(ctx, id, cancelledAt, note)
}

// --- ShipmentStatusEventRepository fake ---

type fakeShipmentStatusEventRepo struct {
	appendFn           func(ctx context.Context, e *shippingDomain.ShipmentStatusEvent) error
	listByShipmentIDFn func(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error)
}

func (f *fakeShipmentStatusEventRepo) Append(ctx context.Context, e *shippingDomain.ShipmentStatusEvent) error {
	return f.appendFn(ctx, e)
}
func (f *fakeShipmentStatusEventRepo) ListByShipmentID(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error) {
	if f.listByShipmentIDFn == nil {
		return nil, nil
	}
	return f.listByShipmentIDFn(ctx, shipmentID)
}

// --- webhooks.WebhookEventRepository fake (narrow -- only List and
// MarkDispatchResult are exercised by BackfillShipmentWebhookDispatchUseCase) ---

type fakeWebhookEventRepoForBackfill struct {
	events            []webhooksDomain.WebhookEvent
	listErr           error
	findByIDErr       error
	markDispatchCalls []backfillMarkDispatchCall
	markDispatchErr   error
}

type backfillMarkDispatchCall struct {
	id            string
	status        webhooksDomain.WebhookEventStatus
	dispatchError string
}

func (f *fakeWebhookEventRepoForBackfill) Save(context.Context, *webhooksDomain.WebhookEvent) error {
	panic("not used")
}
func (f *fakeWebhookEventRepoForBackfill) DistinctTopics(context.Context) ([]webhooksDomain.TopicRef, error) {
	panic("not used")
}
func (f *fakeWebhookEventRepoForBackfill) List(_ context.Context, provider string, limit int) ([]webhooksDomain.WebhookEvent, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.events, nil
}
func (f *fakeWebhookEventRepoForBackfill) MarkDispatchResult(_ context.Context, id string, status webhooksDomain.WebhookEventStatus, dispatchError string) error {
	f.markDispatchCalls = append(f.markDispatchCalls, backfillMarkDispatchCall{id: id, status: status, dispatchError: dispatchError})
	// Keep the in-memory row consistent with what a real repo would show on
	// a subsequent FindByID -- RetryWebhookEventDispatchUseCase re-fetches
	// after dispatching to return the caller the row's current state.
	for i := range f.events {
		if f.events[i].ID == id {
			f.events[i].Status, f.events[i].DispatchError = status, dispatchError
		}
	}
	return f.markDispatchErr
}

func (f *fakeWebhookEventRepoForBackfill) FindByID(_ context.Context, id string) (*webhooksDomain.WebhookEvent, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	for _, e := range f.events {
		if e.ID == id {
			evt := e
			return &evt, nil
		}
	}
	return nil, webhooksDomain.ErrWebhookEventNotFound
}

// --- AddressLookupClient fake ---

type fakeAddressLookupClient struct {
	result *shippingDomain.AddressLookupResult
	err    error
}

func (f *fakeAddressLookupClient) Lookup(ctx context.Context, postalCode, houseNumber string) (*shippingDomain.AddressLookupResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// --- ShippingCarrierOverrideRepository fake ---

type fakeShippingCarrierOverrideRepo struct {
	listByConnectionFn           func(ctx context.Context, connectionID string) ([]shippingDomain.ShippingCarrierOverride, error)
	findByConnectionAndCarrierFn func(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error)
	upsertFn                     func(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error
}

func (f *fakeShippingCarrierOverrideRepo) ListByConnection(ctx context.Context, connectionID string) ([]shippingDomain.ShippingCarrierOverride, error) {
	return f.listByConnectionFn(ctx, connectionID)
}
func (f *fakeShippingCarrierOverrideRepo) FindByConnectionAndCarrier(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
	return f.findByConnectionAndCarrierFn(ctx, connectionID, carrier)
}
func (f *fakeShippingCarrierOverrideRepo) Upsert(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error {
	return f.upsertFn(ctx, o)
}

// --- MyParcelFulfilmentOrderRepository fake ---

type fakeMyParcelFulfilmentOrderRepo struct {
	createFn                        func(ctx context.Context, o *shippingDomain.MyParcelFulfilmentOrder) error
	findByConnectionAndSalesOrderFn func(ctx context.Context, connectionID, salesOrderID string) (*shippingDomain.MyParcelFulfilmentOrder, error)
	findByMyParcelUUIDFn            func(ctx context.Context, myParcelUUID string) (*shippingDomain.MyParcelFulfilmentOrder, error)
}

func (f *fakeMyParcelFulfilmentOrderRepo) Create(ctx context.Context, o *shippingDomain.MyParcelFulfilmentOrder) error {
	return f.createFn(ctx, o)
}
func (f *fakeMyParcelFulfilmentOrderRepo) FindByConnectionAndSalesOrder(ctx context.Context, connectionID, salesOrderID string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	return f.findByConnectionAndSalesOrderFn(ctx, connectionID, salesOrderID)
}
func (f *fakeMyParcelFulfilmentOrderRepo) FindByMyParcelUUID(ctx context.Context, myParcelUUID string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	if f.findByMyParcelUUIDFn == nil {
		return nil, nil
	}
	return f.findByMyParcelUUIDFn(ctx, myParcelUUID)
}

// --- recordActivityUC fake ---

type fakeRecordActivity struct {
	calls []appoms.RecordSalesOrderActivityInput
	err   error
}

func (f *fakeRecordActivity) Execute(_ context.Context, in appoms.RecordSalesOrderActivityInput) (*omsDomain.SalesOrderActivity, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return &omsDomain.SalesOrderActivity{ID: "activity-1"}, nil
}

// --- myParcelOrderExporter fake -- zero value panics if Execute is called,
// proving CreateShipmentUseCase skips it when the connection's
// ExportOrderToMyParcel setting is off (the default in every existing test
// that doesn't set it). ---

type fakeOrderExporter struct {
	executeFn func(ctx context.Context, in appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error)
}

func (f *fakeOrderExporter) Execute(ctx context.Context, in appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	if f.executeFn == nil {
		panic("fakeOrderExporter: Execute called but ExportOrderToMyParcel should have been off")
	}
	return f.executeFn(ctx, in)
}

// --- salesOrderTotalGetter fake (narrow -- just what CreateShipmentUseCase
// needs to compute the insured amount from the order's own value) ---

type fakeSalesOrderTotalGetter struct {
	findByIDFn func(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

func (f *fakeSalesOrderTotalGetter) FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error) {
	return f.findByIDFn(ctx, id)
}

// --- salesOrderLineItemLister fake (narrow -- just what
// ExportMyParcelOrderUseCase needs to build MyParcel's required
// order_lines[]) ---

type fakeSalesOrderLineItemLister struct {
	listBySalesOrderFn func(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error)
}

func (f *fakeSalesOrderLineItemLister) ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error) {
	if f.listBySalesOrderFn == nil {
		return nil, nil
	}
	return f.listBySalesOrderFn(ctx, salesOrderID)
}

// --- skuWeightGetter fake -- zero value returns no line items above, so
// CreateShipmentUseCase's weight computation resolves to "unknown" (nil)
// by default in every test that doesn't care about it. ---

type fakeSKURepo struct {
	findByIDFn func(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

func (f *fakeSKURepo) FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error) {
	if f.findByIDFn == nil {
		return nil, errors.New("fakeSKURepo: FindByID called unexpectedly")
	}
	return f.findByIDFn(ctx, id)
}

// --- settings.Repository fake (tenant-wide Localization settings) ---

type fakeTenantSettingsRepo struct {
	getFn func(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

func (f *fakeTenantSettingsRepo) Get(ctx context.Context) (*settingsDomain.TenantSettings, error) {
	return f.getFn(ctx)
}
func (f *fakeTenantSettingsRepo) Upsert(context.Context, *settingsDomain.TenantSettings) error {
	return nil
}
// tenantSettingsRepoWithCurrency is the terse default most CreateShipment
// tests reach for -- they only care that some currency reaches
// MyParcelShipmentOptions.Currency, not the exact value; tests asserting
// the currency threading itself build their own fakeTenantSettingsRepo.
func tenantSettingsRepoWithCurrency(currency string) *fakeTenantSettingsRepo {
	return &fakeTenantSettingsRepo{getFn: func(context.Context) (*settingsDomain.TenantSettings, error) {
		return &settingsDomain.TenantSettings{DefaultCurrency: currency}, nil
	}}
}

// --- oms.SalesOrderRepository fake (only Update is exercised by this
// package -- the rest are no-ops satisfying the interface) ---

type fakeSalesOrderRepo struct {
	updateFn   func(ctx context.Context, id string, fields map[string]any) error
	findByIDFn func(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

func (f *fakeSalesOrderRepo) Create(context.Context, *omsDomain.SalesOrder) error { return nil }
func (f *fakeSalesOrderRepo) FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error) {
	if f.findByIDFn == nil {
		return nil, nil
	}
	return f.findByIDFn(ctx, id)
}
func (f *fakeSalesOrderRepo) FindByNumber(context.Context, string) (*omsDomain.SalesOrder, error) {
	return nil, nil
}
func (f *fakeSalesOrderRepo) FindByExternalID(context.Context, omsDomain.SalesOrderSource, string) (*omsDomain.SalesOrder, error) {
	return nil, nil
}
func (f *fakeSalesOrderRepo) List(context.Context, omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	return nil, 0, nil
}
func (f *fakeSalesOrderRepo) Search(context.Context, string, int) ([]omsDomain.SalesOrder, error) {
	return nil, nil
}
func (f *fakeSalesOrderRepo) ListByConnection(context.Context, string) ([]omsDomain.SalesOrder, error) {
	return nil, nil
}
func (f *fakeSalesOrderRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	return f.updateFn(ctx, id, fields)
}
func (f *fakeSalesOrderRepo) TrySetStockDecremented(context.Context, string, bool, bool) (bool, error) {
	return false, nil
}
func (f *fakeSalesOrderRepo) Delete(context.Context, string) error { return nil }

// --- salesOrderStatusUpdater fake (narrow -- what completeOrderOnDelivery /
// HandleOrderStatusWebhookUseCase need from *appoms.UpdateSalesOrderUseCase) ---

type fakeSalesOrderStatusUpdater struct {
	calls     []fakeSalesOrderStatusUpdateCall
	executeFn func(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}

type fakeSalesOrderStatusUpdateCall struct {
	id     string
	status omsDomain.SalesOrderStatus
}

func (f *fakeSalesOrderStatusUpdater) Execute(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	var status omsDomain.SalesOrderStatus
	if in.Status != nil {
		status = *in.Status
	}
	f.calls = append(f.calls, fakeSalesOrderStatusUpdateCall{id: id, status: status})
	if f.executeFn != nil {
		return f.executeFn(ctx, id, in)
	}
	return &omsDomain.SalesOrder{ID: id, Status: status}, nil
}
