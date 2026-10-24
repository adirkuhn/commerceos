package oms_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
)

type stubListSalesOrderLineChanges struct {
	out []appoms.SalesOrderLineChange
	err error
	got string
}

func (s *stubListSalesOrderLineChanges) Execute(_ context.Context, salesOrderID string) ([]appoms.SalesOrderLineChange, error) {
	s.got = salesOrderID
	return s.out, s.err
}

func newTestRouterWithLineChanges(changes *stubListSalesOrderLineChanges) *gin.Engine {
	h := omshandler.NewHandler(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{},
		&stubBulkUpdateSalesOrderStatus{}, &stubListSalesOrderActivities{},
		&stubStartOrResumePacking{}, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{}, &stubBuildPickList{}, &stubListPackingQueue{}, &stubMarkOrderPicked{}, &stubResolvePickScanCode{}, omshandler.SessionLockUseCases{}, &stubListPackingQueue{}, changes)
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListSalesOrderLineChanges_success(t *testing.T) {
	covered := int64(4)
	list := &stubListSalesOrderLineChanges{out: []appoms.SalesOrderLineChange{{
		ID: "c-1", Step: appoms.OrderChangeStepPacking, Action: omsDomain.ChangeSubstitute, LineItemID: strPtr("line-1"),
		SKUID: "sku-2", SKUCode: "FEP-1", ProductName: "Feijão Preto", Quantity: 4, OriginalQtyCovered: &covered,
		ReturnToStock: true, Reason: "out of stock", By: "user-1", At: time.Date(2026, 10, 24, 10, 49, 0, 0, time.UTC),
	}}}

	w := doRequest(newTestRouterWithLineChanges(list), "GET", "/api/v1/sales-orders/so-1/line-changes", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", list.got)
	body := w.Body.String()
	assert.Contains(t, body, `"action":"substitute"`)
	assert.Contains(t, body, `"step":"packing"`)
	assert.Contains(t, body, `"line_item_id":"line-1"`)
	assert.Contains(t, body, `"original_qty_covered":4`)
	assert.Contains(t, body, `"product_name":"Feijão Preto"`)
}

func TestListSalesOrderLineChanges_emptyList(t *testing.T) {
	w := doRequest(newTestRouterWithLineChanges(&stubListSalesOrderLineChanges{}), "GET", "/api/v1/sales-orders/so-1/line-changes", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"changes":[]`)
}

func TestListSalesOrderLineChanges_orderNotFound(t *testing.T) {
	list := &stubListSalesOrderLineChanges{err: omsDomain.ErrSalesOrderNotFound}

	w := doRequest(newTestRouterWithLineChanges(list), "GET", "/api/v1/sales-orders/missing/line-changes", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
