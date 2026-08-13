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
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
)

func init() { gin.SetMode(gin.TestMode) }

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

func newTestRouter(list *stubListSalesOrders, get *stubGetSalesOrder, create *stubCreateSalesOrder, update *stubUpdateSalesOrder, del *stubDeleteSalesOrder) *gin.Engine {
	h := omshandler.NewHandler(list, get, create, update, del)
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
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}), "GET", "/api/v1/sales-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pos-1")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestGetSalesOrder_success(t *testing.T) {
	get := &stubGetSalesOrder{out: &appoms.GetSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", ExternalOrderID: "pos-1"},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: "sku-1", Quantity: 2}},
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}), "GET", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sku-1")
}

func TestGetSalesOrder_notFound(t *testing.T) {
	get := &stubGetSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}), "GET", "/api/v1/sales-orders/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateSalesOrder_success(t *testing.T) {
	create := &stubCreateSalesOrder{out: &appoms.CreateSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: "generated-1"},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: "sku-1", Quantity: 2}},
	}}
	body := map[string]any{
		"warehouse_id": "wh-1", "currency_code": "USD", "occurred_at": "2026-01-01T00:00:00Z",
		"line_items": []map[string]any{{"sku_id": "sku-1", "quantity": 2, "price_minor_units": 500}},
	}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, create, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}), "POST", "/api/v1/sales-orders", body)

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
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, create, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}), "POST", "/api/v1/sales-orders", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSalesOrder_success(t *testing.T) {
	update := &stubUpdateSalesOrder{out: &omsDomain.SalesOrder{ID: "so-1", Status: omsDomain.SalesOrderStatusCancelled}}
	body := map[string]any{"status": "cancelled"}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, update, &stubDeleteSalesOrder{}), "PATCH", "/api/v1/sales-orders/so-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", update.gotID)
	require.NotNil(t, update.gotIn.Status)
	assert.Equal(t, omsDomain.SalesOrderStatusCancelled, *update.gotIn.Status)
	assert.Contains(t, w.Body.String(), "cancelled")
}

func TestUpdateSalesOrder_notFound(t *testing.T) {
	update := &stubUpdateSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, update, &stubDeleteSalesOrder{}), "PATCH", "/api/v1/sales-orders/missing", map[string]any{"customer_note": "x"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSalesOrder_success(t *testing.T) {
	del := &stubDeleteSalesOrder{}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, del), "DELETE", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "so-1", del.gotID)
}

func TestDeleteSalesOrder_notFound(t *testing.T) {
	del := &stubDeleteSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, del), "DELETE", "/api/v1/sales-orders/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
