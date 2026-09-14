package emailintegration_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genemail "github.com/adirkuhn/commerceos/api/emailintegration"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/domain/integrations"
	emailhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/emailintegration"
)

func init() { gin.SetMode(gin.TestMode) }

type stubConnectEmail struct {
	out *integrations.Connection
	err error
	got appintegrations.ConnectEmailInput
}

func (s *stubConnectEmail) Execute(_ context.Context, in appintegrations.ConnectEmailInput) (*integrations.Connection, error) {
	s.got = in
	return s.out, s.err
}

func newTestRouter(h *emailhandler.Handler) *gin.Engine {
	r := gin.New()
	genemail.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestConnectEmailHandler_success(t *testing.T) {
	connect := &stubConnectEmail{out: &integrations.Connection{
		ID: "conn-1", Provider: integrations.ProviderEmail, Status: integrations.StatusConnected, CreatedAt: time.Now(),
	}}
	h := emailhandler.NewHandler(connect)
	body := `{"host":"smtp.mystore.com","port":"587","username":"apikey","password":"secret","from":"orders@mystore.com"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/integrations/email/connect", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "smtp.mystore.com", connect.got.Host)
	assert.Equal(t, "587", connect.got.Port)
	assert.Equal(t, "apikey", connect.got.Username)
	assert.Equal(t, "secret", connect.got.Password)
	assert.Equal(t, "orders@mystore.com", connect.got.From)
	assert.Contains(t, w.Body.String(), "conn-1")
}

func TestConnectEmailHandler_allowsMissingUsernamePassword(t *testing.T) {
	connect := &stubConnectEmail{out: &integrations.Connection{ID: "conn-1", CreatedAt: time.Now()}}
	h := emailhandler.NewHandler(connect)
	body := `{"host":"smtp.mystore.com","port":"587","from":"orders@mystore.com"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/integrations/email/connect", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", connect.got.Username)
	assert.Equal(t, "", connect.got.Password)
}

func TestConnectEmailHandler_invalidInput(t *testing.T) {
	connect := &stubConnectEmail{err: integrations.ErrEmailHostRequired}
	h := emailhandler.NewHandler(connect)
	body := `{"host":"","port":"587","from":"orders@mystore.com"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/integrations/email/connect", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
