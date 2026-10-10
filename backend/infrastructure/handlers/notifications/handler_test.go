package notifications_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gennotifications "github.com/adirkuhn/commerceos/api/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	notificationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/notifications"
)

type stubList struct {
	out []notificationsDomain.Notification
	err error
}

func (s *stubList) Execute(context.Context) ([]notificationsDomain.Notification, error) {
	return s.out, s.err
}

type stubMarkRead struct {
	gotID string
	err   error
}

func (s *stubMarkRead) Execute(_ context.Context, id string) error { s.gotID = id; return s.err }

type stubMarkAll struct {
	called bool
	err    error
}

func (s *stubMarkAll) Execute(context.Context) error { s.called = true; return s.err }

func router(h *notificationshandler.Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	gennotifications.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func do(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestListNotifications_returnsTypeParamsAndFallbackText(t *testing.T) {
	list := &stubList{out: []notificationsDomain.Notification{
		{ID: "n-1", Type: notificationsDomain.TypeStockCloseToExpiry, Params: map[string]any{"count": 4, "window_days": 30},
			Title: "Stock close to expiry", Message: "4 products expire within 30 days", Link: "/inventory", CreatedAt: time.Now()},
		{ID: "n-2", Type: "legacy", Title: "Old", CreatedAt: time.Now()},
	}}
	w := do(router(notificationshandler.NewHandler(list, &stubMarkRead{}, &stubMarkAll{})), http.MethodGet, "/api/v1/notifications")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"type":"inventory.stock.close_to_expiry"`)
	assert.Contains(t, body, `"params":{"count":4,"window_days":30}`)
	assert.Contains(t, body, `"title":"Stock close to expiry"`)
	assert.Contains(t, body, `"params":{}`, "a notification without params lists {} not null")
}

func TestListNotifications_errorIs500(t *testing.T) {
	w := do(router(notificationshandler.NewHandler(&stubList{err: errors.New("db down")}, &stubMarkRead{}, &stubMarkAll{})), http.MethodGet, "/api/v1/notifications")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMarkNotificationRead(t *testing.T) {
	mark := &stubMarkRead{}
	w := do(router(notificationshandler.NewHandler(&stubList{}, mark, &stubMarkAll{})), http.MethodPost, "/api/v1/notifications/n-1/read")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "n-1", mark.gotID)
}

func TestMarkAllNotificationsRead(t *testing.T) {
	all := &stubMarkAll{}
	w := do(router(notificationshandler.NewHandler(&stubList{}, &stubMarkRead{}, all)), http.MethodPost, "/api/v1/notifications/read-all")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, all.called)
}
