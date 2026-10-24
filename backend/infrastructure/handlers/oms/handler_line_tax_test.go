package oms_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestGetSalesOrder_linesCarryTheirTotalsWithAndWithoutVATAndRefunds(t *testing.T) {
	get := &stubGetSalesOrder{out: &appoms.GetSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{ID: "so-1"},
		LineItems: []omsDomain.SalesOrderLineItem{{ID: "li-1", Quantity: 3, TotalMinorUnits: 1373, TotalTaxMinorUnits: 124,
			RefundedQuantity: 3, RefundedAmountMinorUnits: 1497}},
	}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/so-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data struct {
			LineItems []map[string]any `json:"line_items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.LineItems, 1)
	line := body.Data.LineItems[0]
	assert.Equal(t, false, line["total_includes_tax"])
	assert.EqualValues(t, 1373, line["total_excl_tax_minor_units"])
	assert.EqualValues(t, 1497, line["total_incl_tax_minor_units"])
	assert.EqualValues(t, 3, line["refunded_quantity"])
	assert.EqualValues(t, 1497, line["refunded_amount_minor_units"])
}

func TestCreateSalesOrder_passesWhetherLineTotalsIncludeTax(t *testing.T) {
	create := &stubCreateSalesOrder{}
	body := map[string]any{
		"warehouse_id": "wh-1", "currency_code": "EUR", "occurred_at": "2026-01-01T00:00:00Z",
		"line_items": []map[string]any{{"sku_id": "sku-1", "quantity": 2, "price_minor_units": 199,
			"total_minor_units": 398, "total_tax_minor_units": 33, "total_includes_tax": true}},
	}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, &stubGetSalesOrder{}, create, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "POST", "/api/v1/sales-orders", body)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.True(t, create.gotInput.LineItems[0].TotalIncludesTax)
}
