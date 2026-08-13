package pos_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpos "github.com/adirkuhn/commerceos/api/pos"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	poshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pos"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListProducts struct {
	out      *apppms.ListProductsResult
	err      error
	gotInput apppms.ListProductsInput
}

func (s *stubListProducts) Execute(_ context.Context, in apppms.ListProductsInput) (*apppms.ListProductsResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &apppms.ListProductsResult{}, nil
}

type stubListStockLevels struct {
	out      *appinventory.ListStockLevelsResult
	err      error
	gotInput appinventory.ListStockLevelsInput
}

func (s *stubListStockLevels) Execute(_ context.Context, in appinventory.ListStockLevelsInput) (*appinventory.ListStockLevelsResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.ListStockLevelsResult{}, nil
}

type stubReceivePOSSalesOrder struct {
	out      *omsDomain.SalesOrder
	err      error
	gotInput appoms.ReceivePOSSalesOrderInput
}

func (s *stubReceivePOSSalesOrder) Execute(_ context.Context, in appoms.ReceivePOSSalesOrderInput) (*omsDomain.SalesOrder, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &omsDomain.SalesOrder{}, nil
}

type stubs struct {
	listProducts    stubListProducts
	listStockLevels stubListStockLevels
	receiveOrder    stubReceivePOSSalesOrder
}

func newTestRouter(s *stubs) *gin.Engine {
	h := poshandler.NewHandler(&s.listProducts, &s.listStockLevels, &s.receiveOrder)
	r := gin.New()
	genpos.RegisterHandlers(r.Group("/api/v1"), h)
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

// ── ListPOSProducts ──────────────────────────────────────────────────────────

func TestListPOSProducts_success(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListProductsResult{
		Products: []pmsDomain.Product{{ID: "p1", Name: "Widget", Slug: "widget", Status: pmsDomain.ProductStatusActive}},
		Total:    1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Widget")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestListPOSProducts_passesUpdatedAfterThrough(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products?updated_after=2026-08-01T00%3A00%3A00Z", nil)

	require.Equal(t, 200, w.Code)
	require.NotNil(t, s.listProducts.gotInput.UpdatedAfter)
	assert.Equal(t, 2026, s.listProducts.gotInput.UpdatedAfter.Year())
}

func TestListPOSProducts_propagatesRepoError(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{err: assert.AnError}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	assert.Equal(t, 500, w.Code)
}

// ── ListPOSStockLevels ───────────────────────────────────────────────────────

func TestListPOSStockLevels_success(t *testing.T) {
	s := &stubs{listStockLevels: stubListStockLevels{out: &appinventory.ListStockLevelsResult{
		Levels: []appinventory.StockLevel{{SKUID: "sku-1", Total: 10, Syncable: 8}},
		Total:  1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/stock-levels", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_id":"sku-1"`)
	assert.Contains(t, w.Body.String(), `"total":10`)
	assert.Contains(t, w.Body.String(), `"syncable":8`)
}

func TestListPOSStockLevels_passesChangedSinceThrough(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/stock-levels?changed_since=2026-08-01T00%3A00%3A00Z", nil)

	require.Equal(t, 200, w.Code)
	require.NotNil(t, s.listStockLevels.gotInput.ChangedSince)
	assert.Equal(t, 2026, s.listStockLevels.gotInput.ChangedSince.Year())
}

// ── CreatePOSSalesOrder ──────────────────────────────────────────────────────

func TestCreatePOSSalesOrder_success(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{out: &omsDomain.SalesOrder{
		ID: "so-1", ExternalOrderID: "pos-1", WarehouseID: "wh-1", CurrencyCode: "USD",
		TotalMinorUnits: 1000, OccurredAt: time.Now(), CreatedAt: time.Now(),
	}}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: "sku-1", Quantity: 2, UnitPriceMinorUnits: 500}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"external_order_id":"pos-1"`)
	require.Len(t, s.receiveOrder.gotInput.LineItems, 1)
	assert.Equal(t, "sku-1", s.receiveOrder.gotInput.LineItems[0].SKUID)
	assert.Equal(t, int64(2), s.receiveOrder.gotInput.LineItems[0].Quantity)
}

func TestCreatePOSSalesOrder_mapsEmptyLineItemsToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: omsDomain.ErrEmptyLineItems}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: "sku-1", Quantity: 1, UnitPriceMinorUnits: 500}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code)
}

func TestCreatePOSSalesOrder_mapsInsufficientStockToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: invDomain.ErrInsufficientStock}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: "sku-1", Quantity: 100, UnitPriceMinorUnits: 500}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code)
}
