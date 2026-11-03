package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// The event is part of the write: for every oms use case that records one,
// an outbox failure fails the use case (its transaction rolls back) instead
// of leaving a change nobody hears about. Complements
// TestOMSUseCases_FailWhenTheEventCantBeRecorded.

var errOMSOutboxDown = errors.New("outbox down")

func failingOutbox() *eventrecordtest.Outbox { return &eventrecordtest.Outbox{Err: errOMSOutboxDown} }

func failingOutboxRepo() *stubOutboxRepo { return &stubOutboxRepo{err: errOMSOutboxDown} }

// subjectFailingOutbox fails only the enqueue of one subject, so a test can
// prove a specific event in a multi-event flow is part of the write.
type subjectFailingOutbox struct {
	stubOutboxRepo
	subject string
}

func (o *subjectFailingOutbox) Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error {
	if subject == o.subject {
		return errOMSOutboxDown
	}
	return o.stubOutboxRepo.Enqueue(ctx, subject, eventID, payload, occurredAt)
}

func inProgressPacking() *stubPackingSessionRepo {
	return &stubPackingSessionRepo{findByIDResult: &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress}}
}

func inProgressPicking() *stubPickingSessionRepo {
	return &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
}

func TestSalesOrderUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	ctx := context.Background()

	t.Run("create sales order", func(t *testing.T) {
		uc := appoms.NewCreateSalesOrderUseCase(newMockGormDBN(t, 3), &stubSalesOrderRepo{}, &stubOrigins{}, &stubSalesOrderLineItemRepo{},
			&stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.CreateSalesOrderInput{
			WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(), Status: omsDomain.SalesOrderStatusPending,
			LineItems: []appoms.CreateSalesOrderLineItemInput{{Name: "Widget", Quantity: 1, PriceMinorUnits: 500, TotalMinorUnits: 500}},
		})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("receive POS sales order", func(t *testing.T) {
		uc := appoms.NewReceivePOSSalesOrderUseCase(newMockGormDBN(t, 3), &stubSalesOrderRepo{}, &stubOrigins{}, &stubSalesOrderLineItemRepo{},
			&stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.ReceivePOSSalesOrderInput{
			ExternalOrderID: "pos-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(), TotalMinorUnits: 500,
			LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{Name: "Widget", Quantity: 1, PriceMinorUnits: 500, TotalMinorUnits: 500}},
		})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("update sales order", func(t *testing.T) {
		lines := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{SKUID: strPtr("sku-1"), Quantity: 3}}}
		uc := appoms.NewUpdateSalesOrderUseCase(newMockGormDBN(t, 4), &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}, lines,
			&stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &stubReverseMovement{}, failingOutboxRepo())
		status := omsDomain.SalesOrderStatusProcessing
		_, err := uc.Execute(ctx, "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("delete sales order", func(t *testing.T) {
		uc := appoms.NewDeleteSalesOrderUseCase(newMockGormDBN(t, 3), &stubSalesOrderRepo{findByIDResult: existingOrder(nil)},
			&stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubReverseMovement{}, failingOutbox())
		assert.ErrorIs(t, uc.Execute(ctx, "sales-order-1"), errOMSOutboxDown)
	})

	t.Run("add line item", func(t *testing.T) {
		skus := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", SKUCode: "SKU-1"}}}
		uc := appoms.NewAddSalesOrderLineItemUseCase(newMockGormDB(t), &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}},
			&stubSalesOrderLineItemRepo{}, skus, &stubLinePricer{includesTax: true}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Name: "Freebie", Quantity: 1, PriceMinorUnits: 500})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("link line item to SKU", func(t *testing.T) {
		lines := &stubSalesOrderLineItemRepo{
			findByIDResult:   &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "sales-order-1"},
			listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "sales-order-1", SKUID: strPtr("sku-1")}},
		}
		uc := appoms.NewLinkLineItemSKUUseCase(newMockGormDB(t), &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) { o.NeedsSKUReview = true })},
			lines, &stubLinkSKURepo{result: &pmsDomain.SKU{ID: "sku-1", SKUCode: "WID-A"}}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.LinkLineItemSKUInput{SalesOrderID: "sales-order-1", LineItemID: "li-1", SKUID: "sku-1"})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("mark order picked", func(t *testing.T) {
		f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
		f.outbox.Err = errOMSOutboxDown
		assert.ErrorIs(t, f.run(t, omsDomain.LockActor{UserID: "user-1"}), errOMSOutboxDown)
	})
}

func TestStockReviewUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	ctx := context.Background()
	pendingOrder := func() *stubSalesOrderRepo {
		return &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
			o.Status = omsDomain.SalesOrderStatusProcessing
			o.NeedsStockReview = true
		})}
	}
	pendingLine := func() *stubSalesOrderLineItemRepo {
		return &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
			{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2, StockStatus: omsDomain.StockStatusPending},
		}}
	}

	t.Run("force accept stock", func(t *testing.T) {
		uc := appoms.NewForceAcceptStockUseCase(newMockGormDBN(t, 5), pendingOrder(), pendingLine(), &stubSalesOrderLineItemStockMovementRepo{},
			&stubSKURepo{}, &stubRecordMovement{}, failingOutbox())
		_, err := uc.Execute(ctx, "sales-order-1")
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("retry stock decrement", func(t *testing.T) {
		uc := appoms.NewRetryStockDecrementUseCase(newMockGormDBN(t, 5), pendingOrder(), pendingLine(), &stubSalesOrderLineItemStockMovementRepo{},
			&stubSKURepo{}, &stubRecordMovement{}, failingOutbox())
		_, err := uc.Execute(ctx, "sales-order-1")
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("repair stock review flags", func(t *testing.T) {
		orders := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
			{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: true},
		}}
		lines := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
			{ID: "li-1", SalesOrderID: "so-1", SKUID: strPtr("sku-1"), StockStatus: omsDomain.StockStatusDecremented},
		}}
		uc := appoms.NewRepairStockReviewFlagsUseCase(newMockGormDB(t), orders, lines, &stubSalesOrderLineItemStockMovementRepo{existsResult: false}, failingOutbox())
		_, err := uc.Execute(ctx)
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})
}

func TestPickingAndPackingUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	ctx := context.Background()
	actor := omsDomain.LockActor{UserID: "user-1"}

	t.Run("resume picking", func(t *testing.T) {
		sessions := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusPaused}}
		uc := appoms.NewStartOrResumePickingUseCase(newMockGormDB(t), &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}},
			sessions, &stubPackingSessionRepo{}, &stubPackScanRepo{}, failingOutbox())
		_, err := uc.Execute(ctx, "order-1", actor)
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("start packing", func(t *testing.T) {
		orders := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}}
		uc := appoms.NewStartOrResumePackingUseCase(newMockGormDB(t), orders, &stubPackingSessionRepo{}, &stubPickingSessionRepo{}, &stubPickScanRepo{}, failingOutbox())
		_, err := uc.Execute(ctx, "order-1", actor)
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("start packing over a stale picking session (takeover)", func(t *testing.T) {
		picker, stale := "picker", time.Now().Add(-omsDomain.SessionLockTTL-time.Minute)
		picking := &stubPickingSessionRepo{activeResult: &omsDomain.PickingSession{
			ID: "picking-1", Status: omsDomain.PickingSessionStatusPaused, LockedBy: &picker, LockedAt: &stale,
		}}
		scans := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ActionType: omsDomain.PickScanActionPick}}}
		uc := appoms.NewStartOrResumePackingUseCase(newMockGormDB(t), &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}},
			&stubPackingSessionRepo{}, picking, scans, failingOutbox())
		_, err := uc.Execute(ctx, "order-1", actor)
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("reset packing", func(t *testing.T) {
		uc := appoms.NewResetPackingSessionUseCase(newMockGormDB(t), inProgressPacking(), &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}}}, failingOutbox())
		_, err := uc.Execute(ctx, "session-1")
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("pick scan", func(t *testing.T) {
		lines := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1")}}}
		uc := appoms.NewScanPickItemUseCase(newMockGormDB(t), inProgressPicking(), &stubPickScanRepo{}, lines, &stubSalesOrderRepo{}, &stubSKURepo{},
			&stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.ScanPickItemInput{SessionID: "session-1", SKUID: "sku-1", Quantity: 1, ScannedBy: "user-1"})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("delete pick scan", func(t *testing.T) {
		uc := appoms.NewDeletePickScanUseCase(newMockGormDB(t), inProgressPicking(), &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-1"}}}, failingOutbox())
		assert.ErrorIs(t, uc.Execute(ctx, "session-1", "scan-1"), errOMSOutboxDown)
	})

	t.Run("pack scan", func(t *testing.T) {
		skuID := "sku-1"
		lines := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: &skuID, Quantity: 3}}}
		skus := &stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BARCODE-1": {ID: "sku-1"}}}
		uc := appoms.NewScanPackItemUseCase(newMockGormDB(t), inProgressPacking(), &stubPackScanRepo{}, lines, skus,
			&stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.ScanPackItemInput{SessionID: "session-1", Code: "BARCODE-1", Quantity: 1, ScannedBy: "user-1"})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("delete pack scan", func(t *testing.T) {
		uc := appoms.NewDeletePackScanUseCase(newMockGormDB(t), inProgressPacking(), &stubPackScanRepo{listBySession: []omsDomain.PackScan{{ID: "scan-1"}}}, failingOutbox())
		assert.ErrorIs(t, uc.Execute(ctx, "session-1", "scan-1"), errOMSOutboxDown)
	})

	t.Run("stage a substitute", func(t *testing.T) {
		lines := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", Quantity: 5}}
		uc := appoms.NewSubstituteLineItemUseCase(newMockGormDB(t), inProgressPacking(), &stubPackScanRepo{}, lines, failingOutbox())
		_, err := uc.Execute(ctx, appoms.SubstituteLineItemInput{
			SessionID: "session-1", LineItemID: "li-1", ScannedBy: "user-1", Reason: "out of stock",
			Lines: []appoms.SubstituteLineInput{{SKUID: "sku-sub", SubstituteQuantity: 5, OriginalQtyCovered: 5}},
		})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("stage a refund", func(t *testing.T) {
		lines := &stubSalesOrderLineItemRepo{findByIDResult: &omsDomain.SalesOrderLineItem{ID: "li-1", SalesOrderID: "order-1", SKUID: strPtr("sku-1"), Quantity: 3}}
		uc := appoms.NewRefundLineItemUseCase(newMockGormDB(t), inProgressPacking(), &stubPackScanRepo{}, lines, &stubGroupPermissionRepo{}, failingOutbox())
		_, err := uc.Execute(ctx, appoms.RefundLineItemInput{SessionID: "session-1", LineItemID: "li-1", Quantity: 1, Reason: "damaged", ScannedBy: "user-1"})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("complete packing", func(t *testing.T) {
		f := newCompletePackingFixture([]omsDomain.SalesOrderLineItem{widget()},
			[]omsDomain.PackScan{{LineItemID: strPtrLI("li-1"), SKUID: "sku-1", ActionType: omsDomain.PackScanActionPack, Quantity: 2}})
		f.outbox.Err = errOMSOutboxDown
		_, err := f.useCase(t).Execute(ctx, appoms.CompletePackingInput{SessionID: "session-1", CompletedBy: "user-1"})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})
}

// Completing picking applies the changes staged during it (see
// OrderChangeApplier); each applied change and the completion itself
// record events, so any of them failing fails the completion.
func TestCompletePicking_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	ctx := context.Background()
	completePicking := func(t *testing.T, failSubject string, order *omsDomain.SalesOrder, line omsDomain.SalesOrderLineItem, scan omsDomain.PickScan) error {
		t.Helper()
		sessions, orders := completablePickingFixtures()
		if order != nil {
			orders = &stubSalesOrderRepo{findByIDResult: order}
		}
		links := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
			"li-1": {{SalesOrderLineItemID: "li-1", QuantityDelta: -line.Quantity}},
		}}
		resolver := &stubPaymentRefundClientResolver{resolveResult: &stubPaymentLinkClient{refundRef: "re_1"}, resolveOK: true}
		uc := appoms.NewCompletePickingUseCase(newMockGormDBN(t, 10), sessions, orders, &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{line}},
			links, &stubPickScanRepo{listBySession: []omsDomain.PickScan{scan}}, &stubAddLineItemForPickCompletion{}, &stubRecordMovement{}, &stubReverseMovement{},
			&stubStockAvailabilityChecker{}, &stubSKUFinderForShortfall{}, &subjectFailingOutbox{subject: failSubject}, resolver)
		_, err := uc.Execute(ctx, appoms.CompletePickingInput{SessionID: "session-1", CompletedBy: "user-1"})
		return err
	}
	widgetLine := omsDomain.SalesOrderLineItem{ID: "li-1", SKUID: strPtr("sku-1"), Name: "Widget", Quantity: 2, PriceMinorUnits: 500}
	covered := int64(2)

	t.Run("plain completion", func(t *testing.T) {
		err := completePicking(t, "oms.picking_session.completed", nil, widgetLine,
			omsDomain.PickScan{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionPick, Quantity: 2})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	t.Run("applied substitution", func(t *testing.T) {
		err := completePicking(t, "oms.sales_order_line_item.substituted", nil, widgetLine, omsDomain.PickScan{
			ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-sub", ActionType: omsDomain.PickScanActionSubstitute,
			Quantity: 2, OriginalQtyCovered: &covered, ReturnToStock: true,
		})
		assert.ErrorIs(t, err, errOMSOutboxDown)
	})

	refund := omsDomain.PickScan{ID: "scan-1", LineItemID: strPtr("li-1"), SKUID: "sku-1", ActionType: omsDomain.PickScanActionRefund, Quantity: 1, Reason: "damaged"}
	paidByCard := func() *omsDomain.SalesOrder {
		return &omsDomain.SalesOrder{ID: "order-1", WarehouseID: "warehouse-1", PaymentProvider: "stripe", PaymentExternalID: "cs_1"}
	}
	for _, subject := range []string{"oms.sales_order_line_item.updated", "oms.sales_order.updated", "oms.sales_order.status_changed", "payments.refund.issued"} {
		t.Run("refund issued through the provider: "+subject, func(t *testing.T) {
			assert.ErrorIs(t, completePicking(t, subject, paidByCard(), widgetLine, refund), errOMSOutboxDown)
		})
	}
	for _, subject := range []string{"oms.sales_order_line_item.updated", "oms.sales_order.updated", "payments.refund.manual_review_required"} {
		t.Run("refund left for manual review: "+subject, func(t *testing.T) {
			assert.ErrorIs(t, completePicking(t, subject, nil, widgetLine, refund), errOMSOutboxDown)
		})
	}
}
