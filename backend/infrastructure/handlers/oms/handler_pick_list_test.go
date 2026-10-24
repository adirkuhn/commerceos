package oms_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBuildPickList struct {
	out    *omsDomain.PickList
	err    error
	gotIDs []string
}

func (s *stubBuildPickList) Execute(_ context.Context, ids []string) (*omsDomain.PickList, error) {
	s.gotIDs = ids
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.PickList{}, nil
}

func newTestRouterWithPickList(pickList *stubBuildPickList, list *stubListSalesOrders) *gin.Engine {
	h := omshandler.NewHandler(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, &stubBulkUpdateSalesOrderStatus{},
		&stubListSalesOrderActivities{},
		&stubStartOrResumePacking{}, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{}, pickList, &stubListPackingQueue{}, &stubMarkOrderPicked{}, &stubResolvePickScanCode{}, omshandler.SessionLockUseCases{}, &stubListPackingQueue{}, &stubListSalesOrderLineChanges{})
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestGetPickList_success(t *testing.T) {
	skuID := "sku-a"
	stub := &stubBuildPickList{out: &omsDomain.PickList{Items: []omsDomain.PickListItem{{
		SKUID: &skuID, SKUCode: "A-1", Barcode: "111", Name: "Apple", TotalQuantity: 3,
		Orders: []omsDomain.PickListOrderLine{
			{SalesOrderID: "so-1", OrderNumber: "1001", LineItemID: "li-1", Quantity: 2},
			{SalesOrderID: "so-2", OrderNumber: "1002", LineItemID: "li-2", Quantity: 1},
		},
	}}}}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}),
		"GET", "/api/v1/sales-orders/pick-list?sales_order_id=so-1&sales_order_id=so-2", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"so-1", "so-2"}, stub.gotIDs)
	var resp genoms.PickListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	require.Len(t, resp.Data.Items, 1)
	item := resp.Data.Items[0]
	require.NotNil(t, item.SkuId)
	assert.Equal(t, "sku-a", *item.SkuId)
	assert.Equal(t, "A-1", item.SkuCode)
	assert.Equal(t, int64(3), item.TotalQuantity)
	require.Len(t, item.Orders, 2)
	assert.Equal(t, "1001", item.Orders[0].OrderNumber)
	assert.Equal(t, "li-1", item.Orders[0].LineItemId)
}

func TestGetPickList_unlinkedLineHasNullSKUID(t *testing.T) {
	stub := &stubBuildPickList{out: &omsDomain.PickList{Items: []omsDomain.PickListItem{{Name: "Gift wrap", TotalQuantity: 1}}}}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}),
		"GET", "/api/v1/sales-orders/pick-list?sales_order_id=so-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_id":null`)
	assert.Contains(t, w.Body.String(), `"orders":[]`, "empty breakdown must serialize as [], not null")
}

func TestGetPickList_missingParamIsBadRequest(t *testing.T) {
	stub := &stubBuildPickList{}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}), "GET", "/api/v1/sales-orders/pick-list", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, stub.gotIDs, "use case must not run without ids")
}

func TestGetPickList_orderNotFound(t *testing.T) {
	stub := &stubBuildPickList{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}),
		"GET", "/api/v1/sales-orders/pick-list?sales_order_id=missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPickList_tooManyOrdersIsUnprocessable(t *testing.T) {
	stub := &stubBuildPickList{err: omsDomain.ErrPickListTooManyOrders}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}),
		"GET", "/api/v1/sales-orders/pick-list?sales_order_id=so-1", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetPickList_emptySelectionIsUnprocessable(t *testing.T) {
	stub := &stubBuildPickList{err: omsDomain.ErrPickListOrderIDsRequired}
	w := doRequest(newTestRouterWithPickList(stub, &stubListSalesOrders{}),
		"GET", "/api/v1/sales-orders/pick-list?sales_order_id=so-1", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
