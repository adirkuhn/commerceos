package fx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genfx "github.com/adirkuhn/commerceos/api/fx"
	appfx "github.com/adirkuhn/commerceos/application/fx"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	fxhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/fx"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListExchangeRates struct {
	out []fxDomain.ExchangeRate
	err error
}

func (s *stubListExchangeRates) Execute(_ context.Context) ([]fxDomain.ExchangeRate, error) {
	return s.out, s.err
}

type stubUpsertExchangeRate struct {
	out      *fxDomain.ExchangeRate
	err      error
	gotInput appfx.UpsertExchangeRateInput
}

func (s *stubUpsertExchangeRate) Execute(_ context.Context, in appfx.UpsertExchangeRateInput) (*fxDomain.ExchangeRate, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &fxDomain.ExchangeRate{CurrencyCode: in.CurrencyCode, RateBasisPoints: in.RateBasisPoints}, nil
}

type stubDeleteExchangeRate struct {
	err         error
	gotCurrency string
}

func (s *stubDeleteExchangeRate) Execute(_ context.Context, currencyCode string) error {
	s.gotCurrency = currencyCode
	return s.err
}

func newTestHandler(list *stubListExchangeRates, upsert *stubUpsertExchangeRate, del *stubDeleteExchangeRate) *fxhandler.Handler {
	return fxhandler.NewHandler(list, upsert, del)
}

func newTestRouter(h *fxhandler.Handler) *gin.Engine {
	r := gin.New()
	genfx.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestListExchangeRatesHandler_returnsTheList(t *testing.T) {
	list := &stubListExchangeRates{out: []fxDomain.ExchangeRate{{CurrencyCode: "USD", RateBasisPoints: 10876, UpdatedAt: time.Now()}}}
	h := newTestHandler(list, &stubUpsertExchangeRate{}, &stubDeleteExchangeRate{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/fx/rates", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "USD")
	assert.Contains(t, w.Body.String(), "10876")
}

func TestUpsertExchangeRateHandler_success(t *testing.T) {
	upsert := &stubUpsertExchangeRate{}
	h := newTestHandler(&stubListExchangeRates{}, upsert, &stubDeleteExchangeRate{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/fx/rates", map[string]any{
		"currency_code": "USD", "rate_basis_points": 10876,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "USD", upsert.gotInput.CurrencyCode)
	assert.Equal(t, int64(10876), upsert.gotInput.RateBasisPoints)
}

func TestUpsertExchangeRateHandler_validationError(t *testing.T) {
	upsert := &stubUpsertExchangeRate{err: fxDomain.ErrInvalidCurrencyCode}
	h := newTestHandler(&stubListExchangeRates{}, upsert, &stubDeleteExchangeRate{})
	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/fx/rates", map[string]any{
		"currency_code": "us", "rate_basis_points": 10876,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteExchangeRateHandler_success(t *testing.T) {
	del := &stubDeleteExchangeRate{}
	h := newTestHandler(&stubListExchangeRates{}, &stubUpsertExchangeRate{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/fx/rates/USD", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "USD", del.gotCurrency)
}
