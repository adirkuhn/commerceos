package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func newPushSalesOrderStatusUseCase(t *testing.T, n int, salesOrderRepo *stubOrdersSalesOrderRepo, connRepo *stubConnectionRepo, wooFactory *stubWooClientFactory) *appchannels.PushSalesOrderStatusToChannelsUseCase {
	t.Helper()
	return newPushSalesOrderStatusUseCaseWithShipments(t, n, salesOrderRepo, &stubShipmentRepo{}, connRepo, wooFactory)
}

func newPushSalesOrderStatusUseCaseWithShipments(t *testing.T, n int, salesOrderRepo *stubOrdersSalesOrderRepo, shipmentRepo *stubShipmentRepo, connRepo *stubConnectionRepo, wooFactory *stubWooClientFactory) *appchannels.PushSalesOrderStatusToChannelsUseCase {
	t.Helper()
	return appchannels.NewPushSalesOrderStatusToChannelsUseCase(newMockGormDB(t, n), salesOrderRepo, shipmentRepo, connRepo, wooFactory, testEncryptionSecret)
}

func TestPushSalesOrderStatusToChannels_wooOrder_pushesMappedStatus(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCase(t, 2, salesOrderRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, wooClient.updateOrderStatusCalls, 1)
	assert.Equal(t, "501", wooClient.updateOrderStatusCalls[0].externalOrderID)
	assert.Equal(t, "on-hold", wooClient.updateOrderStatusCalls[0].status)
}

func TestPushSalesOrderStatusToChannels_onHoldFulfilledWooOrder_attachesShipmentNote(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold, Fulfilled: true,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	shipmentRepo := &stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{
		"so-1": {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1"}},
	}}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 3, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, wooClient.addOrderNoteCalls, 1)
	assert.Equal(t, "501", wooClient.addOrderNoteCalls[0].externalOrderID)
	assert.Equal(t, "Shipped via PostNL, tracking 3SMYPA1", wooClient.addOrderNoteCalls[0].note)
}

func TestPushSalesOrderStatusToChannels_onHoldNotFulfilled_doesNotAttachNote(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold, Fulfilled: false,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	shipmentRepo := &stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{
		"so-1": {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1"}},
	}}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 2, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Empty(t, wooClient.addOrderNoteCalls, "on-hold for an unrelated reason (not a fulfillment) must not get a shipment note")
}

func TestPushSalesOrderStatusToChannels_noteFailure_doesNotFailThePush(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold, Fulfilled: true,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	shipmentRepo := &stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{
		"so-1": {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1"}},
	}}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{addOrderNoteErr: errors.New("woo unreachable")}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 3, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err, "the status update already succeeded -- a note failure is best-effort, logged not returned")
}

func TestPushSalesOrderStatusToChannels_nonWooSource_isNoOp(t *testing.T) {
	order := &omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusCompleted}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCase(t, 1, salesOrderRepo, &stubConnectionRepo{}, wooFactory)

	err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Empty(t, wooClient.updateOrderStatusCalls)
}

func TestPushSalesOrderStatusToChannels_statusWithNoWooEquivalent_isNoOp(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCase(t, 1, salesOrderRepo, &stubConnectionRepo{}, wooFactory)

	err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err, "shipped has no Woo equivalent -- skipped, not an error")
	assert.Empty(t, wooClient.updateOrderStatusCalls)
}

func TestPushSalesOrderStatusToChannels_wooOrderWithNoConnectionID_isNoOp(t *testing.T) {
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: nil, Status: omsDomain.SalesOrderStatusOnHold,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCase(t, 1, salesOrderRepo, &stubConnectionRepo{}, wooFactory)

	err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Empty(t, wooClient.updateOrderStatusCalls)
}

func TestPushSalesOrderStatusToChannels_propagatesWooClientError(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusCompleted,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{updateOrderStatusErr: errors.New("woo unreachable")}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCase(t, 2, salesOrderRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "woo unreachable")
}

func TestPushSalesOrderStatusToChannels_propagatesSalesOrderNotFound(t *testing.T) {
	salesOrderRepo := &stubOrdersSalesOrderRepo{}
	uc := newPushSalesOrderStatusUseCase(t, 1, salesOrderRepo, &stubConnectionRepo{}, &stubWooClientFactory{})

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}
