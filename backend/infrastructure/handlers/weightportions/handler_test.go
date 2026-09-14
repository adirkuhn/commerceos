package weightportions_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genweightportions "github.com/adirkuhn/commerceos/api/weightportions"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	weightportionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/weightportions"
)

func init() { gin.SetMode(gin.TestMode) }

type stubBulkAdd struct {
	out *apppms.BulkAddWeightPortionsResult
	err error
	got apppms.BulkAddWeightPortionsInput
}

func (s *stubBulkAdd) Execute(_ context.Context, in apppms.BulkAddWeightPortionsInput) (*apppms.BulkAddWeightPortionsResult, error) {
	s.got = in
	return s.out, s.err
}

func newTestRouter(h *weightportionshandler.Handler) *gin.Engine {
	r := gin.New()
	genweightportions.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestBulkAddWeightPortionsHandler_success(t *testing.T) {
	weight := int64(404)
	bulkAdd := &stubBulkAdd{out: &apppms.BulkAddWeightPortionsResult{Created: []pmsDomain.SKU{
		{ID: "sku-1", SKUCode: "CARNE-SECA-404G", Barcode: "2994721004047", PriceMinorUnits: 12116, CurrencyCode: "EUR", WeightGrams: &weight},
	}}}
	h := weightportionshandler.NewHandler(bulkAdd)
	body := `{"warehouse_id":"warehouse-1","weights_grams":[404]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/pms/skus/sku-bulk-1/bulk-weight-portions", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sku-bulk-1", bulkAdd.got.BulkSKUID)
	assert.Equal(t, "warehouse-1", bulkAdd.got.WarehouseID)
	assert.Equal(t, []int64{404}, bulkAdd.got.WeightsGrams)
	assert.Contains(t, w.Body.String(), "CARNE-SECA-404G")
}

func TestBulkAddWeightPortionsHandler_passesOptionalFields(t *testing.T) {
	bulkAdd := &stubBulkAdd{out: &apppms.BulkAddWeightPortionsResult{}}
	h := weightportionshandler.NewHandler(bulkAdd)
	body := `{"warehouse_id":"warehouse-1","bulk_batch_id":"batch-1","sku_code_prefix":"DRIED-BEEF","reason":"weekly restock","weights_grams":[404,458]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/pms/skus/sku-bulk-1/bulk-weight-portions", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, bulkAdd.got.BulkBatchID)
	assert.Equal(t, "batch-1", *bulkAdd.got.BulkBatchID)
	assert.Equal(t, "DRIED-BEEF", bulkAdd.got.SKUCodePrefix)
	assert.Equal(t, "weekly restock", bulkAdd.got.Reason)
	assert.Equal(t, []int64{404, 458}, bulkAdd.got.WeightsGrams)
}

func TestBulkAddWeightPortionsHandler_invalidInput(t *testing.T) {
	bulkAdd := &stubBulkAdd{err: pmsDomain.ErrNoWeightsProvided}
	h := weightportionshandler.NewHandler(bulkAdd)
	body := `{"warehouse_id":"warehouse-1","weights_grams":[404]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/pms/skus/sku-bulk-1/bulk-weight-portions", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestBulkAddWeightPortionsHandler_bulkSKUNotFound(t *testing.T) {
	bulkAdd := &stubBulkAdd{err: pmsDomain.ErrSKUNotFound}
	h := weightportionshandler.NewHandler(bulkAdd)
	body := `{"warehouse_id":"warehouse-1","weights_grams":[404]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/pms/skus/missing/bulk-weight-portions", body)

	require.Equal(t, http.StatusNotFound, w.Code)
}
