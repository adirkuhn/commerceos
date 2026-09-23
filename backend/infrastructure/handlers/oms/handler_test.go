package oms_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
)

func init() { gin.SetMode(gin.TestMode) }

func strPtr(s string) *string { return &s }

type stubListSalesOrders struct {
	out      *appoms.ListSalesOrdersResult
	err      error
	gotInput appoms.ListSalesOrdersInput
}

func (s *stubListSalesOrders) Execute(_ context.Context, in appoms.ListSalesOrdersInput) (*appoms.ListSalesOrdersResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ListSalesOrdersResult{}, nil
}

type stubGetSalesOrder struct {
	out *appoms.GetSalesOrderResult
	err error
}

func (s *stubGetSalesOrder) Execute(_ context.Context, _ string) (*appoms.GetSalesOrderResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.GetSalesOrderResult{}, nil
}

type stubCreateSalesOrder struct {
	out      *appoms.CreateSalesOrderResult
	err      error
	gotInput appoms.CreateSalesOrderInput
}

func (s *stubCreateSalesOrder) Execute(_ context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.CreateSalesOrderResult{}, nil
}

type stubUpdateSalesOrder struct {
	out   *omsDomain.SalesOrder
	err   error
	gotID string
	gotIn appoms.UpdateSalesOrderInput
}

func (s *stubUpdateSalesOrder) Execute(_ context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	s.gotID = id
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.SalesOrder{ID: id}, nil
}

type stubListSalesOrderActivities struct {
	out []omsDomain.SalesOrderActivity
	err error
}

func (s *stubListSalesOrderActivities) Execute(_ context.Context, _ string) ([]omsDomain.SalesOrderActivity, error) {
	return s.out, s.err
}

type stubBulkUpdateSalesOrderStatus struct {
	out      *appoms.BulkUpdateSalesOrderStatusResult
	err      error
	gotInput appoms.BulkUpdateSalesOrderStatusInput
}

func (s *stubBulkUpdateSalesOrderStatus) Execute(_ context.Context, in appoms.BulkUpdateSalesOrderStatusInput) (*appoms.BulkUpdateSalesOrderStatusResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.BulkUpdateSalesOrderStatusResult{}, nil
}

type stubDeleteSalesOrder struct {
	err   error
	gotID string
}

func (s *stubDeleteSalesOrder) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubLinkLineItemSKU struct {
	out   *omsDomain.SalesOrder
	err   error
	gotIn appoms.LinkLineItemSKUInput
}

func (s *stubLinkLineItemSKU) Execute(_ context.Context, in appoms.LinkLineItemSKUInput) (*omsDomain.SalesOrder, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.SalesOrder{ID: in.SalesOrderID}, nil
}

type stubRetryStockDecrement struct {
	out      *appoms.RetryStockDecrementResult
	err      error
	gotOrder string
}

func (s *stubRetryStockDecrement) Execute(_ context.Context, salesOrderID string) (*appoms.RetryStockDecrementResult, error) {
	s.gotOrder = salesOrderID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.RetryStockDecrementResult{Order: omsDomain.SalesOrder{ID: salesOrderID}}, nil
}

type stubForceAcceptStock struct {
	out      *appoms.ForceAcceptStockResult
	err      error
	gotOrder string
}

func (s *stubForceAcceptStock) Execute(_ context.Context, salesOrderID string) (*appoms.ForceAcceptStockResult, error) {
	s.gotOrder = salesOrderID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ForceAcceptStockResult{Order: omsDomain.SalesOrder{ID: salesOrderID}}, nil
}

type stubAddSalesOrderLineItem struct {
	out   *omsDomain.SalesOrderLineItem
	err   error
	gotIn appoms.AddSalesOrderLineItemInput
}

func (s *stubAddSalesOrderLineItem) Execute(_ context.Context, in appoms.AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.SalesOrderLineItem{ID: "line-1", SalesOrderID: in.SalesOrderID}, nil
}

type stubStartOrResumePicking struct {
	out        *omsDomain.PickingSession
	err        error
	gotOrderID string
	gotUserID  string
}

func (s *stubStartOrResumePicking) Execute(_ context.Context, salesOrderID, userID string) (*omsDomain.PickingSession, error) {
	s.gotOrderID, s.gotUserID = salesOrderID, userID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PickingSession{ID: "session-1", SalesOrderID: salesOrderID}, nil
}

type stubGetPickingSession struct {
	out *appoms.PickingSessionDetail
	err error
}

func (s *stubGetPickingSession) Execute(_ context.Context, id string) (*appoms.PickingSessionDetail, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.PickingSessionDetail{Session: omsDomain.PickingSession{ID: id}}, nil
}

type stubTouchPickingSession struct {
	err          error
	gotSessionID string
	gotUserID    string
}

func (s *stubTouchPickingSession) Execute(_ context.Context, sessionID, userID string) error {
	s.gotSessionID, s.gotUserID = sessionID, userID
	return s.err
}

type stubResetPickingSession struct {
	out   *omsDomain.PickingSession
	err   error
	gotID string
}

func (s *stubResetPickingSession) Execute(_ context.Context, id string) (*omsDomain.PickingSession, error) {
	s.gotID = id
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PickingSession{ID: id, Status: omsDomain.PickingSessionStatusInProgress}, nil
}

type stubScanPickItem struct {
	out   *appoms.ScanPickItemResult
	err   error
	gotIn appoms.ScanPickItemInput
}

func (s *stubScanPickItem) Execute(_ context.Context, in appoms.ScanPickItemInput) (*appoms.ScanPickItemResult, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ScanPickItemResult{Scan: omsDomain.PickScan{ID: "scan-1", SessionID: in.SessionID}}, nil
}

type stubSubstituteLineItem struct {
	out   *appoms.SubstituteLineItemResult
	err   error
	gotIn appoms.SubstituteLineItemInput
}

func (s *stubSubstituteLineItem) Execute(_ context.Context, in appoms.SubstituteLineItemInput) (*appoms.SubstituteLineItemResult, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	scans := make([]omsDomain.PickScan, len(in.Lines))
	lineItemID := in.LineItemID
	for i, l := range in.Lines {
		covered := l.OriginalQtyCovered
		scans[i] = omsDomain.PickScan{
			ID: fmt.Sprintf("scan-%d", i+1), SessionID: in.SessionID, LineItemID: &lineItemID, SKUID: l.SKUID,
			ActionType: omsDomain.PickScanActionSubstitute, Quantity: l.SubstituteQuantity,
			OriginalQtyCovered: &covered, ReturnToStock: l.ReturnToStock,
		}
	}
	return &appoms.SubstituteLineItemResult{Scans: scans}, nil
}

type stubRefundLineItem struct {
	out   *appoms.RefundLineItemResult
	err   error
	gotIn appoms.RefundLineItemInput
}

func (s *stubRefundLineItem) Execute(_ context.Context, in appoms.RefundLineItemInput) (*appoms.RefundLineItemResult, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	lineItemID := in.LineItemID
	return &appoms.RefundLineItemResult{Scan: omsDomain.PickScan{
		ID: "scan-1", SessionID: in.SessionID, LineItemID: &lineItemID, ActionType: omsDomain.PickScanActionRefund, Quantity: in.Quantity,
	}}, nil
}

type stubResolvePickScanCode struct {
	out          *appoms.ResolvePickScanCodeResult
	err          error
	gotSessionID string
	gotCode      string
}

func (s *stubResolvePickScanCode) Execute(_ context.Context, sessionID, code string) (*appoms.ResolvePickScanCodeResult, error) {
	s.gotSessionID, s.gotCode = sessionID, code
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ResolvePickScanCodeResult{}, nil
}

type stubDeletePickScan struct {
	err          error
	gotSessionID string
	gotScanID    string
}

func (s *stubDeletePickScan) Execute(_ context.Context, sessionID, scanID string) error {
	s.gotSessionID, s.gotScanID = sessionID, scanID
	return s.err
}

type stubPausePicking struct {
	out   *omsDomain.PickingSession
	err   error
	gotID string
}

func (s *stubPausePicking) Execute(_ context.Context, id string) (*omsDomain.PickingSession, error) {
	s.gotID = id
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PickingSession{ID: id}, nil
}

type stubCompletePicking struct {
	out            *appoms.CompletePickingResult
	err            error
	gotID          string
	gotCompletedBy string
	gotIn          appoms.CompletePickingInput
}

func (s *stubCompletePicking) Execute(_ context.Context, in appoms.CompletePickingInput) (*appoms.CompletePickingResult, error) {
	s.gotID, s.gotCompletedBy, s.gotIn = in.SessionID, in.CompletedBy, in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.CompletePickingResult{Completed: true, Session: omsDomain.PickingSession{ID: in.SessionID}}, nil
}

type stubStartOrResumePacking struct {
	out        *omsDomain.PackingSession
	err        error
	gotOrderID string
	gotUserID  string
}

func (s *stubStartOrResumePacking) Execute(_ context.Context, salesOrderID, userID string) (*omsDomain.PackingSession, error) {
	s.gotOrderID, s.gotUserID = salesOrderID, userID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PackingSession{ID: "pack-session-1", SalesOrderID: salesOrderID}, nil
}

type stubGetPackingSession struct {
	out *appoms.PackingSessionDetail
	err error
}

func (s *stubGetPackingSession) Execute(_ context.Context, id string) (*appoms.PackingSessionDetail, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.PackingSessionDetail{Session: omsDomain.PackingSession{ID: id}}, nil
}

type stubTouchPackingSession struct {
	err          error
	gotSessionID string
	gotUserID    string
}

func (s *stubTouchPackingSession) Execute(_ context.Context, sessionID, userID string) error {
	s.gotSessionID, s.gotUserID = sessionID, userID
	return s.err
}

type stubResetPackingSession struct {
	out   *omsDomain.PackingSession
	err   error
	gotID string
}

func (s *stubResetPackingSession) Execute(_ context.Context, id string) (*omsDomain.PackingSession, error) {
	s.gotID = id
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PackingSession{ID: id, Status: omsDomain.PackingSessionStatusInProgress}, nil
}

type stubScanPackItem struct {
	out   *appoms.ScanPackItemResult
	err   error
	gotIn appoms.ScanPackItemInput
}

func (s *stubScanPackItem) Execute(_ context.Context, in appoms.ScanPackItemInput) (*appoms.ScanPackItemResult, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ScanPackItemResult{Scan: omsDomain.PackScan{ID: "pack-scan-1", SessionID: in.SessionID}}, nil
}

type stubDeletePackScan struct {
	err          error
	gotSessionID string
	gotScanID    string
}

func (s *stubDeletePackScan) Execute(_ context.Context, sessionID, scanID string) error {
	s.gotSessionID, s.gotScanID = sessionID, scanID
	return s.err
}

type stubCompletePacking struct {
	out   *appoms.CompletePackingResult
	err   error
	gotIn appoms.CompletePackingInput
}

func (s *stubCompletePacking) Execute(_ context.Context, in appoms.CompletePackingInput) (*appoms.CompletePackingResult, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.CompletePackingResult{Session: omsDomain.PackingSession{ID: in.SessionID, Status: omsDomain.PackingSessionStatusCompleted}}, nil
}

func newTestRouter(list *stubListSalesOrders, get *stubGetSalesOrder, create *stubCreateSalesOrder, update *stubUpdateSalesOrder, del *stubDeleteSalesOrder, link *stubLinkLineItemSKU, retry *stubRetryStockDecrement, force *stubForceAcceptStock) *gin.Engine {
	return newTestRouterWithPicking(list, get, create, update, del, link, retry, force,
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, &stubBulkUpdateSalesOrderStatus{})
}

func newTestRouterWithPicking(
	list *stubListSalesOrders, get *stubGetSalesOrder, create *stubCreateSalesOrder, update *stubUpdateSalesOrder,
	del *stubDeleteSalesOrder, link *stubLinkLineItemSKU, retry *stubRetryStockDecrement, force *stubForceAcceptStock,
	addLineItem *stubAddSalesOrderLineItem, startPicking *stubStartOrResumePicking, getPicking *stubGetPickingSession,
	touchPicking *stubTouchPickingSession, resetPicking *stubResetPickingSession,
	scan *stubScanPickItem, substitute *stubSubstituteLineItem, refund *stubRefundLineItem, resolve *stubResolvePickScanCode, deleteScan *stubDeletePickScan,
	pause *stubPausePicking, complete *stubCompletePicking, bulk *stubBulkUpdateSalesOrderStatus,
) *gin.Engine {
	h := omshandler.NewHandler(list, get, create, update, del, link, retry, force,
		addLineItem, startPicking, getPicking, touchPicking, resetPicking, scan, substitute, refund, resolve, deleteScan, pause, complete, bulk,
		&stubListSalesOrderActivities{},
		&stubStartOrResumePacking{}, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{})
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func newTestRouterWithPacking(packStubs packingStubs) *gin.Engine {
	h := omshandler.NewHandler(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, &stubBulkUpdateSalesOrderStatus{},
		&stubListSalesOrderActivities{},
		packStubs.start, packStubs.get, packStubs.touch, packStubs.reset, packStubs.scan, packStubs.deleteScan, packStubs.complete)
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

type packingStubs struct {
	start      *stubStartOrResumePacking
	get        *stubGetPackingSession
	touch      *stubTouchPackingSession
	reset      *stubResetPackingSession
	scan       *stubScanPackItem
	deleteScan *stubDeletePackScan
	complete   *stubCompletePacking
}

func newPackingStubs() packingStubs {
	return packingStubs{
		start: &stubStartOrResumePacking{}, get: &stubGetPackingSession{}, touch: &stubTouchPackingSession{},
		reset: &stubResetPackingSession{}, scan: &stubScanPackItem{}, deleteScan: &stubDeletePackScan{}, complete: &stubCompletePacking{},
	}
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListSalesOrders_success(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{
		SalesOrders: []omsDomain.SalesOrder{{ID: "so-1", ExternalOrderID: "pos-1", Status: omsDomain.SalesOrderStatusCompleted}},
		Total:       1,
	}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pos-1")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

// TestListSalesOrders_includesFulfilled guards toGenSalesOrder against
// silently dropping the fulfilled field from every response -- it did once
// (the field existed on the domain type and the generated DTO but was never
// copied across in the mapping literal), which rendered every order
// "Unfulfilled" in the UI regardless of its real DB value, with no error
// anywhere to surface the gap.
func TestListSalesOrders_includesFulfilled(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{
		SalesOrders: []omsDomain.SalesOrder{{ID: "so-1", ExternalOrderID: "pos-1", Status: omsDomain.SalesOrderStatusCompleted, Fulfilled: true}},
		Total:       1,
	}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"fulfilled":true`)
}

func TestListSalesOrders_passesSortByAndSortDirQueryParamsThrough(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?sort_by=total&sort_dir=asc", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, omsDomain.SalesOrderSortTotal, list.gotInput.SortBy)
	assert.Equal(t, omsDomain.SortAsc, list.gotInput.SortDir)
}

func TestGetSalesOrder_success(t *testing.T) {
	get := &stubGetSalesOrder{out: &appoms.GetSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", ExternalOrderID: "pos-1", Fulfilled: true},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2}},
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sku-1")
	assert.Contains(t, w.Body.String(), `"fulfilled":true`)
}

func TestGetSalesOrder_notFound(t *testing.T) {
	get := &stubGetSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSalesOrderActivities_success(t *testing.T) {
	list := &stubListSalesOrderActivities{out: []omsDomain.SalesOrderActivity{
		{ID: "a-1", SalesOrderID: "so-1", UserID: "user-1", ActivityType: "myparcel_shipment_created", Summary: "Shipment created", Metadata: []byte(`{"barcode":"3SMYPA1"}`)},
	}}
	w := doRequest(newTestRouterWithActivities(list), "GET", "/api/v1/sales-orders/so-1/activities", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "myparcel_shipment_created")
	assert.Contains(t, w.Body.String(), "3SMYPA1")
}

func TestListSalesOrderActivities_emptyList(t *testing.T) {
	w := doRequest(newTestRouterWithActivities(&stubListSalesOrderActivities{}), "GET", "/api/v1/sales-orders/so-1/activities", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"activities":[]`)
}

func TestCreateSalesOrder_success(t *testing.T) {
	create := &stubCreateSalesOrder{out: &appoms.CreateSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: "generated-1"},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2}},
	}}
	body := map[string]any{
		"warehouse_id": "wh-1", "currency_code": "USD", "occurred_at": "2026-01-01T00:00:00Z",
		"line_items": []map[string]any{{"sku_id": "sku-1", "quantity": 2, "price_minor_units": 500}},
	}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, create, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "POST", "/api/v1/sales-orders", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "so-1")
	assert.Equal(t, "wh-1", create.gotInput.WarehouseID)
	assert.Equal(t, int64(2), create.gotInput.LineItems[0].Quantity)
}

func TestCreateSalesOrder_emptyLineItemsIsUnprocessable(t *testing.T) {
	create := &stubCreateSalesOrder{err: omsDomain.ErrEmptyLineItems}
	body := map[string]any{
		"warehouse_id": "wh-1", "currency_code": "USD", "occurred_at": "2026-01-01T00:00:00Z",
		"line_items": []map[string]any{{"sku_id": "sku-1", "quantity": 2, "price_minor_units": 500}},
	}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, create, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "POST", "/api/v1/sales-orders", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSalesOrder_success(t *testing.T) {
	update := &stubUpdateSalesOrder{out: &omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusCancelled}}
	body := map[string]any{"status": "cancelled"}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, update, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "PATCH", "/api/v1/sales-orders/so-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", update.gotID)
	require.NotNil(t, update.gotIn.Status)
	assert.Equal(t, omsDomain.SalesOrderStatusCancelled, *update.gotIn.Status)
	assert.Contains(t, w.Body.String(), "cancelled")
}

func TestUpdateSalesOrder_notFound(t *testing.T) {
	update := &stubUpdateSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, update, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "PATCH", "/api/v1/sales-orders/missing", map[string]any{"customer_note": "x"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSalesOrder_success(t *testing.T) {
	del := &stubDeleteSalesOrder{}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, del, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "DELETE", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "so-1", del.gotID)
}

func TestDeleteSalesOrder_notFound(t *testing.T) {
	del := &stubDeleteSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, del, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "DELETE", "/api/v1/sales-orders/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSalesOrders_passesNeedsReviewFiltersThrough(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?needs_stock_review=true&needs_sku_review=false", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, list.gotInput.NeedsStockReview)
	assert.True(t, *list.gotInput.NeedsStockReview)
	require.NotNil(t, list.gotInput.NeedsSKUReview)
	assert.False(t, *list.gotInput.NeedsSKUReview)
}

// TestListSalesOrders_passesFulfilledAndStatusFiltersThrough guards against
// the query params being accepted by the route but silently dropped before
// reaching the use case (fulfilled previously was -- the picking screens'
// "orders to pick" search depends on both actually reaching the repo).
func TestListSalesOrders_passesFulfilledAndStatusFiltersThrough(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?fulfilled=false&status=pending&status=processing&status=on-hold", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, list.gotInput.Fulfilled)
	assert.False(t, *list.gotInput.Fulfilled)
	assert.Equal(t, []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold,
	}, list.gotInput.Status)
}

func TestLinkLineItemSKU_success(t *testing.T) {
	link := &stubLinkLineItemSKU{out: &omsDomain.SalesOrder{ID: "so-1", NeedsSKUReview: false}}
	body := map[string]any{"sku_id": "sku-1"}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, link, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/so-1/line-items/li-1/link-sku", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", link.gotIn.SalesOrderID)
	assert.Equal(t, "li-1", link.gotIn.LineItemID)
	assert.Equal(t, "sku-1", link.gotIn.SKUID)
	assert.Contains(t, w.Body.String(), "so-1")
}

func TestLinkLineItemSKU_lineItemNotOnOrderIsUnprocessable(t *testing.T) {
	link := &stubLinkLineItemSKU{err: omsDomain.ErrLineItemNotOnOrder}
	body := map[string]any{"sku_id": "sku-1"}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, link, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/so-1/line-items/li-1/link-sku", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLinkLineItemSKU_unknownSKUIsNotFound(t *testing.T) {
	link := &stubLinkLineItemSKU{err: pmsDomain.ErrSKUNotFound}
	body := map[string]any{"sku_id": "missing-sku"}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, link, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/so-1/line-items/li-1/link-sku", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRetryStockDecrement_success(t *testing.T) {
	retry := &stubRetryStockDecrement{out: &appoms.RetryStockDecrementResult{
		Order: omsDomain.SalesOrder{ID: "so-1", NeedsStockReview: false}, Resolved: 2, StillPending: 1,
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, retry, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/so-1/retry-stock", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", retry.gotOrder)
	assert.Contains(t, w.Body.String(), `"resolved":2`)
	assert.Contains(t, w.Body.String(), `"still_pending":1`)
}

func TestRetryStockDecrement_orderNotStockReducingIsUnprocessable(t *testing.T) {
	retry := &stubRetryStockDecrement{err: omsDomain.ErrOrderNotStockReducing}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, retry, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/so-1/retry-stock", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRetryStockDecrement_notFound(t *testing.T) {
	retry := &stubRetryStockDecrement{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, retry, &stubForceAcceptStock{}),
		"POST", "/api/v1/sales-orders/missing/retry-stock", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestForceAcceptStock_success(t *testing.T) {
	force := &stubForceAcceptStock{out: &appoms.ForceAcceptStockResult{
		Order: omsDomain.SalesOrder{ID: "so-1", NeedsStockReview: false}, Resolved: 2,
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, force),
		"POST", "/api/v1/sales-orders/so-1/force-accept-stock", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", force.gotOrder)
	assert.Contains(t, w.Body.String(), `"resolved":2`)
}

func TestForceAcceptStock_notFound(t *testing.T) {
	force := &stubForceAcceptStock{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, force),
		"POST", "/api/v1/sales-orders/missing/force-accept-stock", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBulkUpdateSalesOrderStatus_success(t *testing.T) {
	bulk := &stubBulkUpdateSalesOrderStatus{out: &appoms.BulkUpdateSalesOrderStatusResult{Updated: []string{"so-1", "so-2"}}}
	body := map[string]any{"order_ids": []string{"so-1", "so-2"}, "status": "processing"}
	w := doRequest(newTestRouterWithPicking(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, bulk),
		"POST", "/api/v1/sales-orders/bulk-status", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"so-1", "so-2"}, bulk.gotInput.OrderIDs)
	require.NotNil(t, bulk.gotInput.Status)
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, *bulk.gotInput.Status)
	assert.Contains(t, w.Body.String(), "so-1")
}

func TestBulkUpdateSalesOrderStatus_emptyOrderIDsIsUnprocessable(t *testing.T) {
	bulk := &stubBulkUpdateSalesOrderStatus{err: omsDomain.ErrBulkUpdateOrderIDsRequired}
	body := map[string]any{"order_ids": []string{"so-1"}, "fulfilled": true}
	w := doRequest(newTestRouterWithPicking(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, bulk),
		"POST", "/api/v1/sales-orders/bulk-status", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func routerWithPicking(overrides func(*pickingStubs)) *gin.Engine {
	s := &pickingStubs{
		addLineItem: &stubAddSalesOrderLineItem{}, startPicking: &stubStartOrResumePicking{}, getPicking: &stubGetPickingSession{},
		touchPicking: &stubTouchPickingSession{}, resetPicking: &stubResetPickingSession{},
		scan: &stubScanPickItem{}, substitute: &stubSubstituteLineItem{}, refund: &stubRefundLineItem{}, resolve: &stubResolvePickScanCode{}, deleteScan: &stubDeletePickScan{},
		pause: &stubPausePicking{}, complete: &stubCompletePicking{},
	}
	if overrides != nil {
		overrides(s)
	}
	return newTestRouterWithPicking(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		s.addLineItem, s.startPicking, s.getPicking, s.touchPicking, s.resetPicking, s.scan, s.substitute, s.refund, s.resolve, s.deleteScan, s.pause, s.complete, &stubBulkUpdateSalesOrderStatus{})
}

func routerWithPacking(overrides func(*packingStubs)) *gin.Engine {
	s := newPackingStubs()
	if overrides != nil {
		overrides(&s)
	}
	return newTestRouterWithPacking(s)
}

func newTestRouterWithActivities(activities *stubListSalesOrderActivities) *gin.Engine {
	h := omshandler.NewHandler(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{},
		&stubBulkUpdateSalesOrderStatus{}, activities,
		&stubStartOrResumePacking{}, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{})
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

type pickingStubs struct {
	addLineItem  *stubAddSalesOrderLineItem
	startPicking *stubStartOrResumePicking
	getPicking   *stubGetPickingSession
	touchPicking *stubTouchPickingSession
	resetPicking *stubResetPickingSession
	scan         *stubScanPickItem
	substitute   *stubSubstituteLineItem
	refund       *stubRefundLineItem
	resolve      *stubResolvePickScanCode
	deleteScan   *stubDeletePickScan
	pause        *stubPausePicking
	complete     *stubCompletePicking
}

func TestAddSalesOrderLineItem_success(t *testing.T) {
	var addLineItem *stubAddSalesOrderLineItem
	body := map[string]any{"sku_id": "sku-1", "name": "Freebie", "quantity": 2, "price_minor_units": 0}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { addLineItem = s.addLineItem }),
		"POST", "/api/v1/sales-orders/so-1/line-items", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "so-1", addLineItem.gotIn.SalesOrderID)
	assert.Equal(t, "sku-1", addLineItem.gotIn.SKUID)
	assert.Equal(t, int64(2), addLineItem.gotIn.Quantity)
}

func TestAddSalesOrderLineItem_notFound(t *testing.T) {
	body := map[string]any{"sku_id": "sku-1", "quantity": 1, "price_minor_units": 0}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.addLineItem.err = omsDomain.ErrSalesOrderNotFound }),
		"POST", "/api/v1/sales-orders/missing/line-items", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddSalesOrderLineItem_invalidQuantityIsUnprocessable(t *testing.T) {
	body := map[string]any{"sku_id": "sku-1", "quantity": 1, "price_minor_units": 0}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.addLineItem.err = omsDomain.ErrInvalidLineItemQuantity }),
		"POST", "/api/v1/sales-orders/so-1/line-items", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestStartOrResumePicking_success(t *testing.T) {
	var startPicking *stubStartOrResumePicking
	w := doRequest(routerWithPicking(func(s *pickingStubs) { startPicking = s.startPicking }),
		"POST", "/api/v1/sales-orders/so-1/picking", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", startPicking.gotOrderID)
	assert.Contains(t, w.Body.String(), "session-1")
}

func TestStartOrResumePicking_alreadyFulfilledIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.startPicking.err = omsDomain.ErrOrderAlreadyFulfilled }),
		"POST", "/api/v1/sales-orders/so-1/picking", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestStartOrResumePicking_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.startPicking.err = omsDomain.ErrSalesOrderNotFound }),
		"POST", "/api/v1/sales-orders/missing/picking", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStartOrResumePicking_lockedByAnotherUserIsConflict(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.startPicking.err = omsDomain.ErrPickingSessionLocked }),
		"POST", "/api/v1/sales-orders/so-1/picking", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTouchPickingSession_success(t *testing.T) {
	var touch *stubTouchPickingSession
	w := doRequest(routerWithPicking(func(s *pickingStubs) { touch = s.touchPicking }),
		"POST", "/api/v1/picking-sessions/session-1/touch", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "session-1", touch.gotSessionID)
}

func TestTouchPickingSession_lockedByAnotherUserIsConflict(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.touchPicking.err = omsDomain.ErrPickingSessionLocked }),
		"POST", "/api/v1/picking-sessions/session-1/touch", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTouchPickingSession_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.touchPicking.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/touch", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestResetPickingSession_success(t *testing.T) {
	var reset *stubResetPickingSession
	w := doRequest(routerWithPicking(func(s *pickingStubs) { reset = s.resetPicking }),
		"POST", "/api/v1/picking-sessions/session-1/reset", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", reset.gotID)
	assert.Contains(t, w.Body.String(), "session-1")
}

func TestResetPickingSession_alreadyCompletedIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.resetPicking.err = omsDomain.ErrPickingSessionAlreadyCompleted }),
		"POST", "/api/v1/picking-sessions/session-1/reset", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResetPickingSession_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.resetPicking.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/reset", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPickingSession_success(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) {
		s.getPicking.out = &appoms.PickingSessionDetail{Session: omsDomain.PickingSession{ID: "session-1"}}
	}), "GET", "/api/v1/picking-sessions/session-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "session-1")
}

func TestGetPickingSession_mapsProductIDForThumbnails(t *testing.T) {
	productID := "product-1"
	w := doRequest(routerWithPicking(func(s *pickingStubs) {
		s.getPicking.out = &appoms.PickingSessionDetail{
			Session: omsDomain.PickingSession{ID: "session-1"},
			LineItems: []omsDomain.SalesOrderLineItem{
				{ID: "li-1", ProductID: &productID},
				{ID: "li-2"}, // unmatched line, no ProductID
			},
		}
	}), "GET", "/api/v1/picking-sessions/session-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"product_id":"product-1"`)
}

func TestGetPickingSession_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.getPicking.err = omsDomain.ErrPickingSessionNotFound }),
		"GET", "/api/v1/picking-sessions/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestScanPickItem_success(t *testing.T) {
	var scan *stubScanPickItem
	body := map[string]any{"sku_id": "sku-1", "quantity": 1}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { scan = s.scan }),
		"POST", "/api/v1/picking-sessions/session-1/scan", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", scan.gotIn.SessionID)
	assert.Equal(t, "sku-1", scan.gotIn.SKUID)
}

func TestScanPickItem_lineItemMismatchIsUnprocessable(t *testing.T) {
	body := map[string]any{"sku_id": "sku-1", "quantity": 1}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.scan.err = omsDomain.ErrLineItemSKUMismatch }),
		"POST", "/api/v1/picking-sessions/session-1/scan", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScanPickItem_notFound(t *testing.T) {
	body := map[string]any{"sku_id": "sku-1", "quantity": 1}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.scan.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/scan", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubstituteLineItem_success(t *testing.T) {
	var substitute *stubSubstituteLineItem
	body := map[string]any{
		"line_item_id": "li-1",
		"lines": []map[string]any{
			{"sku_id": "sku-a", "substitute_quantity": 3, "original_quantity_covered": 1, "return_to_stock": false},
			{"sku_id": "sku-b", "substitute_quantity": 4, "original_quantity_covered": 1, "return_to_stock": true},
		},
	}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { substitute = s.substitute }),
		"POST", "/api/v1/picking-sessions/session-1/substitute-batch", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", substitute.gotIn.SessionID)
	assert.Equal(t, "li-1", substitute.gotIn.LineItemID)
	require.Len(t, substitute.gotIn.Lines, 2)
	assert.Equal(t, "sku-a", substitute.gotIn.Lines[0].SKUID)
	assert.Equal(t, int64(3), substitute.gotIn.Lines[0].SubstituteQuantity)
	assert.Equal(t, int64(1), substitute.gotIn.Lines[0].OriginalQtyCovered)
	assert.False(t, substitute.gotIn.Lines[0].ReturnToStock)
	assert.True(t, substitute.gotIn.Lines[1].ReturnToStock)
	assert.Contains(t, w.Body.String(), "sku-a")
	assert.Contains(t, w.Body.String(), "sku-b")
}

func TestSubstituteLineItem_coverageMismatchIsUnprocessable(t *testing.T) {
	body := map[string]any{
		"line_item_id": "li-1",
		"lines":        []map[string]any{{"sku_id": "sku-a", "substitute_quantity": 1, "original_quantity_covered": 1}},
	}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.substitute.err = omsDomain.ErrSubstitutionCoverageMismatch }),
		"POST", "/api/v1/picking-sessions/session-1/substitute-batch", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubstituteLineItem_emptyLinesIsUnprocessable(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "lines": []map[string]any{}}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.substitute.err = omsDomain.ErrSubstituteLinesRequired }),
		"POST", "/api/v1/picking-sessions/session-1/substitute-batch", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubstituteLineItem_notFound(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "lines": []map[string]any{{"sku_id": "sku-a", "substitute_quantity": 1, "original_quantity_covered": 1}}}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.substitute.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/substitute-batch", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRefundLineItem_success(t *testing.T) {
	var refund *stubRefundLineItem
	body := map[string]any{"line_item_id": "li-1", "quantity": 1, "reason": "damaged"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { refund = s.refund }),
		"POST", "/api/v1/picking-sessions/session-1/refund", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", refund.gotIn.SessionID)
	assert.Equal(t, "li-1", refund.gotIn.LineItemID)
	assert.Equal(t, int64(1), refund.gotIn.Quantity)
	assert.Equal(t, "damaged", refund.gotIn.Reason)
	assert.Contains(t, w.Body.String(), "refund")
}

func TestRefundLineItem_invalidQuantityIsUnprocessable(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "quantity": 1}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.refund.err = omsDomain.ErrInvalidRefundQuantity }),
		"POST", "/api/v1/picking-sessions/session-1/refund", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRefundLineItem_permissionDeniedIsForbidden(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "quantity": 1}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.refund.err = omsDomain.ErrPermissionDenied }),
		"POST", "/api/v1/picking-sessions/session-1/refund", body)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestResolvePickScanCode_success(t *testing.T) {
	var resolve *stubResolvePickScanCode
	body := map[string]any{"code": "BARCODE-1"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { resolve = s.resolve }),
		"POST", "/api/v1/picking-sessions/session-1/resolve-scan", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", resolve.gotSessionID)
	assert.Equal(t, "BARCODE-1", resolve.gotCode)
}

func TestResolvePickScanCode_includesWeightGramsForAWeightEmbeddedScan(t *testing.T) {
	weightGrams := int64(462)
	var resolve *stubResolvePickScanCode
	body := map[string]any{"code": "2994721004622"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) {
		resolve = s.resolve
		resolve.out = &appoms.ResolvePickScanCodeResult{SKUID: "sku-1", ProductName: "Dry Meat", WeightGrams: &weightGrams}
	}), "POST", "/api/v1/picking-sessions/session-1/resolve-scan", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"weight_grams":462`)
}

func TestResolvePickScanCode_weightRequiredForPerWeightSKUReturns422(t *testing.T) {
	body := map[string]any{"code": "DRYMEAT-1"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.resolve.err = pmsDomain.ErrWeightRequiredForPerWeightSKU }),
		"POST", "/api/v1/picking-sessions/session-1/resolve-scan", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResolvePickScanCode_notFound(t *testing.T) {
	body := map[string]any{"code": "BARCODE-1"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.resolve.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/resolve-scan", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePickScan_success(t *testing.T) {
	var deleteScan *stubDeletePickScan
	w := doRequest(routerWithPicking(func(s *pickingStubs) { deleteScan = s.deleteScan }),
		"DELETE", "/api/v1/picking-sessions/session-1/scans/scan-1", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "session-1", deleteScan.gotSessionID)
	assert.Equal(t, "scan-1", deleteScan.gotScanID)
}

func TestDeletePickScan_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.deleteScan.err = omsDomain.ErrPickScanNotFound }),
		"DELETE", "/api/v1/picking-sessions/session-1/scans/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPausePicking_success(t *testing.T) {
	var pause *stubPausePicking
	w := doRequest(routerWithPicking(func(s *pickingStubs) { pause = s.pause }),
		"POST", "/api/v1/picking-sessions/session-1/pause", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", pause.gotID)
}

func TestPausePicking_notActiveIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.pause.err = omsDomain.ErrPickingSessionNotActive }),
		"POST", "/api/v1/picking-sessions/session-1/pause", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPausePicking_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.pause.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/pause", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCompletePicking_success(t *testing.T) {
	var complete *stubCompletePicking
	w := doRequest(routerWithPicking(func(s *pickingStubs) { complete = s.complete }),
		"POST", "/api/v1/picking-sessions/session-1/complete", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", complete.gotID)
}

func TestCompletePicking_notFound(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) { s.complete.err = omsDomain.ErrPickingSessionNotFound }),
		"POST", "/api/v1/picking-sessions/missing/complete", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCompletePicking_stockShortfall_returns200WithCompletedFalse(t *testing.T) {
	var complete *stubCompletePicking
	w := doRequest(routerWithPicking(func(s *pickingStubs) {
		complete = s.complete
		complete.out = &appoms.CompletePickingResult{
			Completed: false,
			StockShortfalls: []appoms.StockShortfall{
				{ScanID: "scan-1", SKUID: "sku-1", Name: "Widget", Needed: 5, Short: 3},
			},
		}
	}), "POST", "/api/v1/picking-sessions/session-1/complete", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"completed":false`)
	assert.Contains(t, w.Body.String(), `"stock_shortfalls"`)
	assert.Contains(t, w.Body.String(), `"short":3`)
}

func TestCompletePicking_threadsResolutionsFromRequestBody(t *testing.T) {
	var complete *stubCompletePicking
	body := map[string]any{"resolutions": map[string]string{"scan-1": "with_item"}}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { complete = s.complete }),
		"POST", "/api/v1/picking-sessions/session-1/complete", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, complete.gotIn.Resolutions, "scan-1")
	assert.Equal(t, appoms.StockShortfallWithItem, complete.gotIn.Resolutions["scan-1"])
}
