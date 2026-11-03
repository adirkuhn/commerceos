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

type fakeMissingPaymentLister struct {
	orders   []omsDomain.SalesOrder
	gotConn  string
	gotAfter *omsDomain.MissingPaymentInfoCursor
	gotLimit int
}

func (f *fakeMissingPaymentLister) ListMissingPaymentInfo(_ context.Context, connectionID string, after *omsDomain.MissingPaymentInfoCursor, limit int) ([]omsDomain.SalesOrder, error) {
	f.gotConn, f.gotAfter, f.gotLimit = connectionID, after, limit
	return f.orders, nil
}

// wooOrdersByID answers GetOrder per external order id.
type wooOrdersByID struct {
	*stubWooClient
	byID map[string]integrationsDomain.WooOrder
}

func (w wooOrdersByID) GetOrder(_ context.Context, id string) (*integrationsDomain.WooOrder, error) {
	o, ok := w.byID[id]
	if !ok {
		return nil, errors.New("woo: order not found")
	}
	return &o, nil
}

type fakeWooClients struct {
	client integrationsDomain.WooClient
	err    error
	gotID  string
	// gotConnections: every connection asked for, in order
	gotConnections []string
}

func (f *fakeWooClients) ClientFor(_ context.Context, connectionID string) (integrationsDomain.WooClient, error) {
	f.gotID = connectionID
	f.gotConnections = append(f.gotConnections, connectionID)
	return f.client, f.err
}

func paidWooOrder(id int, txn string) integrationsDomain.WooOrder {
	return integrationsDomain.WooOrder{ID: id, Status: "completed", PaymentMethod: "stripe", PaymentMethodTitle: "Credit card", TransactionID: txn, DatePaid: "2026-09-20T10:00:00"}
}

func TestBackfillWooOrderPayments_fillsInWhatWooHasForEachOrder(t *testing.T) {
	at := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	lister := &fakeMissingPaymentLister{orders: []omsDomain.SalesOrder{
		{ID: "so-1", ExternalOrderID: "101", Status: omsDomain.SalesOrderStatusCompleted, PaymentMethod: "stripe", OccurredAt: at},
		{ID: "so-2", ExternalOrderID: "102", Status: omsDomain.SalesOrderStatusOnHold, OccurredAt: at.Add(-time.Hour)},
	}}
	bacs := integrationsDomain.WooOrder{ID: 102, Status: "on-hold", PaymentMethod: "bacs", PaymentMethodTitle: "Bank transfer"}
	clients := &fakeWooClients{client: wooOrdersByID{&stubWooClient{}, map[string]integrationsDomain.WooOrder{"101": paidWooOrder(101, "ch_101"), "102": bacs}}}
	update := &stubUpdateSalesOrderUC{}
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 1), lister, clients, update)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, "conn-1", clients.gotID)
	assert.Equal(t, "conn-1", lister.gotConn)
	assert.Nil(t, lister.gotAfter)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 2, result.Updated)
	assert.Empty(t, result.Errors)
	require.Len(t, update.calls, 2)
	assert.Equal(t, "so-1", update.ids[0])
	require.NotNil(t, update.calls[0].TransactionID)
	assert.Equal(t, "ch_101", *update.calls[0].TransactionID)
	require.NotNil(t, update.calls[0].DatePaid)
	assert.Nil(t, update.calls[0].Status, "payment fields only -- never status")
	require.NotNil(t, update.calls[1].PaymentMethodTitle)
	assert.Equal(t, "Bank transfer", *update.calls[1].PaymentMethodTitle)
	assert.Empty(t, result.NextCursor, "fewer than a full batch -- done")
}

func TestBackfillWooOrderPayments_countsAnOrderWooHasNothingNewForAsChecked(t *testing.T) {
	lister := &fakeMissingPaymentLister{orders: []omsDomain.SalesOrder{
		{ID: "so-1", ExternalOrderID: "101", Status: omsDomain.SalesOrderStatusCompleted, PaymentMethod: "cod", PaymentMethodTitle: "Cash on delivery"},
	}}
	cod := integrationsDomain.WooOrder{ID: 101, Status: "completed", PaymentMethod: "cod", PaymentMethodTitle: "Cash on delivery"}
	clients := &fakeWooClients{client: wooOrdersByID{&stubWooClient{}, map[string]integrationsDomain.WooOrder{"101": cod}}}
	update := &stubUpdateSalesOrderUC{}
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 1), lister, clients, update)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Checked)
	assert.Equal(t, 0, result.Updated)
	assert.Empty(t, update.calls)
}

func TestBackfillWooOrderPayments_oneFailingOrderDoesNotStopTheBatch(t *testing.T) {
	lister := &fakeMissingPaymentLister{orders: []omsDomain.SalesOrder{
		{ID: "so-1", ExternalOrderID: "999"},
		{ID: "so-2", ExternalOrderID: "102", Status: omsDomain.SalesOrderStatusCompleted},
	}}
	clients := &fakeWooClients{client: wooOrdersByID{&stubWooClient{}, map[string]integrationsDomain.WooOrder{"102": paidWooOrder(102, "ch_102")}}}
	update := &stubUpdateSalesOrderUC{}
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 1), lister, clients, update)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "999")
}

func TestBackfillWooOrderPayments_aFullBatchReturnsACursorThatResumesAfterIt(t *testing.T) {
	at := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	orders := make([]omsDomain.SalesOrder, appchannels.BackfillWooOrderPaymentsBatchSize)
	byID := map[string]integrationsDomain.WooOrder{}
	for i := range orders {
		ext := string(rune('a'+i%26)) + time.Duration(i).String()
		orders[i] = omsDomain.SalesOrder{ID: "so-" + ext, ExternalOrderID: ext, OccurredAt: at.Add(-time.Duration(i) * time.Minute)}
		byID[ext] = integrationsDomain.WooOrder{}
	}
	lister := &fakeMissingPaymentLister{orders: orders}
	clients := &fakeWooClients{client: wooOrdersByID{&stubWooClient{}, byID}}
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 2), lister, clients, &stubUpdateSalesOrderUC{})

	first, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1"})
	require.NoError(t, err)
	require.NotEmpty(t, first.NextCursor)
	assert.Equal(t, appchannels.BackfillWooOrderPaymentsBatchSize, lister.gotLimit)

	lister.orders = nil
	second, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1", Cursor: first.NextCursor})
	require.NoError(t, err)
	last := orders[len(orders)-1]
	require.NotNil(t, lister.gotAfter)
	assert.True(t, last.OccurredAt.Equal(lister.gotAfter.OccurredAt))
	assert.Equal(t, last.ID, lister.gotAfter.ID)
	assert.Empty(t, second.NextCursor)
}

func TestBackfillWooOrderPayments_rejectsAMalformedCursor(t *testing.T) {
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 0), &fakeMissingPaymentLister{}, &fakeWooClients{client: &stubWooClient{}}, &stubUpdateSalesOrderUC{})

	_, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1", Cursor: "not-a-cursor"})

	assert.ErrorIs(t, err, appchannels.ErrInvalidBackfillCursor)
}

func TestBackfillWooOrderPayments_propagatesAConnectionThatIsNotWooCommerce(t *testing.T) {
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(newMockGormDB(t, 0), &fakeMissingPaymentLister{},
		&fakeWooClients{err: integrationsDomain.ErrProviderNotSupported}, &stubUpdateSalesOrderUC{})

	_, err := uc.Execute(context.Background(), appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-bol"})

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}
