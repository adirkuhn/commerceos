package oms_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// The Sales Orders page filters storefront orders by source=internal --
// pins that the contract accepts it and the handler passes it on.
func TestListSalesOrders_filtersByStorefrontSource(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?source=internal&source=manual", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourceInternal, omsDomain.SalesOrderSourceManual}, list.gotInput.Source)
}
