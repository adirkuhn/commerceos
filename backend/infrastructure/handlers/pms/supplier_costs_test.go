package pms_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type stubListSKUSupplierCosts struct {
	out   appprocurement.SKUSupplierCosts
	err   error
	gotIn appprocurement.ListSKUSupplierCostsInput
}

func (s *stubListSKUSupplierCosts) Execute(_ context.Context, in appprocurement.ListSKUSupplierCostsInput) (appprocurement.SKUSupplierCosts, error) {
	s.gotIn = in
	return s.out, s.err
}

// assertMatchesPMSContract validates the response against api/pms/openapi.yaml.
func assertMatchesPMSContract(t *testing.T, req *http.Request, w *httptest.ResponseRecorder) {
	t.Helper()
	spec, err := genpms.GetSpec()
	require.NoError(t, err)
	spec.Servers = nil
	router, err := gorillamux.NewRouter(spec)
	require.NoError(t, err)
	probe := req.Clone(context.Background())
	probe.URL.Path = req.URL.Path[len("/api/v1"):]
	route, pathParams, err := router.FindRoute(probe)
	require.NoError(t, err)
	in := &openapi3filter.RequestValidationInput{Request: probe, PathParams: pathParams, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
	require.NoError(t, openapi3filter.ValidateRequest(context.Background(), in), "request breaks the contract")
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in, Status: w.Code, Header: w.Header(),
		Body:    io.NopCloser(bytes.NewReader(w.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	require.NoError(t, openapi3filter.ValidateResponse(context.Background(), out), "response breaks the contract: %d %s", w.Code, w.Body.String())
}

func getSupplierCosts(t *testing.T, s *stubs, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	newTestRouter(s).ServeHTTP(w, req)
	assertMatchesPMSContract(t, req, w)
	return w
}

func strPtr(v string) *string { return &v }

func int64Ptr(v int64) *int64 { return &v }

func TestListSKUSupplierCosts(t *testing.T) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	avg, next := int64(600), 1
	acme := strPtr("acme")
	s := &stubs{listSKUSupplierCosts: stubListSKUSupplierCosts{out: appprocurement.SKUSupplierCosts{
		Currency: "EUR",
		Summary: procurementDomain.SupplierCostSummary{
			Last:              &procurementDomain.LastSupplierCost{LandedUnitCostMinorUnits: 700, PurchaseOrderID: "po-2", PurchaseOrderLineItemID: "l2", SupplierID: acme, ReceivedAt: at},
			AverageMinorUnits: &avg,
			BySupplier: []procurementDomain.SupplierCostStat{
				{SupplierID: acme, LastLandedUnitCostMinorUnits: int64Ptr(700), LastReceivedAt: at, AverageMinorUnits: 600, Quantity: 20, Receipts: 2},
				{SupplierID: nil, LastReceivedAt: at, AverageMinorUnits: 0, Quantity: 5, Receipts: 1},
			},
			ExcludedOtherCurrency: 1,
		},
		Page: procurementDomain.SupplierCostPage{Items: []procurementDomain.SupplierCostEntry{{
			Kind: procurementDomain.SupplierCostKindReceipt, Quantity: 10, UnitCostMinorUnits: 650, LandedUnitCostMinorUnits: 700,
			CurrencyCode: "EUR", ReceivedAt: at, PurchaseOrderID: "po-2", PurchaseOrderLineItemID: "l2", SupplierID: acme,
		}}, NextCursor: &next, Total: 3},
		SupplierNames: map[string]string{"acme": "Acme Foods"},
	}}}

	w := getSupplierCosts(t, s, "/api/v1/pms/skus/sku-1/supplier-costs?supplier_id=acme&cursor=0&limit=1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, appprocurement.ListSKUSupplierCostsInput{SKUID: "sku-1", SupplierID: "acme", Cursor: 0, Limit: 1}, s.listSKUSupplierCosts.gotIn)
	var body struct {
		Data genpms.SKUSupplierCosts `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	d := body.Data
	assert.Equal(t, "EUR", d.CurrencyCode)
	require.NotNil(t, d.Last)
	assert.Equal(t, int64(700), d.Last.LandedUnitCostMinorUnits)
	assert.Equal(t, "Acme Foods", *d.Last.SupplierName)
	assert.Equal(t, int64(600), *d.AverageCostMinorUnits)
	require.Len(t, d.BySupplier, 2)
	assert.Equal(t, "Acme Foods", *d.BySupplier[0].SupplierName)
	assert.Nil(t, d.BySupplier[1].SupplierId, "receipts without a supplier")
	assert.Nil(t, d.BySupplier[1].LastLandedUnitCostMinorUnits)
	assert.Equal(t, 1, d.ExcludedOtherCurrency)
	require.Len(t, d.Items, 1)
	assert.Equal(t, genpms.SupplierCostKind("receipt"), d.Items[0].Kind)
	assert.Equal(t, int64(650), d.Items[0].UnitCostMinorUnits)
	assert.Equal(t, 1, *d.NextCursor)
	assert.Equal(t, 3, d.Total)
}

func TestListSKUSupplierCosts_emptyHistory(t *testing.T) {
	s := &stubs{listSKUSupplierCosts: stubListSKUSupplierCosts{out: appprocurement.SKUSupplierCosts{Currency: "EUR"}}}

	w := getSupplierCosts(t, s, "/api/v1/pms/skus/sku-1/supplier-costs")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 50, s.listSKUSupplierCosts.gotIn.Limit, "default page size")
	assert.JSONEq(t, `{"data":{"currency_code":"EUR","last":null,"average_cost_minor_units":null,"by_supplier":[],
		"excluded_other_currency":0,"items":[],"next_cursor":null,"total":0}}`, w.Body.String())
}

func TestListSKUSupplierCosts_unexpectedErrorIs500(t *testing.T) {
	s := &stubs{listSKUSupplierCosts: stubListSKUSupplierCosts{err: errors.New("db down")}}

	w := getSupplierCosts(t, s, "/api/v1/pms/skus/sku-1/supplier-costs")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
