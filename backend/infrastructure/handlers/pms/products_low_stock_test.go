package pms_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubSetProductLowStockThreshold struct {
	called bool
	gotID  string
	got    *int
	err    error
}

func (s *stubSetProductLowStockThreshold) Execute(_ context.Context, id string, threshold *int) (*pmsDomain.Product, error) {
	s.called, s.gotID, s.got = true, id, threshold
	if s.err != nil {
		return nil, s.err
	}
	return &pmsDomain.Product{ID: id, Name: "Queijo minas", LowStockThreshold: threshold}, nil
}

func TestListProducts_passesTheStockFilter(t *testing.T) {
	for query, want := range map[string]pmsDomain.StockLevel{
		"?stock=low": pmsDomain.StockLevelLow,
		"?stock=out": pmsDomain.StockLevelOut,
		"":           pmsDomain.StockLevelAny,
	} {
		s := &stubs{}
		w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products"+query, nil)

		require.Equal(t, http.StatusOK, w.Code, query)
		assert.Equal(t, want, s.listProducts.gotInput.StockLevel, query)
	}
}

func TestListProducts_unknownStockFilterIs422(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products?stock=high", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListProducts_includesTheProductsOwnLowStockThreshold(t *testing.T) {
	seven := 7
	s := &stubs{listProducts: stubListProducts{out: []apppms.POSProductWithSKUs{
		{Product: pmsDomain.Product{ID: "p1", Name: "Own line", LowStockThreshold: &seven}},
		{Product: pmsDomain.Product{ID: "p2", Name: "Tenant line"}},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"low_stock_threshold":7`)
	assert.Contains(t, w.Body.String(), `"low_stock_threshold":null`, "no override is an explicit null")
}

func TestSetProductLowStockThreshold_setsAndReturnsTheProduct(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/low-stock-threshold", map[string]any{"low_stock_threshold": 1000})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", s.setLowStockThreshold.gotID)
	require.NotNil(t, s.setLowStockThreshold.got)
	assert.Equal(t, 1000, *s.setLowStockThreshold.got)
	assert.Contains(t, w.Body.String(), `"low_stock_threshold":1000`)
}

func TestSetProductLowStockThreshold_nullClearsIt(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/low-stock-threshold", map[string]any{"low_stock_threshold": nil})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.True(t, s.setLowStockThreshold.called)
	assert.Nil(t, s.setLowStockThreshold.got)
}

func TestSetProductLowStockThreshold_errors(t *testing.T) {
	s := &stubs{setLowStockThreshold: stubSetProductLowStockThreshold{err: pmsDomain.ErrInvalidLowStockThreshold}}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/low-stock-threshold", map[string]any{"low_stock_threshold": -1})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	s = &stubs{setLowStockThreshold: stubSetProductLowStockThreshold{err: pmsDomain.ErrProductNotFound}}
	w = doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/low-stock-threshold", map[string]any{"low_stock_threshold": 3})
	assert.Equal(t, http.StatusNotFound, w.Code)

	s = &stubs{}
	w = doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/products/p1/low-stock-threshold", "not-an-object")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.False(t, s.setLowStockThreshold.called)
}
