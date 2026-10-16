package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// POs left partially_received by the old per-booking check, though their
// deliveries together brought everything.

type reconcileFixture struct {
	poRepo   *stubPurchaseOrderRepo
	bookings *stubBookingRepo
	lines    *stubLineItemRepo
	scans    *stubBookingScanRepo
	outbox   *eventrecordtest.Outbox
}

func newReconcileFixture() *reconcileFixture {
	completedAt := time.Now().Add(-time.Hour)
	po := func(id string, status procurementDomain.PurchaseOrderStatus) procurementDomain.PurchaseOrder {
		return procurementDomain.PurchaseOrder{ID: id, SupplierID: strPtr("sup-1"), Status: status}
	}
	completed := func(id, poID string) procurementDomain.Booking {
		return procurementDomain.Booking{ID: id, PurchaseOrderID: poID, Status: procurementDomain.BookingStatusCompleted, CompletedAt: &completedAt}
	}
	f := &reconcileFixture{
		// po-stuck: 4 + 6 of 10 over two deliveries -> really received.
		// po-short: 4 + 3 of 10 -> really still short.
		// po-busy: complete over its deliveries, but a third is being booked.
		// po-done / po-draft: other statuses, never touched.
		poRepo: &stubPurchaseOrderRepo{purchaseOrders: []procurementDomain.PurchaseOrder{
			po("po-stuck", procurementDomain.PurchaseOrderStatusPartiallyReceived),
			po("po-short", procurementDomain.PurchaseOrderStatusPartiallyReceived),
			po("po-busy", procurementDomain.PurchaseOrderStatusPartiallyReceived),
			po("po-done", procurementDomain.PurchaseOrderStatusReceived),
			po("po-draft", procurementDomain.PurchaseOrderStatusDraft),
		}},
		bookings: &stubBookingRepo{
			completed: []procurementDomain.Booking{
				completed("b-stuck-1", "po-stuck"), completed("b-stuck-2", "po-stuck"),
				completed("b-short-1", "po-short"), completed("b-short-2", "po-short"),
				completed("b-busy-1", "po-busy"),
			},
			activeByPO: map[string]*procurementDomain.Booking{"po-busy": {ID: "b-busy-2", PurchaseOrderID: "po-busy", Status: procurementDomain.BookingStatusPaused}},
		},
		lines: &stubLineItemRepo{byPOMap: map[string][]procurementDomain.PurchaseOrderLineItem{
			"po-stuck": {productLineItem("line-1", "sku-1", 10)},
			"po-short": {productLineItem("line-2", "sku-1", 10)},
			"po-busy":  {productLineItem("line-3", "sku-1", 5)},
		}},
		scans: &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{
			"b-stuck-1": {received("s1", "sku-1", 4)}, "b-stuck-2": {received("s2", "sku-1", 6)},
			"b-short-1": {received("s3", "sku-1", 4)}, "b-short-2": {received("s4", "sku-1", 3)},
			"b-busy-1": {received("s5", "sku-1", 5)},
		}},
		outbox: &eventrecordtest.Outbox{},
	}
	f.poRepo.byID = map[string]*procurementDomain.PurchaseOrder{}
	for i := range f.poRepo.purchaseOrders {
		f.poRepo.byID[f.poRepo.purchaseOrders[i].ID] = &f.poRepo.purchaseOrders[i]
	}
	return f
}

func (f *reconcileFixture) useCase(t *testing.T) *appprocurement.ReconcilePartiallyReceivedPurchaseOrdersUseCase {
	return appprocurement.NewReconcilePartiallyReceivedPurchaseOrdersUseCase(newMockGormDBN(t, 4), f.poRepo, f.bookings, f.lines, f.scans, f.outbox)
}

func TestReconcilePartiallyReceived_marksOnlyCompletePurchaseOrdersReceived(t *testing.T) {
	f := newReconcileFixture()

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.ReconcilePartiallyReceivedResult{
		Checked: 3, Fixed: []string{"po-stuck"}, StillShort: []string{"po-short"}, BookingInProgress: []string{"po-busy"},
	}, *got)
	require.Len(t, f.poRepo.updated, 1)
	assert.Equal(t, "po-stuck", f.poRepo.updated[0].ID)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, f.poRepo.updated[0].Status)
	e := onlyEvent(t, f.outbox)
	assert.Equal(t, "procurement.purchase_order.updated", e.Subject)
	assert.Equal(t, "po-stuck", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"status": {"partially_received", "received"}}, e.Changes())
	assert.Equal(t, map[string]any{"reason": "deliveries_reconciled"}, e.Data())
}

func TestReconcilePartiallyReceived_dryRunReportsButWritesNothing(t *testing.T) {
	f := newReconcileFixture()

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{DryRun: true})

	require.NoError(t, err)
	assert.Equal(t, []string{"po-stuck"}, got.Fixed)
	assert.Empty(t, f.poRepo.updated)
	assert.Empty(t, f.outbox.Events)
}

func TestReconcilePartiallyReceived_secondRunChangesNothing(t *testing.T) {
	f := newReconcileFixture()
	_, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})
	require.NoError(t, err)
	f.poRepo.byID["po-stuck"].Status = procurementDomain.PurchaseOrderStatusReceived
	f.poRepo.purchaseOrders[0].Status = procurementDomain.PurchaseOrderStatusReceived
	f.poRepo.updated, f.outbox.Events = nil, nil

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.NoError(t, err)
	assert.Empty(t, got.Fixed)
	assert.Equal(t, 2, got.Checked)
	assert.Empty(t, f.poRepo.updated)
}

func TestReconcilePartiallyReceived_damagedAndLostCountAsArrived(t *testing.T) {
	f := newReconcileFixture()
	f.scans.byBookingMap["b-short-2"] = []procurementDomain.BookingScan{
		received("s4", "sku-1", 3), writtenOff("s6", "sku-1", procurementDomain.BookingScanActionDamaged, 2), writtenOff("s7", "sku-1", procurementDomain.BookingScanActionLost, 1),
	}

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.NoError(t, err)
	assert.Equal(t, []string{"po-stuck", "po-short"}, got.Fixed)
	assert.Empty(t, got.StillShort)
}

func TestReconcilePartiallyReceived_noCompletedBookingIsLeftAlone(t *testing.T) {
	f := newReconcileFixture()
	f.bookings.completed = nil

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.NoError(t, err)
	assert.Empty(t, got.Fixed)
	assert.Equal(t, []string{"po-stuck", "po-short"}, got.StillShort)
}

func TestReconcilePartiallyReceived_aStatusChangedSinceListingIsSkipped(t *testing.T) {
	f := newReconcileFixture()
	// Someone cancelled it between the list and its own transaction.
	f.poRepo.byID["po-stuck"] = &procurementDomain.PurchaseOrder{ID: "po-stuck", Status: procurementDomain.PurchaseOrderStatusCancelled}

	got, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.NoError(t, err)
	assert.Empty(t, got.Fixed)
	assert.Empty(t, f.poRepo.updated)
}

func TestReconcilePartiallyReceived_anUpdateFailureStopsWithThePurchaseOrderNamed(t *testing.T) {
	f := newReconcileFixture()
	f.poRepo.updateErr = errors.New("db down")

	_, err := f.useCase(t).Execute(context.Background(), appprocurement.ReconcilePartiallyReceivedInput{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "po-stuck")
	assert.Empty(t, f.outbox.Events)
}
