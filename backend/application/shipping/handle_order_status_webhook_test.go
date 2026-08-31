package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func myParcelOrderHookPayload(order, status string) []byte {
	return []byte(`{"data":{"hooks":[{"order":"` + order + `","status":"` + status + `"}]}}`)
}

func TestHandleOrderStatusWebhook_processed_wooOrder_movesToOnHold(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(_ context.Context, uuid string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		require.Equal(t, "mp-uuid-1", uuid)
		return fulfilment, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 3), fulfilmentRepo, salesOrderRepo, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusOnHold, updater.calls[0].status)
}

func TestHandleOrderStatusWebhook_processed_nonWooOrder_movesToShipped(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusPending}
	updater := &fakeSalesOrderStatusUpdater{}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 3), fulfilmentRepo, salesOrderRepo, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestHandleOrderStatusWebhook_processed_orderAlreadyPastPending_isNoOp(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusCompleted}
	updater := &fakeSalesOrderStatusUpdater{}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 2), fulfilmentRepo, salesOrderRepo, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.Empty(t, updater.calls, "an order already past Pending/Processing must not be moved backward or reapplied")
}

func TestHandleOrderStatusWebhook_openStatus_isNoOp(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &fakeSalesOrderRepo{}, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "open")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.Empty(t, updater.calls)
}

func TestHandleOrderStatusWebhook_unknownMyParcelUUID_isSkippedNotAnError(t *testing.T) {
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return nil, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &fakeSalesOrderRepo{}, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-unknown", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err, "a MyParcel order uuid this tenant never exported must not fail/retry forever")
	assert.Empty(t, updater.calls)
}

func TestHandleOrderStatusWebhook_unrecognizedStatus_returnsTypedError(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &fakeSalesOrderRepo{}, &fakeSalesOrderStatusUpdater{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "invoiced")}
	err := uc.Execute(context.Background(), envelope)

	require.Error(t, err)
	assert.ErrorIs(t, err, appshipping.ErrUnrecognizedMyParcelOrderStatus)
}

func TestHandleOrderStatusWebhook_repoError_propagates(t *testing.T) {
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		return nil, errors.New("db down")
	}}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &fakeSalesOrderRepo{}, &fakeSalesOrderStatusUpdater{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}
