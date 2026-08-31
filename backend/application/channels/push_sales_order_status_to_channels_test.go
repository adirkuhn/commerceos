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
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func newPushSalesOrderStatusUseCase(t *testing.T, n int, salesOrderRepo *stubOrdersSalesOrderRepo, connRepo *stubConnectionRepo, wooFactory *stubWooClientFactory) *appchannels.PushSalesOrderStatusToChannelsUseCase {
	t.Helper()
	return appchannels.NewPushSalesOrderStatusToChannelsUseCase(newMockGormDB(t, n), salesOrderRepo, connRepo, wooFactory, testEncryptionSecret)
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
