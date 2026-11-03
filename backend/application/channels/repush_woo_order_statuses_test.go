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
)

type fakeSalesOrderLister struct {
	orders    []omsDomain.SalesOrder
	err       error
	gotFilter omsDomain.ListSalesOrdersFilter
}

func (f *fakeSalesOrderLister) List(_ context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	f.gotFilter = filter
	return f.orders, int64(len(f.orders)), f.err
}

// wooStatuses answers ListOrderStatuses from a fixed map.
type wooStatuses struct {
	*stubWooClient
	byID   map[string]string
	err    error
	gotIDs []string
}

func (w *wooStatuses) ListOrderStatuses(_ context.Context, ids []string) (map[string]string, error) {
	w.gotIDs = ids
	return w.byID, w.err
}

type fakeStatusPusher struct {
	pushed []string
	err    error
}

func (f *fakeStatusPusher) Execute(_ context.Context, salesOrderID string) error {
	f.pushed = append(f.pushed, salesOrderID)
	return f.err
}

// repushOrigins records each order's Woo order id (conn-1) as tests build
// orders with wooOrder.
var repushOrigins = map[string]string{}

func wooOrder(id, external string, status omsDomain.SalesOrderStatus) omsDomain.SalesOrder {
	repushOrigins[id] = external
	return omsDomain.SalesOrder{ID: id, TenantID: "tenant-1", Status: status}
}

func TestRepushWooOrderStatuses_repushesWhereWooIsBehind(t *testing.T) {
	lister := &fakeSalesOrderLister{orders: []omsDomain.SalesOrder{
		wooOrder("so-stale", "101", omsDomain.SalesOrderStatusShipped),       // Woo missed the push (outage)
		wooOrder("so-synced", "102", omsDomain.SalesOrderStatusShipped),      // Woo has our on-hold
		wooOrder("so-regressed", "103", omsDomain.SalesOrderStatusCompleted), // Woo on-hold, we're completed
		wooOrder("so-cancelled", "104", omsDomain.SalesOrderStatusShipped),   // Woo cancelled: Woo is ahead, import's job
		wooOrder("so-gone", "105", omsDomain.SalesOrderStatusShipped),        // deleted in Woo
	}}
	client := &wooStatuses{stubWooClient: &stubWooClient{}, byID: map[string]string{
		"101": "processing", "102": "on-hold", "103": "on-hold", "104": "cancelled",
	}}
	pusher := &fakeStatusPusher{}
	uc := appchannels.NewRepushWooOrderStatusesUseCase(newMockGormDB(t, 1), lister, wooOrigins(repushOrigins), &fakeWooClients{client: client}, pusher)

	require.NoError(t, uc.Run(context.Background()))

	assert.Equal(t, []string{"so-stale", "so-regressed"}, pusher.pushed)
	assert.ElementsMatch(t, []string{"101", "102", "103", "104", "105"}, client.gotIDs)
	assert.Equal(t, []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourceWooCommerce}, lister.gotFilter.Source)
	assert.ElementsMatch(t, []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted}, lister.gotFilter.Status)
	assert.Positive(t, lister.gotFilter.Limit)
}

// COM-15: orders are grouped by the connection their origin came from; an
// order that resolves nowhere is left out, not grouped under nil.
func TestRepushWooOrderStatuses_groupsOrdersByTheConnectionTheyCameFrom(t *testing.T) {
	lister := &fakeSalesOrderLister{orders: []omsDomain.SalesOrder{
		{ID: "so-1", TenantID: "tenant-1", Status: omsDomain.SalesOrderStatusShipped},
		{ID: "so-2", TenantID: "tenant-1", Status: omsDomain.SalesOrderStatusShipped},
	}}
	origins := shopOrigins(integrationsDomain.OriginWooCommerce, "conn-2", map[string]string{"so-1": "101"})
	origins.byOrder["so-2"] = integrationsDomain.SalesOrderOrigin{SalesOrderID: "so-2", Origin: integrationsDomain.OriginWooCommerce, ExternalOrderID: "102"}
	client := &wooStatuses{stubWooClient: &stubWooClient{}, byID: map[string]string{"101": "processing"}}
	clients := &fakeWooClients{client: client}
	pusher := &fakeStatusPusher{}
	uc := appchannels.NewRepushWooOrderStatusesUseCase(newMockGormDB(t, 1), lister, origins, clients, pusher)

	require.NoError(t, uc.Run(context.Background()))

	assert.Equal(t, []string{"conn-2"}, clients.gotConnections)
	assert.Equal(t, []string{"101"}, client.gotIDs)
	assert.Equal(t, []string{"so-1"}, pusher.pushed)
}

func TestRepushWooOrderStatuses_oneConnectionFailingDoesNotStopTheRun(t *testing.T) {
	lister := &fakeSalesOrderLister{orders: []omsDomain.SalesOrder{wooOrder("so-1", "101", omsDomain.SalesOrderStatusShipped)}}
	client := &wooStatuses{stubWooClient: &stubWooClient{}, err: errors.New("403")}
	pusher := &fakeStatusPusher{}
	uc := appchannels.NewRepushWooOrderStatusesUseCase(newMockGormDB(t, 1), lister, wooOrigins(repushOrigins), &fakeWooClients{client: client}, pusher)

	assert.NoError(t, uc.Run(context.Background()), "the store is still down -- the next run retries")
	assert.Empty(t, pusher.pushed)
}

func TestRepushWooOrderStatuses_aFailedPushIsLoggedNotFatal(t *testing.T) {
	lister := &fakeSalesOrderLister{orders: []omsDomain.SalesOrder{
		wooOrder("so-1", "101", omsDomain.SalesOrderStatusShipped),
		wooOrder("so-2", "102", omsDomain.SalesOrderStatusShipped),
	}}
	client := &wooStatuses{stubWooClient: &stubWooClient{}, byID: map[string]string{"101": "processing", "102": "processing"}}
	pusher := &fakeStatusPusher{err: errors.New("woo down")}
	uc := appchannels.NewRepushWooOrderStatusesUseCase(newMockGormDB(t, 1), lister, wooOrigins(repushOrigins), &fakeWooClients{client: client}, pusher)

	assert.NoError(t, uc.Run(context.Background()))
	assert.Equal(t, []string{"so-1", "so-2"}, pusher.pushed, "keeps going after one failure")
}

func TestRepushWooOrderStatuses_listingFailureFailsTheRun(t *testing.T) {
	lister := &fakeSalesOrderLister{err: errors.New("db down")}
	uc := appchannels.NewRepushWooOrderStatusesUseCase(newMockGormDB(t, 1), lister, wooOrigins(repushOrigins), &fakeWooClients{}, &fakeStatusPusher{})

	assert.Error(t, uc.Run(context.Background()))
}

var _ integrationsDomain.WooClient = (*wooStatuses)(nil)
