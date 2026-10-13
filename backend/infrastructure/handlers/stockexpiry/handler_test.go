package stockexpiry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	genstockexpiry "github.com/adirkuhn/commerceos/api/stockexpiry"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/stockexpiry"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const batchID = "6f1c2a7e-6c0b-4f0e-9a51-3d7f1e2b9c10"

type stubList struct {
	got appinventory.ListCloseToExpiryInput
	out *appinventory.ListCloseToExpiryResult
	err error
}

func (s *stubList) Execute(_ context.Context, in appinventory.ListCloseToExpiryInput) (*appinventory.ListCloseToExpiryResult, error) {
	s.got = in
	return s.out, s.err
}

type stubAck struct {
	got    appinventory.BatchExpiryAcknowledgementInput
	called bool
	err    error
}

func (s *stubAck) Execute(_ context.Context, in appinventory.BatchExpiryAcknowledgementInput) error {
	s.got, s.called = in, true
	return s.err
}

func newRouter(t *testing.T, list *stubList, ack, unack *stubAck) *gin.Engine {
	t.Helper()
	spec, err := genstockexpiry.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte("s")))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genstockexpiry.RegisterHandlers(g, handler.NewHandler(list, ack, unack))
	return r
}

func token(t *testing.T) string {
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "user-1", TenantID: "t", Role: middleware.RoleCustomerUser,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("s"))
	require.NoError(t, err)
	return s
}

func do(t *testing.T, r *gin.Engine, method, path, body string, withToken bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if withToken {
		req.Header.Set("Authorization", "Bearer "+token(t))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func date(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func TestListCloseToExpiry_MapsTheWindowAndItems(t *testing.T) {
	sale := int64(399)
	ackAt := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	window := appinventory.CloseToExpiryWindow{Band: settingsDomain.ExpiryBand{WindowDays: 30, MinDaysLeft: 16},
		From: date("2026-10-28"), To: date("2026-11-11"), Counts: invDomain.CloseToExpiryCounts{Open: 1, Acknowledged: 4}}
	list := &stubList{out: &appinventory.ListCloseToExpiryResult{
		Today: date("2026-10-12"), Window: window, Windows: []appinventory.CloseToExpiryWindow{window},
		Page: 2, PerPage: 10, Total: 4,
		Items: []invDomain.CloseToExpiryBatch{{
			BatchID: "b-1", LotNumber: "L1", ExpiryDate: date("2026-10-30"), Quantity: 7,
			WarehouseID: "wh-1", WarehouseName: "Amsterdam", SKUID: "sku-1", SKUCode: "PQ-1",
			ProductID: "p-1", ProductName: "Pão de queijo", ImageURL: "https://cdn/pq.jpg",
			PriceMinorUnits: 499, SalePriceMinorUnits: &sale, CurrencyCode: "EUR", AcknowledgedAt: &ackAt,
			Clearance: &invDomain.ClearancePromotionRef{PromotionID: "promo-1", PromotionName: "Clearance"},
		}},
	}}
	r := newRouter(t, list, &stubAck{}, &stubAck{})

	w := do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry?window=30&acknowledged=true&page=2&per_page=10", "", true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, appinventory.ListCloseToExpiryInput{WindowDays: 30, Acknowledged: true, Page: 2, PerPage: 10}, list.got)
	var resp genstockexpiry.CloseToExpiryListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2026-10-12", resp.Data.Today.String())
	assert.Equal(t, genstockexpiry.CloseToExpiryWindow{WindowDays: 30, MinDaysLeft: 16, OpenCount: 1, AcknowledgedCount: 4,
		From: resp.Data.Window.From, To: resp.Data.Window.To}, resp.Data.Window)
	assert.Equal(t, "2026-10-28", resp.Data.Window.From.String())
	assert.Equal(t, "2026-11-11", resp.Data.Window.To.String())
	assert.Len(t, resp.Data.Windows, 1)
	assert.Equal(t, 4, resp.Data.Total)
	require.Len(t, resp.Data.Items, 1)
	item := resp.Data.Items[0]
	assert.Equal(t, 18, item.DaysLeft)
	assert.Equal(t, "Pão de queijo", item.ProductName)
	assert.Equal(t, "Amsterdam", item.WarehouseName)
	require.NotNil(t, item.SalePriceMinorUnits)
	assert.Equal(t, 399, *item.SalePriceMinorUnits)
	require.NotNil(t, item.ClearancePromotion)
	assert.Equal(t, "promo-1", item.ClearancePromotion.Id)
	require.NotNil(t, item.AcknowledgedAt)
}

func TestListCloseToExpiry_IncludesCostTaxClassAndWeightPool(t *testing.T) {
	cost, taxClass := int64(210), "tc-food"
	list := &stubList{out: &appinventory.ListCloseToExpiryResult{Today: date("2026-10-12"),
		Items: []invDomain.CloseToExpiryBatch{
			{BatchID: "b-1", ExpiryDate: date("2026-10-30"), UnitCostMinorUnits: &cost, UnitCostSource: invDomain.UnitCostSourceBatch, TaxClassID: &taxClass},
			{BatchID: "b-2", ExpiryDate: date("2026-10-30"), WeightPool: true},
		}}}
	r := newRouter(t, list, &stubAck{}, &stubAck{})

	w := do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry", "", true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp genstockexpiry.CloseToExpiryListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	withCost, pool := resp.Data.Items[0], resp.Data.Items[1]
	require.NotNil(t, withCost.UnitCostMinorUnits)
	assert.Equal(t, 210, *withCost.UnitCostMinorUnits)
	require.NotNil(t, withCost.UnitCostSource)
	assert.Equal(t, genstockexpiry.Batch, *withCost.UnitCostSource)
	require.NotNil(t, withCost.TaxClassId)
	assert.Equal(t, "tc-food", *withCost.TaxClassId)
	assert.False(t, withCost.WeightPool)
	assert.Nil(t, pool.UnitCostMinorUnits)
	assert.Nil(t, pool.UnitCostSource, "no cost known: no source")
	assert.True(t, pool.WeightPool)
	assert.Contains(t, w.Body.String(), `"unit_cost_minor_units":null`)
}

func TestListCloseToExpiry_HidesAnEndedSale(t *testing.T) {
	sale := int64(399)
	ended := date("2026-10-01")
	list := &stubList{out: &appinventory.ListCloseToExpiryResult{Today: date("2026-10-12"),
		Items: []invDomain.CloseToExpiryBatch{{BatchID: "b-1", ExpiryDate: date("2026-10-30"), SalePriceMinorUnits: &sale, SaleEndDate: &ended}}}}

	w := do(t, newRouter(t, list, &stubAck{}, &stubAck{}), http.MethodGet, "/api/v1/inventory/batches/close-to-expiry", "", true)

	require.Equal(t, http.StatusOK, w.Code)
	var resp genstockexpiry.CloseToExpiryListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Data.Items[0].SalePriceMinorUnits)
	assert.Equal(t, appinventory.ListCloseToExpiryInput{Page: 1, PerPage: 50}, list.got, "contract defaults")
}

func TestListCloseToExpiry_Errors(t *testing.T) {
	r := newRouter(t, &stubList{err: invDomain.ErrUnknownExpiryWindow}, &stubAck{}, &stubAck{})

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry", "", false).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry?window=45", "", true).Code, "contract rejects a window that is not an option")
	w := do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry?window=7", "", true)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "EXPIRY_WINDOW_UNKNOWN")

	r = newRouter(t, &stubList{err: errors.New("db down")}, &stubAck{}, &stubAck{})
	assert.Equal(t, http.StatusInternalServerError, do(t, r, http.MethodGet, "/api/v1/inventory/batches/close-to-expiry", "", true).Code)
}

func TestAcknowledgeBatchExpiry_PassesTheCaller(t *testing.T) {
	ack := &stubAck{}
	r := newRouter(t, &stubList{}, ack, &stubAck{})

	w := do(t, r, http.MethodPost, "/api/v1/inventory/batches/"+batchID+"/expiry-acknowledgements", `{"window_days":30}`, true)

	assert.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	assert.Equal(t, appinventory.BatchExpiryAcknowledgementInput{BatchID: batchID, WindowDays: 30, By: "user-1"}, ack.got)
}

func TestAcknowledgeBatchExpiry_Errors(t *testing.T) {
	path := "/api/v1/inventory/batches/" + batchID + "/expiry-acknowledgements"
	ack := &stubAck{}
	r := newRouter(t, &stubList{}, ack, &stubAck{})

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodPost, path, `{"window_days":30}`, false).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPost, path, `{}`, true).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodPost, path, `{"window_days":45}`, true).Code)
	assert.False(t, ack.called)

	for err, status := range map[error]int{
		invDomain.ErrBatchNotFound:          http.StatusNotFound,
		invDomain.ErrBatchNotInExpiryWindow: http.StatusUnprocessableEntity,
		invDomain.ErrUnknownExpiryWindow:    http.StatusUnprocessableEntity,
	} {
		r := newRouter(t, &stubList{}, &stubAck{err: err}, &stubAck{})
		assert.Equal(t, status, do(t, r, http.MethodPost, path, `{"window_days":30}`, true).Code, err.Error())
	}
}

func TestUnacknowledgeBatchExpiry(t *testing.T) {
	path := "/api/v1/inventory/batches/" + batchID + "/expiry-acknowledgements/30"
	unack := &stubAck{}
	r := newRouter(t, &stubList{}, &stubAck{}, unack)

	assert.Equal(t, http.StatusNoContent, do(t, r, http.MethodDelete, path, "", true).Code)
	assert.Equal(t, appinventory.BatchExpiryAcknowledgementInput{BatchID: batchID, WindowDays: 30, By: "user-1"}, unack.got)

	assert.Equal(t, http.StatusUnauthorized, do(t, r, http.MethodDelete, path, "", false).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, r, http.MethodDelete, "/api/v1/inventory/batches/"+batchID+"/expiry-acknowledgements/45", "", true).Code)

	r = newRouter(t, &stubList{}, &stubAck{}, &stubAck{err: invDomain.ErrBatchNotFound})
	assert.Equal(t, http.StatusNotFound, do(t, r, http.MethodDelete, path, "", true).Code)
}

// The close-to-expiry routes share /inventory/batches/... with
// api/inventory; gin panics at startup when the two disagree on a wildcard.
func TestRoutesCoexistWithTheInventoryAPI(t *testing.T) {
	r := gin.New()
	g := r.Group("/api/v1")
	assert.NotPanics(t, func() {
		geninventory.RegisterHandlers(g, struct{ geninventory.ServerInterface }{})
		genstockexpiry.RegisterHandlers(g, handler.NewHandler(&stubList{}, &stubAck{}, &stubAck{}))
	})
}
