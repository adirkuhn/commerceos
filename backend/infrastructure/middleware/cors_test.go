package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

func newCORSRouter(allowedOrigin string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(allowedOrigin))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

func TestCORS_setsAllowOriginHeaderOnGET(t *testing.T) {
	r := newCORSRouter("https://commerceos.dev.localhost")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://commerceos.dev.localhost", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "pong", w.Body.String())
}

func TestCORS_preflightOPTIONS_returns204WithoutReachingHandler(t *testing.T) {
	r := newCORSRouter("https://commerceos.dev.localhost")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.Equal(t, "https://commerceos.dev.localhost", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_allowsAuthorizationAndContentTypeHeaders(t *testing.T) {
	r := newCORSRouter("https://commerceos.dev.localhost")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}
