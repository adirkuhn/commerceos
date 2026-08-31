package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

type fakeRouterHandler struct {
	called   bool
	envelope webhooks.Envelope
	err      error
}

func (f *fakeRouterHandler) Execute(_ context.Context, e webhooks.Envelope) error {
	f.called, f.envelope = true, e
	return f.err
}

func TestMyParcelWebhookRouter_shipmentStatusChange_routesToShipmentHandler(t *testing.T) {
	shipmentHandler := &fakeRouterHandler{}
	orderStatusHandler := &fakeRouterHandler{}
	router := appshipping.NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler)

	envelope := webhooks.Envelope{RawPayload: []byte(`{}`), Headers: map[string]string{"X-Myparcel-Hook": "shipment_status_change"}}
	err := router.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.True(t, shipmentHandler.called)
	assert.False(t, orderStatusHandler.called)
}

func TestMyParcelWebhookRouter_orderStatusChange_routesToOrderStatusHandler(t *testing.T) {
	shipmentHandler := &fakeRouterHandler{}
	orderStatusHandler := &fakeRouterHandler{}
	router := appshipping.NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler)

	envelope := webhooks.Envelope{RawPayload: []byte(`{}`), Headers: map[string]string{"X-Myparcel-Hook": "order_status_change"}}
	err := router.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.False(t, shipmentHandler.called)
	assert.True(t, orderStatusHandler.called)
}

func TestMyParcelWebhookRouter_unrecognizedTopic_isNoOp(t *testing.T) {
	shipmentHandler := &fakeRouterHandler{}
	orderStatusHandler := &fakeRouterHandler{}
	router := appshipping.NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler)

	envelope := webhooks.Envelope{RawPayload: []byte(`{}`), Headers: map[string]string{"X-Myparcel-Hook": "shop_updated"}}
	err := router.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.False(t, shipmentHandler.called)
	assert.False(t, orderStatusHandler.called)
}

func TestMyParcelWebhookRouter_nilHandlerForTopic_isNoOp(t *testing.T) {
	router := appshipping.NewMyParcelWebhookRouter(nil, nil)

	envelope := webhooks.Envelope{RawPayload: []byte(`{}`), Headers: map[string]string{"X-Myparcel-Hook": "order_status_change"}}
	err := router.Execute(context.Background(), envelope)

	require.NoError(t, err)
}

func TestMyParcelWebhookRouter_handlerError_propagates(t *testing.T) {
	shipmentHandler := &fakeRouterHandler{err: errors.New("boom")}
	router := appshipping.NewMyParcelWebhookRouter(shipmentHandler, nil)

	envelope := webhooks.Envelope{RawPayload: []byte(`{}`), Headers: map[string]string{"X-Myparcel-Hook": "shipment_status_change"}}
	err := router.Execute(context.Background(), envelope)

	require.Error(t, err)
	assert.Equal(t, "boom", err.Error())
}
