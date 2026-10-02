package orderpayments_test

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

	genorderpayments "github.com/adirkuhn/commerceos/api/orderpayments"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	orderpaymentshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/orderpayments"
)

func init() { gin.SetMode(gin.TestMode) }

type stubBackfill struct {
	out *appchannels.BackfillWooOrderPaymentsResult
	err error
	got appchannels.BackfillWooOrderPaymentsInput
}

func (s *stubBackfill) Execute(_ context.Context, in appchannels.BackfillWooOrderPaymentsInput) (*appchannels.BackfillWooOrderPaymentsResult, error) {
	s.got = in
	return s.out, s.err
}

func newTestRouter(t *testing.T, h *orderpaymentshandler.Handler) *gin.Engine {
	t.Helper()
	spec, err := genorderpayments.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genorderpayments.RegisterHandlers(g, h)
	return r
}

func post(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/connections/conn-1/backfill-order-payments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBackfillWooOrderPayments_firstBatch(t *testing.T) {
	uc := &stubBackfill{out: &appchannels.BackfillWooOrderPaymentsResult{Checked: 25, Updated: 20, Errors: []string{"woo order 9: fetch: boom"}, NextCursor: "123_so-9"}}

	w := post(newTestRouter(t, orderpaymentshandler.NewHandler(uc)), `{}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, appchannels.BackfillWooOrderPaymentsInput{ConnectionID: "conn-1"}, uc.got)
	assert.JSONEq(t, `{"data":{"checked":25,"updated":20,"errors":["woo order 9: fetch: boom"],"next_cursor":"123_so-9"},"error":null}`, w.Body.String())
}

func TestBackfillWooOrderPayments_lastBatchHasNullCursorAndEmptyErrors(t *testing.T) {
	uc := &stubBackfill{out: &appchannels.BackfillWooOrderPaymentsResult{Checked: 3}}

	w := post(newTestRouter(t, orderpaymentshandler.NewHandler(uc)), `{"cursor":"123_so-9"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "123_so-9", uc.got.Cursor)
	assert.JSONEq(t, `{"data":{"checked":3,"updated":0,"errors":[],"next_cursor":null},"error":null}`, w.Body.String())
}

func TestBackfillWooOrderPayments_mapsErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		code int
	}{
		"connection not found": {integrationsDomain.ErrConnectionNotFound, http.StatusNotFound},
		"not a woo connection": {integrationsDomain.ErrProviderNotSupported, http.StatusUnprocessableEntity},
		"bad cursor":           {appchannels.ErrInvalidBackfillCursor, http.StatusUnprocessableEntity},
		"woo unreachable":      {integrationsDomain.ErrWooRequestFailed, http.StatusBadGateway},
		"anything else":        {errors.New("db down"), http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			w := post(newTestRouter(t, orderpaymentshandler.NewHandler(&stubBackfill{err: tc.err})), `{}`)
			assert.Equal(t, tc.code, w.Code)
		})
	}
}

func TestBackfillWooOrderPayments_contractRejectsAMalformedBody(t *testing.T) {
	uc := &stubBackfill{out: &appchannels.BackfillWooOrderPaymentsResult{}}
	w := post(newTestRouter(t, orderpaymentshandler.NewHandler(uc)), `{"cursor":42}`)
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.Less(t, w.Code, 500)
	assert.Empty(t, uc.got.ConnectionID)
}
