package oms_test

import (
	"context"
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

func newTestRouter(list *stubListSalesOrders, get *stubGetSalesOrder) *gin.Engine {
	h := omshandler.NewHandler(list, get)
	r := gin.New()
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestListSalesOrders_success(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{
		SalesOrders: []omsDomain.SalesOrder{{ID: "so-1", ExternalOrderID: "pos-1", Status: omsDomain.SalesOrderStatusCompleted}},
		Total:       1,
	}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}), "GET", "/api/v1/sales-orders")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pos-1")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestGetSalesOrder_success(t *testing.T) {
	get := &stubGetSalesOrder{out: &appoms.GetSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1", ExternalOrderID: "pos-1"},
		LineItems:  []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: "sku-1", Quantity: 2}},
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get), "GET", "/api/v1/sales-orders/so-1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sku-1")
}

func TestGetSalesOrder_notFound(t *testing.T) {
	get := &stubGetSalesOrder{err: omsDomain.ErrSalesOrderNotFound}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get), "GET", "/api/v1/sales-orders/missing")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
