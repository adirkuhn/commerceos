package weightportions_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genweightportions "github.com/adirkuhn/commerceos/api/weightportions"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	weightportionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/weightportions"
)

func init() { gin.SetMode(gin.TestMode) }

type stubPack struct {
	out    *apppms.PackWeightPortionsResult
	err    error
	got    apppms.PackWeightPortionsInput
	called bool
}

func (s *stubPack) Execute(_ context.Context, in apppms.PackWeightPortionsInput) (*apppms.PackWeightPortionsResult, error) {
	s.got, s.called = in, true
	return s.out, s.err
}

// newTestRouter mounts the handler behind the same OpenAPI request
// validator cmd/api/main.go uses, so the contract itself is under test.
type stubLink struct {
	out    *apppms.LinkExistingWeightPackagesResult
	err    error
	got    apppms.LinkExistingWeightPackagesInput
	called bool
}

func (s *stubLink) Execute(_ context.Context, in apppms.LinkExistingWeightPackagesInput) (*apppms.LinkExistingWeightPackagesResult, error) {
	s.got, s.called = in, true
	return s.out, s.err
}

func newTestRouter(t *testing.T, h *weightportionshandler.Handler) *gin.Engine {
	t.Helper()
	spec, err := genweightportions.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genweightportions.RegisterHandlers(g, h)
	return r
}

func doRequest(rg *gin.Engine, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

const packPath = "/api/v1/pms/skus/sku-bulk-1/weight-packages"

func TestPackWeightPortionsHandler_success(t *testing.T) {
	weight := int64(380)
	pack := &stubPack{out: &apppms.PackWeightPortionsResult{Lines: []apppms.PackedWeightLine{{
		SKU:         pmsDomain.SKU{ID: "sku-1", SKUCode: "CARNE-SECA-380G", Barcode: "2994721003804", PriceMinorUnits: 11396, CurrencyCode: "EUR", WeightGrams: &weight},
		WeightGrams: 380, Quantity: 10, Created: true,
	}}}}
	body := `{"warehouse_id":"warehouse-1","lines":[{"weight_grams":380,"quantity":10}]}`

	w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(pack, &stubLink{})), packPath, body)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "sku-bulk-1", pack.got.BulkSKUID)
	assert.Equal(t, "warehouse-1", pack.got.WarehouseID)
	assert.Equal(t, []apppms.WeightPackageLine{{WeightGrams: 380, Quantity: 10}}, pack.got.Lines)
	assert.JSONEq(t, `{"data":{"lines":[{"sku_id":"sku-1","sku_code":"CARNE-SECA-380G","barcode":"2994721003804",
		"price_minor_units":11396,"currency_code":"EUR","weight_grams":380,"quantity":10,"created":true}]},"error":null}`, w.Body.String())
}

func TestPackWeightPortionsHandler_passesOptionalFields(t *testing.T) {
	pack := &stubPack{out: &apppms.PackWeightPortionsResult{}}
	body := `{"warehouse_id":"warehouse-1","bulk_batch_id":"batch-1","sku_code_prefix":"DRIED-BEEF","reason":"weekly restock",
		"lines":[{"weight_grams":430,"quantity":1},{"weight_grams":540,"quantity":4}]}`

	w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(pack, &stubLink{})), packPath, body)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, pack.got.BulkBatchID)
	assert.Equal(t, "batch-1", *pack.got.BulkBatchID)
	assert.Equal(t, "DRIED-BEEF", pack.got.SKUCodePrefix)
	assert.Equal(t, "weekly restock", pack.got.Reason)
	assert.Len(t, pack.got.Lines, 2)
	assert.JSONEq(t, `{"data":{"lines":[]},"error":null}`, w.Body.String())
}

func TestPackWeightPortionsHandler_contractRejectsBadBodies(t *testing.T) {
	for name, body := range map[string]string{
		"no lines":         `{"warehouse_id":"warehouse-1","lines":[]}`,
		"missing lines":    `{"warehouse_id":"warehouse-1"}`,
		"zero quantity":    `{"warehouse_id":"warehouse-1","lines":[{"weight_grams":380,"quantity":0}]}`,
		"zero weight":      `{"warehouse_id":"warehouse-1","lines":[{"weight_grams":0,"quantity":1}]}`,
		"missing quantity": `{"warehouse_id":"warehouse-1","lines":[{"weight_grams":380}]}`,
		"no warehouse":     `{"lines":[{"weight_grams":380,"quantity":1}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			pack := &stubPack{out: &apppms.PackWeightPortionsResult{}}
			w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(pack, &stubLink{})), packPath, body)
			assert.GreaterOrEqual(t, w.Code, 400)
			assert.Less(t, w.Code, 500)
			assert.False(t, pack.called, "use case must not run for a body the contract rejects")
		})
	}
}

func TestPackWeightPortionsHandler_mapsDomainErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		code int
	}{
		"sku not found":      {pmsDomain.ErrSKUNotFound, http.StatusNotFound},
		"batch not found":    {invDomain.ErrBatchNotFound, http.StatusNotFound},
		"duplicate sku code": {pmsDomain.ErrDuplicateSKUCode, http.StatusUnprocessableEntity},
		"not a weight pool":  {pmsDomain.ErrBulkSKUNotWeightPool, http.StatusUnprocessableEntity},
		"not enough in lot":  {invDomain.ErrInsufficientStock, http.StatusUnprocessableEntity},
		"lot of another sku": {invDomain.ErrBatchDoesNotBelongToSKU, http.StatusUnprocessableEntity},
		"unexpected failure": {errors.New("db down"), http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			pack := &stubPack{err: tc.err}
			body := `{"warehouse_id":"warehouse-1","lines":[{"weight_grams":380,"quantity":1}]}`
			w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(pack, &stubLink{})), packPath, body)
			assert.Equal(t, tc.code, w.Code)
		})
	}
}

const linkPath = "/api/v1/pms/skus/sku-bulk-1/weight-packages/link-existing"

func TestLinkExistingWeightPackagesHandler_previewAndLink(t *testing.T) {
	link := &stubLink{out: &apppms.LinkExistingWeightPackagesResult{
		Linked: []apppms.LinkedWeightPackage{{
			SKU: pmsDomain.SKU{ID: "sku-528", SKUCode: "2994721005285"}, VariantName: "~ 528g", WeightGrams: 528, PriceMinorUnits: 15835, PriceChanged: true,
		}},
		Skipped: []apppms.SkippedWeightPackage{{SKU: pmsDomain.SKU{ID: "sku-x", SKUCode: "7897122604154"}, VariantName: "Gift box", Reason: apppms.LinkSkipNotAScaleLabel}},
	}}

	w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(&stubPack{}, link)), linkPath, `{"dry_run":true}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, apppms.LinkExistingWeightPackagesInput{BulkSKUID: "sku-bulk-1", DryRun: true}, link.got)
	assert.JSONEq(t, `{"data":{
		"linked":[{"sku_id":"sku-528","sku_code":"2994721005285","variant_name":"~ 528g","weight_grams":528,"price_minor_units":15835,"price_changed":true}],
		"skipped":[{"sku_id":"sku-x","sku_code":"7897122604154","variant_name":"Gift box","reason":"not_a_scale_label"}],
		"errors":[]},"error":null}`, w.Body.String())
}

func TestLinkExistingWeightPackagesHandler_dryRunDefaultsToFalse(t *testing.T) {
	link := &stubLink{out: &apppms.LinkExistingWeightPackagesResult{}}

	w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(&stubPack{}, link)), linkPath, `{}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, link.got.DryRun)
	assert.JSONEq(t, `{"data":{"linked":[],"skipped":[],"errors":[]},"error":null}`, w.Body.String())
}

func TestLinkExistingWeightPackagesHandler_mapsErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		code int
	}{
		"pool not found":        {pmsDomain.ErrSKUNotFound, http.StatusNotFound},
		"not a scale-sold pool": {pmsDomain.ErrBulkSKUNotWeightPool, http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			w := doRequest(newTestRouter(t, weightportionshandler.NewHandler(&stubPack{}, &stubLink{err: tc.err})), linkPath, `{}`)
			assert.Equal(t, tc.code, w.Code)
		})
	}
}
