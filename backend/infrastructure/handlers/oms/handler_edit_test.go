package oms_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
)

type stubEditSalesOrder struct {
	gotInput appoms.EditSalesOrderInput
	err      error
}

func (s *stubEditSalesOrder) Execute(_ context.Context, in appoms.EditSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	return &appoms.CreateSalesOrderResult{SalesOrder: omsDomain.SalesOrder{ID: in.ID, Number: "CO-1001", TotalMinorUnits: in.TotalMinorUnits},
		LineItems: []omsDomain.SalesOrderLineItem{{ID: "li-1", Quantity: 17}}}, nil
}

func newEditRouter(edit *stubEditSalesOrder) *gin.Engine {
	h := omshandler.NewHandler(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, &stubStartOrResumePicking{}, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, &stubBulkUpdateSalesOrderStatus{},
		&stubListSalesOrderActivities{},
		&stubStartOrResumePacking{}, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{}, &stubBuildPickList{}, &stubListPackingQueue{}, &stubMarkOrderPicked{}, &stubResolvePickScanCode{}, omshandler.SessionLockUseCases{}, &stubListPackingQueue{}, &stubListSalesOrderLineChanges{}).WithOrigins(&stubOriginLister{}).
		WithEditSalesOrder(edit)
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func editBody() map[string]any {
	return map[string]any{
		"warehouse_id": "wh-1", "occurred_at": "2026-10-26T10:00:00Z", "customer_note": "Back door",
		"billing":                    map[string]any{"company": "Claus Hoofddorp B.V."},
		"shipping_total_minor_units": 799, "tax_total_minor_units": 489, "total_minor_units": 5882,
		"line_items": []map[string]any{{"sku_id": "sku-1", "name": "Chá Mate", "quantity": 17, "price_minor_units": 299,
			"total_minor_units": 5083, "total_tax_minor_units": 420, "total_includes_tax": true}},
	}
}

func TestEditSalesOrder_replacesTheOrder(t *testing.T) {
	edit := &stubEditSalesOrder{}

	w := doRequest(newEditRouter(edit), "PUT", "/api/v1/sales-orders/so-1", editBody())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "so-1", edit.gotInput.ID)
	assert.Equal(t, "Back door", edit.gotInput.CustomerNote)
	assert.Equal(t, "Claus Hoofddorp B.V.", edit.gotInput.Billing.Company)
	assert.Equal(t, int64(489), edit.gotInput.TaxTotalMinorUnits)
	require.Len(t, edit.gotInput.LineItems, 1)
	assert.Equal(t, int64(420), edit.gotInput.LineItems[0].TotalTaxMinorUnits)
	assert.True(t, edit.gotInput.LineItems[0].TotalIncludesTax)
	assert.Contains(t, w.Body.String(), `"number":"CO-1001"`)
}

func TestEditSalesOrder_errors(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"not editable": {omsDomain.ErrSalesOrderNotEditable, http.StatusConflict},
		"being picked": {omsDomain.ErrOrderPickingInProgress, http.StatusConflict},
		"not found":    {omsDomain.ErrSalesOrderNotFound, http.StatusNotFound},
		"no lines":     {omsDomain.ErrEmptyLineItems, http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			w := doRequest(newEditRouter(&stubEditSalesOrder{err: tc.err}), "PUT", "/api/v1/sales-orders/so-1", editBody())
			assert.Equal(t, tc.want, w.Code)
		})
	}
}

func TestEditSalesOrder_rejectsABodyThatIsNotJSON(t *testing.T) {
	w := doRequest(newEditRouter(&stubEditSalesOrder{}), "PUT", "/api/v1/sales-orders/so-1", "nope")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
