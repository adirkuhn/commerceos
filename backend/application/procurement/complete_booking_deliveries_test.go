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

// A purchase order delivered in parts gets one booking per delivery
// (StartOrResumeBooking only resumes an unfinished one). Finishing a later
// booking must judge the PO by everything that arrived across all of its
// deliveries, not only by what this booking scanned.

type deliveryScenario struct {
	lineItems []procurementDomain.PurchaseOrderLineItem
	// earlier are the PO's already completed bookings, with their scans.
	earlier map[string][]procurementDomain.BookingScan
	// current is the scans of the booking being finished ("booking-now").
	current []procurementDomain.BookingScan
	// otherPOs are completed bookings of other purchase orders.
	otherPOs map[string][]procurementDomain.BookingScan
}

type deliveryRun struct {
	result   *appprocurement.CompleteBookingResult
	err      error
	poRepo   *stubPurchaseOrderRepo
	booking  *stubBookingRepo
	scans    *stubBookingScanRepo
	receive  *stubReceiveStockForBooking
	writeOff *stubMarkDamagedOrLostForBooking
	outbox   *eventrecordtest.Outbox
}

func productLineItem(id, skuID string, qty int64) procurementDomain.PurchaseOrderLineItem {
	return procurementDomain.PurchaseOrderLineItem{ID: id, PurchaseOrderID: "po-1", SKUID: strPtr(skuID), RawDescription: "Item " + id, Quantity: qty, LineType: procurementDomain.LineItemTypeProduct}
}

func received(id, skuID string, qty int64) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{ID: id, SKUID: skuID, ActionType: procurementDomain.BookingScanActionReceive, Quantity: qty}
}

func writtenOff(id, skuID string, action procurementDomain.BookingScanActionType, qty int64) procurementDomain.BookingScan {
	return procurementDomain.BookingScan{ID: id, SKUID: skuID, ActionType: action, WriteOffQuantity: qty}
}

func finishDelivery(t *testing.T, sc deliveryScenario, bookingErr error) deliveryRun {
	t.Helper()
	completedAt := time.Now().Add(-24 * time.Hour)
	run := deliveryRun{
		poRepo:   &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1"), Status: procurementDomain.PurchaseOrderStatusPartiallyReceived}},
		scans:    &stubBookingScanRepo{byBookingMap: map[string][]procurementDomain.BookingScan{"booking-now": sc.current}},
		receive:  &stubReceiveStockForBooking{},
		writeOff: &stubMarkDamagedOrLostForBooking{},
		outbox:   &eventrecordtest.Outbox{},
	}
	run.booking = &stubBookingRepo{
		findResult:   &procurementDomain.Booking{ID: "booking-now", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress},
		completedErr: bookingErr,
	}
	for id, scans := range sc.earlier {
		run.booking.completed = append(run.booking.completed, procurementDomain.Booking{ID: id, PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted, CompletedAt: &completedAt})
		run.scans.byBookingMap[id] = scans
	}
	for id, scans := range sc.otherPOs {
		run.booking.completed = append(run.booking.completed, procurementDomain.Booking{ID: id, PurchaseOrderID: "po-2", Status: procurementDomain.BookingStatusCompleted, CompletedAt: &completedAt})
		run.scans.byBookingMap[id] = scans
	}
	uc := appprocurement.NewCompleteBookingUseCase(newMockGormDBN(t, 2+len(sc.current)), run.booking, run.poRepo,
		&stubLineItemRepo{byPO: sc.lineItems}, run.scans, run.receive, &stubRecordAdjustmentForBooking{}, run.writeOff, run.outbox)
	run.result, run.err = uc.Execute(context.Background(), "booking-now", "user-1")
	return run
}

func (r deliveryRun) poStatus(t *testing.T) procurementDomain.PurchaseOrderStatus {
	t.Helper()
	require.Len(t, r.poRepo.updated, 1)
	return r.poRepo.updated[0].Status
}

func (r deliveryRun) completedEventMissingCount(t *testing.T) any {
	t.Helper()
	for _, e := range r.outbox.Events {
		if e.Subject == "procurement.booking.completed" {
			return e.Data()["missing_item_count"]
		}
	}
	t.Fatal("no procurement.booking.completed event")
	return nil
}

func TestCompleteBooking_secondDeliveryBringingTheRestMarksThePurchaseOrderReceived(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier:   map[string][]procurementDomain.BookingScan{"booking-1": {received("scan-1", "sku-1", 4)}},
		current:   []procurementDomain.BookingScan{received("scan-2", "sku-1", 6)},
	}, nil)

	require.NoError(t, run.err)
	assert.Empty(t, run.result.MissingItems)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, run.poStatus(t))
	assert.Equal(t, float64(0), run.completedEventMissingCount(t))
}

func TestCompleteBooking_secondDeliveryStillShortReportsWhatIsMissingOverall(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier:   map[string][]procurementDomain.BookingScan{"booking-1": {received("scan-1", "sku-1", 4)}},
		current:   []procurementDomain.BookingScan{received("scan-2", "sku-1", 3)},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.result.MissingItems, 1)
	assert.Equal(t, appprocurement.MissingBookingItem{LineItemID: "line-1", RawDescription: "Item line-1", ExpectedQty: 10, ScannedQty: 7}, run.result.MissingItems[0])
	assert.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, run.poStatus(t))
	assert.Equal(t, float64(1), run.completedEventMissingCount(t))
}

func TestCompleteBooking_threeDeliveriesAddUp(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier: map[string][]procurementDomain.BookingScan{
			"booking-1": {received("scan-1", "sku-1", 3)},
			"booking-2": {received("scan-2", "sku-1", 3)},
		},
		current: []procurementDomain.BookingScan{received("scan-3", "sku-1", 4)},
	}, nil)

	require.NoError(t, run.err)
	assert.Empty(t, run.result.MissingItems)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, run.poStatus(t))
}

func TestCompleteBooking_damagedAndLostInAnEarlierDeliveryCountAsArrived(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier: map[string][]procurementDomain.BookingScan{"booking-1": {
			received("scan-1", "sku-1", 5),
			writtenOff("scan-2", "sku-1", procurementDomain.BookingScanActionDamaged, 2),
			writtenOff("scan-3", "sku-1", procurementDomain.BookingScanActionLost, 1),
		}},
		current: []procurementDomain.BookingScan{received("scan-4", "sku-1", 2)},
	}, nil)

	require.NoError(t, run.err)
	assert.Empty(t, run.result.MissingItems, "5 received + 2 damaged + 1 lost + 2 now = 10")
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, run.poStatus(t))
}

func TestCompleteBooking_aCorrectionInAnEarlierDeliveryStillCounts(t *testing.T) {
	correction := procurementDomain.BookingScan{ID: "scan-2", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionCorrection, Quantity: -3}
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier:   map[string][]procurementDomain.BookingScan{"booking-1": {received("scan-1", "sku-1", 8), correction}},
		current:   []procurementDomain.BookingScan{received("scan-3", "sku-1", 4)},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.result.MissingItems, 1)
	assert.Equal(t, int64(9), run.result.MissingItems[0].ScannedQty, "8 - 3 earlier + 4 now")
	assert.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, run.poStatus(t))
}

func TestCompleteBooking_aLineFinishedEarlierStaysFinishedWhenThisDeliveryHasOnlyOtherItems(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-a", "sku-a", 5), productLineItem("line-b", "sku-b", 8)},
		earlier:   map[string][]procurementDomain.BookingScan{"booking-1": {received("scan-1", "sku-a", 5), received("scan-2", "sku-b", 2)}},
		current:   []procurementDomain.BookingScan{received("scan-3", "sku-b", 3)},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.result.MissingItems, 1)
	assert.Equal(t, "line-b", run.result.MissingItems[0].LineItemID)
	assert.Equal(t, int64(5), run.result.MissingItems[0].ScannedQty)
}

func TestCompleteBooking_anEmptyLaterBookingKeepsTheEarlierShortfall(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier:   map[string][]procurementDomain.BookingScan{"booking-1": {received("scan-1", "sku-1", 6)}},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.result.MissingItems, 1)
	assert.Equal(t, int64(6), run.result.MissingItems[0].ScannedQty)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, run.poStatus(t))
}

func TestCompleteBooking_otherPurchaseOrdersDeliveriesDoNotCount(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		otherPOs:  map[string][]procurementDomain.BookingScan{"booking-other": {received("scan-x", "sku-1", 10)}},
		current:   []procurementDomain.BookingScan{received("scan-1", "sku-1", 4)},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.result.MissingItems, 1)
	assert.Equal(t, int64(4), run.result.MissingItems[0].ScannedQty)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, run.poStatus(t))
}

func TestCompleteBooking_firstDeliveryAloneIsJudgedOnItsOwnScans(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		current:   []procurementDomain.BookingScan{received("scan-1", "sku-1", 10)},
	}, nil)

	require.NoError(t, run.err)
	assert.Empty(t, run.result.MissingItems)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusReceived, run.poStatus(t))
}

func TestCompleteBooking_onlyThisDeliveryIsTurnedIntoStock(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		earlier: map[string][]procurementDomain.BookingScan{"booking-1": {
			received("scan-1", "sku-1", 4), writtenOff("scan-2", "sku-1", procurementDomain.BookingScanActionDamaged, 1),
		}},
		current: []procurementDomain.BookingScan{received("scan-3", "sku-1", 5)},
	}, nil)

	require.NoError(t, run.err)
	require.Len(t, run.receive.calls, 1, "earlier deliveries are already stock; replaying them would double it")
	assert.Equal(t, int64(5), run.receive.calls[0].Quantity)
	assert.Empty(t, run.writeOff.calls)
	require.Len(t, run.scans.markReplayed, 1)
	assert.Equal(t, "scan-3", run.scans.markReplayed[0].scanID)
}

func TestCompleteBooking_failingToReadEarlierDeliveriesChangesNothing(t *testing.T) {
	run := finishDelivery(t, deliveryScenario{
		lineItems: []procurementDomain.PurchaseOrderLineItem{productLineItem("line-1", "sku-1", 10)},
		current:   []procurementDomain.BookingScan{received("scan-1", "sku-1", 10)},
	}, errors.New("db down"))

	require.Error(t, run.err)
	assert.Empty(t, run.receive.calls, "nothing becomes stock")
	assert.Empty(t, run.booking.updated, "booking stays open to retry")
	assert.Empty(t, run.poRepo.updated)
	assert.Empty(t, run.outbox.Events)
}
