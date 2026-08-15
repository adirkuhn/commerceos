package middleware_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// captureLog redirects the standard logger's output for the duration of fn,
// then restores it -- the standard `log` package is a shared global, so
// tests can't safely run in parallel with this, but nothing else in this
// package's tests writes to it.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)
	fn()
	return buf.String()
}

func newBodyLoggerRouter(gotBody *[]byte) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestBodyLogger())
	r.POST("/test", func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		*gotBody = b
		c.Status(http.StatusOK)
	})
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequestBodyLogger_logsMethodPathAndBody(t *testing.T) {
	var gotBody []byte
	router := newBodyLoggerRouter(&gotBody)

	logOutput := captureLog(t, func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test?foo=bar", strings.NewReader(`{"hello":"world"}`))
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	require.Contains(t, logOutput, "POST")
	require.Contains(t, logOutput, "/test?foo=bar")
	require.Contains(t, logOutput, `{"hello":"world"}`)
}

func TestRequestBodyLogger_handlerStillReceivesTheFullBody(t *testing.T) {
	var gotBody []byte
	router := newBodyLoggerRouter(&gotBody)

	captureLog(t, func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"hello":"world"}`))
		router.ServeHTTP(w, req)
	})

	require.Equal(t, `{"hello":"world"}`, string(gotBody))
}

func TestRequestBodyLogger_getWithNoBodyDoesNotPanic(t *testing.T) {
	var gotBody []byte
	router := newBodyLoggerRouter(&gotBody)

	logOutput := captureLog(t, func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	})

	require.Contains(t, logOutput, "GET")
}

func TestRequestBodyLogger_truncatesOversizedBodies(t *testing.T) {
	var gotBody []byte
	router := newBodyLoggerRouter(&gotBody)

	huge := strings.Repeat("a", 9*1024)
	logOutput := captureLog(t, func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(huge))
		router.ServeHTTP(w, req)
	})

	require.Contains(t, logOutput, "...(truncated)")
	require.Less(t, len(logOutput), len(huge), "log line must be shorter than the untruncated body")
	// The handler must still see every byte -- only the log line is capped.
	require.Equal(t, huge, string(gotBody))
}
