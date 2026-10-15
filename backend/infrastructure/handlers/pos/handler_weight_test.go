package pos_test

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpos "github.com/adirkuhn/commerceos/api/pos"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	poshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pos"
)

const salesOrdersPath = "/api/v1/pos/sales-orders"

// newContractTestRouter mounts the handler behind the same OpenAPI request
// validator cmd/api/main.go uses, so the contract itself is under test.
func newContractTestRouter(t *testing.T, s *stubs) *gin.Engine {
	t.Helper()
	spec, err := genpos.GetSpec()
	require.NoError(t, err)
	h := poshandler.NewHandler(&s.listProducts, &s.listStockLevels, &s.receiveOrder, &s.listWarehouses)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genpos.RegisterHandlers(g, h)
	return r
}

func receivedOrderStubs() *stubs {
	return &stubs{receiveOrder: stubReceivePOSSalesOrder{out: &omsDomain.SalesOrder{
		ID: "so-1", ExternalOrderID: "pos-1", WarehouseID: "wh-1", CurrencyCode: "EUR",
		TotalMinorUnits: 1225, OccurredAt: time.Now(), CreatedAt: time.Now(),
	}}}
}

func weighedOrderBody(weightGrams int64) genpos.CreatePOSSalesOrderRequest {
	return genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "EUR", TotalMinorUnits: 1225, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{SkuId: strPtr("sku-1"), Quantity: 1, WeightGrams: &weightGrams}},
	}
}

func TestCreatePOSSalesOrder_passesTheLineWeightToTheUseCase(t *testing.T) {
	s := receivedOrderStubs()

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, weighedOrderBody(350))

	require.Equal(t, 200, w.Code, w.Body.String())
	require.NotNil(t, s.receiveOrder.gotInput.LineItems[0].WeightGrams)
	assert.Equal(t, int64(350), *s.receiveOrder.gotInput.LineItems[0].WeightGrams)
}

func TestCreatePOSSalesOrder_contractRejectsAZeroWeight(t *testing.T) {
	w := doRequest(newContractTestRouter(t, receivedOrderStubs()), "POST", salesOrdersPath, weighedOrderBody(0))

	assert.Equal(t, 400, w.Code)
}

func TestCreatePOSSalesOrder_mapsWeighedLineQuantityErrorToValidationError(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: omsDomain.ErrWeighedLineItemQuantityMustBeOne}}

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, weighedOrderBody(350))

	assert.Equal(t, 422, w.Code)
}
