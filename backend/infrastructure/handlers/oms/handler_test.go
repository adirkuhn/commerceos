package oms_test

import (
	"bytes"
	"context"
	"encoding/json"
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
}

func (s *stubStartOrResumePicking) Execute(_ context.Context, salesOrderID string) (*omsDomain.PickingSession, error) {
	s.gotOrderID = salesOrderID
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
}

func (s *stubCompletePicking) Execute(_ context.Context, id, completedBy string) (*appoms.CompletePickingResult, error) {
	s.gotID, s.gotCompletedBy = id, completedBy
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.CompletePickingResult{Session: omsDomain.PickingSession{ID: id}}, nil
}

func newTestRouter(list *stubListSalesOrders, get *stubGetSalesOrder, create *stubCreateSalesOrder, update *stubUpdateSalesOrder, del *stubDeleteSalesOrder, link *stubLinkLineItemSKU, retry *stubRetryStockDecrement, force *stubForceAcceptStock) *gin.Engine {
	return newTestRouterWithPicking(list, get, create, update, del, link, retry, force,
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubScanPickItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{})
}

func newTestRouterWithPicking(
	list *stubListSalesOrders, get *stubGetSalesOrder, create *stubCreateSalesOrder, update *stubUpdateSalesOrder,
	del *stubDeleteSalesOrder, link *stubLinkLineItemSKU, retry *stubRetryStockDecrement, force *stubForceAcceptStock,
	addLineItem *stubAddSalesOrderLineItem, startPicking *stubStartOrResumePicking, getPicking *stubGetPickingSession,
	scan *stubScanPickItem, resolve *stubResolvePickScanCode, deleteScan *stubDeletePickScan,
	pause *stubPausePicking, complete *stubCompletePicking,
) *gin.Engine {
	h := omshandler.NewHandler(list, get, create, update, del, link, retry, force,
		addLineItem, startPicking, getPicking, scan, resolve, deleteScan, pause, complete)
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
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
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", ExternalOrderID: "pos-1"},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2}},
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sku-1")
}

func TestGetSalesOrder_notFound(t *testing.T) {
	get := &stubGetSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
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

func routerWithPicking(overrides func(*pickingStubs)) *gin.Engine {
	s := &pickingStubs{
		addLineItem: &stubAddSalesOrderLineItem{}, startPicking: &stubStartOrResumePicking{}, getPicking: &stubGetPickingSession{},
		scan: &stubScanPickItem{}, resolve: &stubResolvePickScanCode{}, deleteScan: &stubDeletePickScan{},
		pause: &stubPausePicking{}, complete: &stubCompletePicking{},
	}
	if overrides != nil {
		overrides(s)
	}
	return newTestRouterWithPicking(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		s.addLineItem, s.startPicking, s.getPicking, s.scan, s.resolve, s.deleteScan, s.pause, s.complete)
}

type pickingStubs struct {
	addLineItem  *stubAddSalesOrderLineItem
	startPicking *stubStartOrResumePicking
	getPicking   *stubGetPickingSession
	scan         *stubScanPickItem
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

func TestGetPickingSession_success(t *testing.T) {
	w := doRequest(routerWithPicking(func(s *pickingStubs) {
		s.getPicking.out = &appoms.PickingSessionDetail{Session: omsDomain.PickingSession{ID: "session-1"}}
	}), "GET", "/api/v1/picking-sessions/session-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "session-1")
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

func TestResolvePickScanCode_success(t *testing.T) {
	var resolve *stubResolvePickScanCode
	body := map[string]any{"code": "BARCODE-1"}
	w := doRequest(routerWithPicking(func(s *pickingStubs) { resolve = s.resolve }),
		"POST", "/api/v1/picking-sessions/session-1/resolve-scan", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", resolve.gotSessionID)
	assert.Equal(t, "BARCODE-1", resolve.gotCode)
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
