package newsletter_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/newsletter"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const subID = "11111111-1111-4111-8111-111111111111"

type listF func(context.Context, appnewsletter.ListInput) ([]newsletter.Subscriber, int64, error)

func (f listF) Execute(ctx context.Context, in appnewsletter.ListInput) ([]newsletter.Subscriber, int64, error) {
	return f(ctx, in)
}

type exportF func(context.Context, io.Writer) error

func (f exportF) Execute(ctx context.Context, w io.Writer) error { return f(ctx, w) }

type eraseF func(context.Context, string) error

func (f eraseF) Execute(ctx context.Context, id string) error { return f(ctx, id) }

func router(t *testing.T, got *appnewsletter.ListInput, eraseErr error) *gin.Engine {
	uc := handler.UseCases{
		List: listF(func(_ context.Context, in appnewsletter.ListInput) ([]newsletter.Subscriber, int64, error) {
			*got = in
			return []newsletter.Subscriber{{ID: subID, Email: "a@x.io", Status: newsletter.StatusConfirmed}}, 1, nil
		}),
		Export: exportF(func(_ context.Context, w io.Writer) error {
			_, err := io.WriteString(w, "email,confirmed_at\na@x.io,\n")
			return err
		}),
		Erase: eraseF(func(context.Context, string) error { return eraseErr }),
	}
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte("s")))
	handler.Register(g, handler.NewHandler(uc))
	return r
}

func token(t *testing.T, role middleware.Role) string {
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "u", TenantID: "t", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte("s"))
	require.NoError(t, err)
	return s
}

func get(r *gin.Engine, method, path, tok string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestNewsletterAdmin(t *testing.T) {
	var got appnewsletter.ListInput
	r := router(t, &got, nil)

	assert.Equal(t, http.StatusForbidden, get(r, http.MethodGet, "/api/v1/newsletter/subscribers", token(t, middleware.RoleCustomerUser)).Code, "subscriber emails are personal data")

	w := get(r, http.MethodGet, "/api/v1/newsletter/subscribers?status=confirmed&search=a&page=2", token(t, middleware.RoleCustomerAdmin))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, newsletter.StatusConfirmed, *got.Status)
	assert.Equal(t, 2, got.Page)
	assert.Contains(t, w.Body.String(), `"email":"a@x.io"`)

	w = get(r, http.MethodGet, "/api/v1/newsletter/subscribers/export", token(t, middleware.RoleCustomerOwner))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Contains(t, w.Body.String(), "a@x.io")

	assert.Equal(t, http.StatusOK, get(r, http.MethodDelete, "/api/v1/newsletter/subscribers/"+subID, token(t, middleware.RoleCustomerOwner)).Code)
	assert.Equal(t, http.StatusNotFound, get(router(t, &got, newsletter.ErrSubscriberNotFound), http.MethodDelete, "/api/v1/newsletter/subscribers/"+subID, token(t, middleware.RoleCustomerOwner)).Code)
}
