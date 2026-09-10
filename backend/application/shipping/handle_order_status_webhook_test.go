package shipping_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/domain/events"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func myParcelOrderHookPayload(order, status string) []byte {
	return []byte(`{"data":{"hooks":[{"order":"` + order + `","status":"` + status + `"}]}}`)
}

func TestHandleOrderStatusWebhook_processed_enqueuesFulfillmentSignal(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(_ context.Context, uuid string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		require.Equal(t, "mp-uuid-1", uuid)
		return fulfilment, nil
	}}
	outboxRepo := &stubOutboxRepo{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 2), fulfilmentRepo, outboxRepo)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, outboxRepo.calls, 1)
	assert.Equal(t, events.ShipmentFulfillmentSignal{}.EventName(), outboxRepo.calls[0].subject)
	var event events.ShipmentFulfillmentSignal
	require.NoError(t, json.Unmarshal(outboxRepo.calls[0].payload, &event))
	assert.Equal(t, "order-1", event.SalesOrderID)
	assert.Equal(t, events.ShipmentFulfillmentSignalKindOrderProcessed, event.Kind)
}

func TestHandleOrderStatusWebhook_openStatus_isNoOp(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}
	outboxRepo := &stubOutboxRepo{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, outboxRepo)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "open")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.Empty(t, outboxRepo.calls)
}

func TestHandleOrderStatusWebhook_unknownMyParcelUUID_isSkippedNotAnError(t *testing.T) {
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return nil, nil }}
	outboxRepo := &stubOutboxRepo{}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, outboxRepo)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-unknown", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err, "a MyParcel order uuid this tenant never exported must not fail/retry forever")
	assert.Empty(t, outboxRepo.calls)
}

func TestHandleOrderStatusWebhook_unrecognizedStatus_returnsTypedError(t *testing.T) {
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &stubOutboxRepo{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "invoiced")}
	err := uc.Execute(context.Background(), envelope)

	require.Error(t, err)
	assert.ErrorIs(t, err, appshipping.ErrUnrecognizedMyParcelOrderStatus)
}

func TestHandleOrderStatusWebhook_repoError_propagates(t *testing.T) {
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) {
		return nil, errors.New("db down")
	}}

	uc := appshipping.NewHandleOrderStatusWebhookUseCase(newMockGormDB(t, 1), fulfilmentRepo, &stubOutboxRepo{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelOrderHookPayload("mp-uuid-1", "processed")}
	err := uc.Execute(context.Background(), envelope)

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}
