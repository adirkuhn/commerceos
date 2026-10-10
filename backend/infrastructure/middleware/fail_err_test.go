package middleware_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/errs"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

var errWidgetGone = errs.New("TEST_WIDGET_GONE", "widget is gone")

func failErr(err error) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	return w
}

func TestFailErr_codedErrorSendsItsCodeAndPlainSentence(t *testing.T) {
	w := failErr(fmt.Errorf("deleteWidget: %w", errWidgetGone))

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"data":null,"error":{"code":"TEST_WIDGET_GONE","message":"widget is gone"}}`, w.Body.String())
}

func TestFailErr_uncodedErrorKeepsTheFallbackCode(t *testing.T) {
	w := failErr(errors.New("something else"))

	assert.JSONEq(t, `{"data":null,"error":{"code":"NOT_FOUND","message":"something else"}}`, w.Body.String())
}

var errProviderDown = errs.NewWithCause("TEST_PROVIDER_FAILED", "provider request failed")

func TestFailErr_causeErrorAlsoSendsTheDetail(t *testing.T) {
	w := failErr(fmt.Errorf("sync: %w", fmt.Errorf("%w: 503 service unavailable", errProviderDown)))

	assert.JSONEq(t, `{"data":null,"error":{"code":"TEST_PROVIDER_FAILED","message":"provider request failed","detail":"503 service unavailable"}}`, w.Body.String())
}
