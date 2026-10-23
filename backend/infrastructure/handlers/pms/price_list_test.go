package pms_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubListPriceList struct {
	out *apppms.ListPriceListResult
	err error
	got apppms.ListPriceListInput
}

func (s *stubListPriceList) Execute(_ context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error) {
	s.got = in
	return s.out, s.err
}

func TestListPriceList_mapsRowsAndMeta(t *testing.T) {
	sale, cost := int64(899), int64(500)
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	s := &stubs{listPriceList: stubListPriceList{out: &apppms.ListPriceListResult{
		Total: 3, PricesIncludeTax: true,
		Rows: []apppms.PriceListRow{{
			ProductID: "p-1", ProductName: "Feijão", VariantName: "1kg", SKUID: "sku-1", SKUCode: "FEI-1",
			UnitType: pmsDomain.SKUUnitTypeUnit, CurrencyCode: "EUR", PriceMinorUnits: 999, SalePriceMinorUnits: &sale,
			CostPriceMinorUnits: &cost, AvailableQuantity: 12, EarliestExpiry: &expiry,
			TaxRateBasisPoints: 900, TargetMarkupBasisPoints: 4000,
		}},
	}}}

	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/price-list", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
		"data": [{
			"product_id": "p-1", "product_name": "Feijão", "variant_name": "1kg", "sku_id": "sku-1", "sku_code": "FEI-1",
			"unit_type": "unit", "currency_code": "EUR", "price_minor_units": 999, "sale_price_minor_units": 899,
			"cost_price_minor_units": 500, "available_quantity": 12, "earliest_expiry": "2026-12-01",
			"tax_rate_basis_points": 900, "target_markup_basis_points": 4000
		}],
		"meta": {"total": 3, "prices_include_tax": true}
	}`, w.Body.String())
}

func TestListPriceList_passesFiltersThrough(t *testing.T) {
	s := &stubs{listPriceList: stubListPriceList{out: &apppms.ListPriceListResult{}}}
	p1, p2 := "0b0e7e8e-0000-4000-8000-000000000001", "0b0e7e8e-0000-4000-8000-000000000002"

	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/price-list?page=2&per_page=25&search=fei&product_ids="+p1+","+p2, nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, apppms.ListPriceListInput{Page: 2, PerPage: 25, Search: "fei", ProductIDs: []string{p1, p2}}, s.listPriceList.got)
	assert.Contains(t, w.Body.String(), `"data":[]`, "an empty page is an empty list, not null")
}

func TestListPriceList_rejectsMalformedProductIDs(t *testing.T) {
	s := &stubs{listPriceList: stubListPriceList{out: &apppms.ListPriceListResult{}}}

	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/price-list?product_ids=not-a-uuid", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListPriceList_useCaseErrorIsServerError(t *testing.T) {
	s := &stubs{listPriceList: stubListPriceList{err: errors.New("boom")}}

	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/price-list", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "boom", "internal errors don't leak")
}

func TestSetAutomaticCostPrice_success(t *testing.T) {
	cost := int64(450)
	s := &stubs{setAutomaticCostPrice: stubSetManualCostPrice{out: &pmsDomain.SKU{
		ID: "sku-1", CurrencyCode: "EUR", CostPriceMinorUnits: &cost, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed, Status: pmsDomain.SKUStatusActive,
	}}}

	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/sku-1/cost-price/automatic", map[string]int64{"cost_price_minor_units": 450})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sku-1", s.setAutomaticCostPrice.gotSKUID)
	assert.Equal(t, int64(450), s.setAutomaticCostPrice.gotCostPriceMinorUnits)
	assert.Contains(t, w.Body.String(), `"cost_price_source":"computed"`)
}

func TestSetAutomaticCostPrice_notFound(t *testing.T) {
	s := &stubs{setAutomaticCostPrice: stubSetManualCostPrice{err: pmsDomain.ErrSKUNotFound}}

	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/missing/cost-price/automatic", map[string]int64{"cost_price_minor_units": 450})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetAutomaticCostPrice_negativeIsValidationError(t *testing.T) {
	s := &stubs{setAutomaticCostPrice: stubSetManualCostPrice{err: pmsDomain.ErrInvalidCostPrice}}

	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/sku-1/cost-price/automatic", map[string]int64{"cost_price_minor_units": -1})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "COST_PRICE_NEGATIVE")
}

func TestSetAutomaticCostPrice_missingBodyIsValidationError(t *testing.T) {
	w := doRequest(newTestRouter(&stubs{}), http.MethodPut, "/api/v1/pms/skus/sku-1/cost-price/automatic", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
