package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type fakeComparableOrders struct {
	orders   []omsDomain.SalesOrder
	err      error
	gotConn  string
	gotAfter *omsDomain.OrderPageCursor
	gotLimit int
}

func (f *fakeComparableOrders) ListChannelComparableOrders(_ context.Context, connectionID string, after *omsDomain.OrderPageCursor, limit int) ([]omsDomain.SalesOrder, error) {
	f.gotConn, f.gotAfter, f.gotLimit = connectionID, after, limit
	return f.orders, f.err
}

func newAudit(t *testing.T, orders *fakeComparableOrders, woo map[string]integrationsDomain.WooOrder, recorder *fakeDifferenceRecorder) *appchannels.AuditWooOrderDifferencesUseCase {
	t.Helper()
	clients := &fakeWooClients{client: wooOrdersByID{stubWooClient: &stubWooClient{}, byID: woo}}
	return appchannels.NewAuditWooOrderDifferencesUseCase(newMockGormDB(t, 1), orders, clients, recorder)
}

func TestAuditWooOrderDifferences_comparesEachOrderWithWoo(t *testing.T) {
	orders := &fakeComparableOrders{orders: []omsDomain.SalesOrder{
		{ID: "so-1", ExternalOrderID: "25922"}, {ID: "so-2", ExternalOrderID: "25921"},
	}}
	woo := map[string]integrationsDomain.WooOrder{"25922": pinducaWooOrder(), "25921": simpleWooOrder()}
	recorder := &fakeDifferenceRecorder{}

	result, err := newAudit(t, orders, woo, recorder).Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.NextCursor, "a short batch is the last one")
	assert.Equal(t, "conn-1", orders.gotConn)
	assert.Nil(t, orders.gotAfter)
	require.Len(t, recorder.calls, 2)
	assert.Equal(t, "so-1", recorder.calls[0].SalesOrderID)
	assert.Len(t, recorder.calls[0].ChannelLines, 2)
	assert.Equal(t, int64(1600), recorder.calls[0].ChannelTotalMinorUnits)
}

func TestAuditWooOrderDifferences_fullBatch_returnsCursorToContinue(t *testing.T) {
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	var batch []omsDomain.SalesOrder
	woo := map[string]integrationsDomain.WooOrder{}
	for i := 0; i < appchannels.AuditWooOrderDifferencesBatchSize; i++ {
		id := string(rune('a' + i))
		batch = append(batch, omsDomain.SalesOrder{ID: "so-" + id, ExternalOrderID: id, OccurredAt: at})
		woo[id] = simpleWooOrder()
	}
	orders := &fakeComparableOrders{orders: batch}
	uc := newAudit(t, orders, woo, &fakeDifferenceRecorder{})

	result, err := uc.Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1"})
	require.NoError(t, err)
	require.NotEmpty(t, result.NextCursor)

	_, err = newAudit(t, orders, woo, &fakeDifferenceRecorder{}).Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1", Cursor: result.NextCursor})
	require.NoError(t, err)
	require.NotNil(t, orders.gotAfter)
	assert.Equal(t, batch[len(batch)-1].ID, orders.gotAfter.ID)
	assert.True(t, at.Equal(orders.gotAfter.OccurredAt))
}

func TestAuditWooOrderDifferences_oneOrderFails_restAreStillChecked(t *testing.T) {
	orders := &fakeComparableOrders{orders: []omsDomain.SalesOrder{
		{ID: "so-gone", ExternalOrderID: "999"}, {ID: "so-1", ExternalOrderID: "25921"},
	}}
	recorder := &fakeDifferenceRecorder{}

	result, err := newAudit(t, orders, map[string]integrationsDomain.WooOrder{"25921": simpleWooOrder()}, recorder).
		Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "999")
	assert.Len(t, recorder.calls, 1)
}

func TestAuditWooOrderDifferences_recordFails_isReported(t *testing.T) {
	orders := &fakeComparableOrders{orders: []omsDomain.SalesOrder{{ID: "so-1", ExternalOrderID: "25921"}}}

	result, err := newAudit(t, orders, map[string]integrationsDomain.WooOrder{"25921": simpleWooOrder()}, &fakeDifferenceRecorder{err: errors.New("db down")}).
		Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Len(t, result.Errors, 1)
}

func TestAuditWooOrderDifferences_badCursor_isRejected(t *testing.T) {
	_, err := newAudit(t, &fakeComparableOrders{}, nil, &fakeDifferenceRecorder{}).
		Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1", Cursor: "garbage"})

	assert.ErrorIs(t, err, appchannels.ErrInvalidBackfillCursor)
}

func TestAuditWooOrderDifferences_listingFails_returnsError(t *testing.T) {
	_, err := newAudit(t, &fakeComparableOrders{err: errors.New("db down")}, nil, &fakeDifferenceRecorder{}).
		Execute(context.Background(), appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1"})

	assert.Error(t, err)
}
