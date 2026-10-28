package procurement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func openDiscrepancy(id string, typ procurementDomain.DiscrepancyType) *procurementDomain.ReceivingDiscrepancy {
	return &procurementDomain.ReceivingDiscrepancy{ID: id, ReceivingRecordID: "rec-1", Type: typ, Status: procurementDomain.DiscrepancyStatusOpen}
}

func TestAcknowledgeReceivingDiscrepancy_RecordsAcknowledged(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubReceivingDiscrepancyRepo{findResult: openDiscrepancy("d-1", procurementDomain.DiscrepancyTypeMissingItem)}
	uc := appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), "d-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.receiving_discrepancy.acknowledged", e.Subject)
	assert.Equal(t, "d-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"receiving_record_id": "rec-1", "type": "missing_item"}, e.Data())
}

func TestLinkReceivingDiscrepancies_RecordsBothSides(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubReceivingDiscrepancyRepo{byID: map[string]*procurementDomain.ReceivingDiscrepancy{
		"u-1": openDiscrepancy("u-1", procurementDomain.DiscrepancyTypeUnexpectedItem),
		"m-1": openDiscrepancy("m-1", procurementDomain.DiscrepancyTypeMissingItem),
	}}
	uc := appprocurement.NewLinkReceivingDiscrepanciesUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "u-1", "m-1"))

	require.Equal(t, []string{"procurement.receiving_discrepancy.linked", "procurement.receiving_discrepancy.linked"}, outbox.Subjects())
	assert.Equal(t, "u-1", outbox.Events[0].Payload["entity_id"])
	assert.Equal(t, map[string]any{"receiving_record_id": "rec-1", "type": "unexpected_item", "linked_discrepancy_id": "m-1"}, outbox.Events[0].Data())
	assert.Equal(t, "m-1", outbox.Events[1].Payload["entity_id"])
	assert.Equal(t, "u-1", outbox.Events[1].Data()["linked_discrepancy_id"])
}

func TestFlagDamagedLineItem_RecordsManualDiscrepancy(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{ID: "rec-1", PurchaseOrderID: "po-1"}}
	lines := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1"}}
	uc := appprocurement.NewFlagDamagedLineItemUseCase(newMockGormDB(t), recRepo, lines, &stubReceivingDiscrepancyRepo{}, outbox)

	_, err := uc.Execute(context.Background(), "rec-1", "line-1", 2, "crushed")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.receiving_discrepancy.detected", e.Subject)
	assert.Equal(t, map[string]any{
		"receiving_record_id": "rec-1", "type": "damaged_item", "line_item_id": "line-1",
		"expected_qty": float64(0), "received_qty": float64(2), "source": "manual",
	}, e.Data())
}

func TestCompleteReceiving_RecordsCompleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{ID: "rec-1", PurchaseOrderID: "po-1"}}
	uc := appprocurement.NewCompleteReceivingUseCase(newMockGormDB(t), recRepo, outbox)

	_, err := uc.Execute(context.Background(), "rec-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.receiving_record.completed", e.Subject)
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1"}, e.Data())
}

// The first booking of a PO picks its warehouse (a PO change) and starts.
func TestStartOrResumeBooking_RecordsWarehouseAndStarted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(newMockGormDB(t), poRepo, &stubBookingRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1", WarehouseID: strPtr("wh-1")})

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.purchase_order.updated", "procurement.booking.started"}, outbox.Subjects())
	assert.Equal(t, map[string][2]any{"warehouse_id": {nil, "wh-1"}}, outbox.Events[0].Changes())
	assert.Equal(t, map[string]any{"reason": "booking_started"}, outbox.Events[0].Data())
	assert.Equal(t, "booking-1", outbox.Events[1].Payload["entity_id"])
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1"}, outbox.Events[1].Data())
}

func TestStartOrResumeBooking_RecordsResumed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("wh-1")}}
	bookings := &stubBookingRepo{activeResult: &procurementDomain.Booking{ID: "booking-7", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusPaused}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(newMockGormDB(t), poRepo, bookings, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.booking.resumed", e.Subject)
	assert.Equal(t, "booking-7", e.Payload["entity_id"])
}

// Opening a booking that's already in progress changes nothing.
func TestStartOrResumeBooking_InProgressRecordsNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("wh-1")}}
	bookings := &stubBookingRepo{activeResult: &procurementDomain.Booking{ID: "booking-7", Status: procurementDomain.BookingStatusInProgress}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(newMockGormDB(t), poRepo, bookings, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	require.NoError(t, err)
	assert.Empty(t, outbox.Events)
}

func TestPauseBooking_RecordsPaused(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	bookings := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	uc := appprocurement.NewPauseBookingUseCase(newMockGormDB(t), bookings, outbox)

	_, err := uc.Execute(context.Background(), "booking-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.booking.paused", e.Subject)
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1"}, e.Data())
}

func TestScanBookingItem_RecordsScan(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	bookings := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("wh-1")}}
	uc := appprocurement.NewScanBookingItemUseCase(newMockGormDBN(t, 2), bookings, poRepo, &stubBookingScanRepo{}, &stubSKURepo{},
		&stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubLineItemRepo{}, outbox, &stubScopedStock{})

	_, err := uc.Execute(context.Background(), appprocurement.ScanBookingItemInput{BookingID: "booking-1", SKUID: "sku-1", Quantity: 4, LotNumber: "L1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.booking_scan.created", e.Subject)
	assert.Equal(t, "booking-scan-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"booking_id": "booking-1", "sku_id": "sku-1", "action_type": "receive", "quantity": float64(4),
		"write_off_quantity": float64(0), "lot_number": "L1", "expiry_date": nil,
	}, e.Data())
}

func TestDeleteBookingScan_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	bookings := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusInProgress}}
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive}}}
	uc := appprocurement.NewDeleteBookingScanUseCase(newMockGormDBN(t, 2), bookings, &stubPurchaseOrderRepo{}, scans, outbox, &stubScopedStock{})

	require.NoError(t, uc.Execute(context.Background(), "booking-1", "scan-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.booking_scan.deleted", e.Subject)
	assert.Equal(t, map[string]any{"booking_id": "booking-1", "sku_id": "sku-1", "action_type": "receive"}, e.Data())
}

func TestCompleteBooking_RecordsCompletedAndPurchaseOrderStatus(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	bookingRepo, poRepo := completableBookingFixtures()
	poRepo.findResult.Status = procurementDomain.PurchaseOrderStatusSent
	lines := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: strPtr("sku-1"), Quantity: 10, LineType: procurementDomain.LineItemTypeProduct}}}
	scans := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{{ID: "scan-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 6}}}
	uc := appprocurement.NewCompleteBookingUseCase(newMockGormDBN(t, 3), bookingRepo, poRepo, lines, scans,
		&stubReceiveStockForBooking{}, &stubRecordAdjustmentForBooking{}, &stubMarkDamagedOrLostForBooking{}, outbox)

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.booking.completed", "procurement.purchase_order.updated"}, outbox.Subjects())
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "completed_by": "user-1", "missing_item_count": float64(1)}, outbox.Events[0].Data())
	assert.Equal(t, map[string][2]any{"status": {"sent", "partially_received"}}, outbox.Events[1].Changes())
	assert.Equal(t, map[string]any{"reason": "booking_completed"}, outbox.Events[1].Data())
}

func TestReceivingAndBookingUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()

	_, err := appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(newMockGormDB(t),
		&stubReceivingDiscrepancyRepo{findResult: openDiscrepancy("d-1", procurementDomain.DiscrepancyTypeMissingItem)}, failing).Execute(ctx, "d-1")
	assert.ErrorIs(t, err, boom, "acknowledge")
	_, err = appprocurement.NewCompleteReceivingUseCase(newMockGormDB(t),
		&stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{ID: "rec-1"}}, failing).Execute(ctx, "rec-1")
	assert.ErrorIs(t, err, boom, "complete receiving")
	_, err = appprocurement.NewPauseBookingUseCase(newMockGormDB(t),
		&stubBookingRepo{findResult: &procurementDomain.Booking{ID: "b", Status: procurementDomain.BookingStatusInProgress}}, failing).Execute(ctx, "b")
	assert.ErrorIs(t, err, boom, "pause")
	_, err = appprocurement.NewStartOrResumeBookingUseCase(newMockGormDB(t),
		&stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("wh")}}, &stubBookingRepo{}, failing).
		Execute(ctx, appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})
	assert.ErrorIs(t, err, boom, "start")
}
