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

func strPtr(s string) *string { return &s }

type stubListProducts struct {
	out      *apppms.ListPOSProductCatalogResult
	err      error
	gotInput apppms.ListProductsInput
}

func (s *stubListProducts) Execute(_ context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &apppms.ListPOSProductCatalogResult{}, nil
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

type stubListWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s *stubListWarehouses) Execute(_ context.Context) ([]invDomain.Warehouse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubs struct {
	listProducts    stubListProducts
	listStockLevels stubListStockLevels
	receiveOrder    stubReceivePOSSalesOrder
	listWarehouses  stubListWarehouses
}

func newTestRouter(s *stubs) *gin.Engine {
	h := poshandler.NewHandler(&s.listProducts, &s.listStockLevels, &s.receiveOrder, &s.listWarehouses)
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
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{Product: pmsDomain.Product{ID: "p1", Name: "Widget", Slug: "widget", Status: pmsDomain.ProductStatusActive}}},
		Total:    1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "Widget")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestListPOSProducts_includesSKUsFlattenedAcrossVariants(t *testing.T) {
	salePrice := int64(450)
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "p1", Name: "Widget", Slug: "widget", Status: pmsDomain.ProductStatusActive},
			SKUs: []apppms.POSProductSKU{
				{SKU: pmsDomain.SKU{ID: "sku-1", VariantID: "v1", SKUCode: "WID-S", Barcode: "111", PriceMinorUnits: 500, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}, VariantName: "Small"},
				{SKU: pmsDomain.SKU{ID: "sku-2", VariantID: "v2", SKUCode: "WID-L", Barcode: "222", PriceMinorUnits: 600, SalePriceMinorUnits: &salePrice, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}, VariantName: "Large"},
			},
		}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"WID-S"`)
	assert.Contains(t, w.Body.String(), `"barcode":"111"`)
	assert.Contains(t, w.Body.String(), `"variant_name":"Small"`)
	assert.Contains(t, w.Body.String(), `"sku_code":"WID-L"`)
	assert.Contains(t, w.Body.String(), `"sale_price_minor_units":450`)
}

func TestListPOSProducts_includesScaleCodeAndPricingModeForWeightEmbeddedSKU(t *testing.T) {
	pricePerKilo := int64(4999)
	scaleCode := "94721"
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "p1", Name: "Dry Meat", Slug: "dry-meat", Status: pmsDomain.ProductStatusActive},
			SKUs: []apppms.POSProductSKU{
				{SKU: pmsDomain.SKU{
					ID: "sku-1", VariantID: "v1", SKUCode: "DRYMEAT-1", PriceMinorUnits: 0, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
					PricingMode: pmsDomain.SKUPricingModePerWeight, PricePerKiloMinorUnits: &pricePerKilo, ScaleCode: &scaleCode,
				}, VariantName: "Default"},
			},
		}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"scale_code":"94721"`)
	assert.Contains(t, w.Body.String(), `"pricing_mode":"per_weight"`)
	assert.Contains(t, w.Body.String(), `"price_per_kilo_minor_units":4999`)
}

func TestListPOSProducts_includesBarcodeAndScaleCodeAliases(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "p1", Name: "Dry Meat", Slug: "dry-meat", Status: pmsDomain.ProductStatusActive},
			SKUs: []apppms.POSProductSKU{
				{
					SKU:              pmsDomain.SKU{ID: "sku-1", VariantID: "v1", SKUCode: "DRYMEAT-1", CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive},
					VariantName:      "Default",
					BarcodeAliases:   []apppms.POSBarcodeAlias{{Code: "9876543210987", UnitsPerScan: 1}},
					ScaleCodeAliases: []string{"94722"},
				},
			},
		}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"9876543210987"`)
	assert.Contains(t, w.Body.String(), `"94722"`)
}

func TestListPOSProducts_includesImageURLAndSaleEndDate(t *testing.T) {
	salePrice := int64(450)
	saleEndDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	imageURL := "https://example.com/widget.jpg"
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{
			Product:  pmsDomain.Product{ID: "p1", Name: "Widget", Slug: "widget", Status: pmsDomain.ProductStatusActive},
			ImageURL: &imageURL,
			SKUs: []apppms.POSProductSKU{
				{SKU: pmsDomain.SKU{ID: "sku-1", VariantID: "v1", SKUCode: "WID-S", PriceMinorUnits: 600, SalePriceMinorUnits: &salePrice, SaleEndDate: &saleEndDate, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}, VariantName: "Small"},
			},
		}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"image_url":"https://example.com/widget.jpg"`)
	assert.Contains(t, w.Body.String(), `"sale_end_date":"2026-09-01T00:00:00Z"`)
}

func TestListPOSProducts_omitsImageURLAndSaleEndDateWhenAbsent(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{out: &apppms.ListPOSProductCatalogResult{
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "p1", Name: "Widget", Slug: "widget", Status: pmsDomain.ProductStatusActive},
			SKUs:    []apppms.POSProductSKU{{SKU: pmsDomain.SKU{ID: "sku-1", VariantID: "v1", SKUCode: "WID-S", PriceMinorUnits: 600, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}, VariantName: "Small"}},
		}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/products", nil)

	require.Equal(t, 200, w.Code)
	assert.NotContains(t, w.Body.String(), `"image_url"`)
	assert.NotContains(t, w.Body.String(), `"sale_end_date"`)
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

// ── ListPOSWarehouses ────────────────────────────────────────────────────────

func TestListPOSWarehouses_success(t *testing.T) {
	s := &stubs{listWarehouses: stubListWarehouses{out: []invDomain.Warehouse{
		{ID: "wh-1", Name: "Main", Active: true},
		{ID: "wh-2", Name: "Overflow", Active: false},
	}}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/warehouses", nil)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"wh-1"`)
	assert.Contains(t, w.Body.String(), `"name":"Main"`)
	assert.Contains(t, w.Body.String(), `"active":true`)
	assert.Contains(t, w.Body.String(), `"active":false`)
}

func TestListPOSWarehouses_propagatesRepoError(t *testing.T) {
	s := &stubs{listWarehouses: stubListWarehouses{err: assert.AnError}}
	w := doRequest(newTestRouter(s), "GET", "/api/v1/pos/warehouses", nil)

	assert.Equal(t, 500, w.Code)
}

// ── CreatePOSSalesOrder ──────────────────────────────────────────────────────

func TestCreatePOSSalesOrder_success(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{out: &omsDomain.SalesOrder{
		ID: "so-1", ExternalOrderID: "pos-1", WarehouseID: "wh-1", CurrencyCode: "USD",
		TotalMinorUnits: 1000, OccurredAt: time.Now(), CreatedAt: time.Now(),
	}}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 1000, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: strPtr("sku-1"), Quantity: 2}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"external_order_id":"pos-1"`)
	require.Len(t, s.receiveOrder.gotInput.LineItems, 1)
	require.NotNil(t, s.receiveOrder.gotInput.LineItems[0].SKUID)
	assert.Equal(t, "sku-1", *s.receiveOrder.gotInput.LineItems[0].SKUID)
	assert.Equal(t, int64(2), s.receiveOrder.gotInput.LineItems[0].Quantity)
}

func TestCreatePOSSalesOrder_mapsEmptyLineItemsToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: omsDomain.ErrEmptyLineItems}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 500, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: strPtr("sku-1"), Quantity: 1}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code)
}

func TestCreatePOSSalesOrder_mapsInsufficientStockToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: invDomain.ErrInsufficientStock}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 50000, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: strPtr("sku-1"), Quantity: 100}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code)
}

// TestCreatePOSSalesOrder_acceptsALineItemWithNoSkuId proves an adhoc
// line item (no sku_id) round-trips through the HTTP layer -- the request
// binds fine and the use case receives a nil SKUID, not an empty string.
func TestCreatePOSSalesOrder_acceptsALineItemWithNoSkuId(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{out: &omsDomain.SalesOrder{
		ID: "so-1", ExternalOrderID: "pos-1", WarehouseID: "wh-1", CurrencyCode: "USD",
		TotalMinorUnits: 500, OccurredAt: time.Now(), CreatedAt: time.Now(),
	}}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 500, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{Name: strPtr("Gift wrapping"), Quantity: 1}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	require.Equal(t, 200, w.Code)
	require.Len(t, s.receiveOrder.gotInput.LineItems, 1)
	assert.Nil(t, s.receiveOrder.gotInput.LineItems[0].SKUID)
	assert.Equal(t, "Gift wrapping", s.receiveOrder.gotInput.LineItems[0].Name)
}

func TestCreatePOSSalesOrder_mapsAdhocLineItemRequiresNameToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: omsDomain.ErrAdhocLineItemRequiresName}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 500, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{Quantity: 1}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code)
}

func TestCreatePOSSalesOrder_mapsSKUNotFoundToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: pmsDomain.ErrSKUNotFound}}
	body := genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "USD", TotalMinorUnits: 500, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: strPtr("missing-sku"), Quantity: 1}},
	}
	w := doRequest(newTestRouter(s), "POST", "/api/v1/pos/sales-orders", body)

	assert.Equal(t, 422, w.Code, "a genuinely wrong sku_id should be a clear validation error, not a generic 500")
}
