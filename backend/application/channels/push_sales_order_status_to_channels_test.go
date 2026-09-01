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
	return newPushSalesOrderStatusUseCaseWithOverrides(t, n, salesOrderRepo, shipmentRepo, &stubCarrierOverrideRepo{}, connRepo, wooFactory)
}

func newPushSalesOrderStatusUseCaseWithOverrides(t *testing.T, n int, salesOrderRepo *stubOrdersSalesOrderRepo, shipmentRepo *stubShipmentRepo, overrideRepo *stubCarrierOverrideRepo, connRepo *stubConnectionRepo, wooFactory *stubWooClientFactory) *appchannels.PushSalesOrderStatusToChannelsUseCase {
	t.Helper()
	return appchannels.NewPushSalesOrderStatusToChannelsUseCase(newMockGormDB(t, n), salesOrderRepo, shipmentRepo, overrideRepo, connRepo, wooFactory, testEncryptionSecret)
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

func TestPushSalesOrderStatusToChannels_shippedWooOrder_pushesOnHoldAndAttachesShipmentNote(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	shipmentRepo := &stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{
		"so-1": {{ID: "ship-1", Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1", TrackingURL: "https://postnl.nl/tracktrace/?B=3SMYPA1"}},
	}}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 4, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, wooClient.updateOrderStatusCalls, 1)
	assert.Equal(t, "on-hold", wooClient.updateOrderStatusCalls[0].status, "Woo has no native shipped status -- CommerceOS's own record still says Shipped")
	require.Len(t, wooClient.addOrderNoteCalls, 1)
	assert.Equal(t, "501", wooClient.addOrderNoteCalls[0].externalOrderID)
	assert.Equal(t, "Shipped via PostNL, tracking 3SMYPA1 - https://postnl.nl/tracktrace/?B=3SMYPA1", wooClient.addOrderNoteCalls[0].note)
}

func TestPushSalesOrderStatusToChannels_shippedWooOrder_configuredTemplateWinsOverMyParcelLink(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
	}
	salesOrderRepo := &stubOrdersSalesOrderRepo{findByIDResult: order}
	shipmentRepo := &stubShipmentRepo{bySalesOrder: map[string][]shippingDomain.Shipment{
		"so-1": {{
			ID: "ship-1", ConnectionID: connID, Carrier: shippingDomain.CarrierPostNL, Barcode: "3SMYPA1",
			RecipientPostalCode: "1251 SB", TrackingURL: "https://myparcel-default/x",
		}},
	}}
	tmpl := "https://postnl.nl/tracktrace/?B={barcode}&P={postcode}&D=NL&T=C&L=EN"
	overrideRepo := &stubCarrierOverrideRepo{byConnectionAndCarrier: map[string]shippingDomain.ShippingCarrierOverride{
		connID + "|" + shippingDomain.CarrierPostNL.String(): {TrackingURLTemplate: &tmpl},
	}}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, ConsumerSecretEncrypted: encrypted}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := newPushSalesOrderStatusUseCaseWithOverrides(t, 4, salesOrderRepo, shipmentRepo, overrideRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, wooClient.addOrderNoteCalls, 1)
	assert.Equal(t, "Shipped via PostNL, tracking 3SMYPA1 - https://postnl.nl/tracktrace/?B=3SMYPA1&P=1251%20SB&D=NL&T=C&L=EN", wooClient.addOrderNoteCalls[0].note)
}

func TestPushSalesOrderStatusToChannels_shippedWooOrder_noTrackingURLYet_noteOmitsLink(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
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
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 4, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

	err = uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	require.Len(t, wooClient.addOrderNoteCalls, 1)
	assert.Equal(t, "Shipped via PostNL, tracking 3SMYPA1", wooClient.addOrderNoteCalls[0].note, "no tracking URL yet -- note falls back to barcode only")
}

func TestPushSalesOrderStatusToChannels_onHoldForUnrelatedReason_doesNotAttachNote(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusOnHold,
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
	assert.Empty(t, wooClient.addOrderNoteCalls, "on-hold set for a reason other than fulfillment (e.g. a manual/payment-review hold) must not get a shipment note")
}

func TestPushSalesOrderStatusToChannels_noteFailure_doesNotFailThePush(t *testing.T) {
	connID := "conn-1"
	order := &omsDomain.SalesOrder{
		ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, ExternalOrderID: "501",
		ConnectionID: &connID, Status: omsDomain.SalesOrderStatusShipped,
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
	uc := newPushSalesOrderStatusUseCaseWithShipments(t, 4, salesOrderRepo, shipmentRepo, connRepo, wooFactory)

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
