package collections_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcollections "github.com/adirkuhn/commerceos/application/collections"
	"github.com/adirkuhn/commerceos/domain/collections"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/collections"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	secret = "s"
	colID  = "11111111-1111-4111-8111-111111111111"
	prodID = "22222222-2222-4222-8222-222222222222"
)

type stub struct {
	err      error
	gotDraft collections.Draft
	gotIDs   []string
}

func (s *stub) sample() collections.Collection {
	c := collections.Collection{ID: colID, CreatedAt: time.Unix(0, 0), UpdatedAt: time.Unix(0, 0)}
	c.Draft = collections.Draft{Name: "Sale", Slug: "sale", Type: collections.TypeSmart, Match: collections.MatchAny, Sort: collections.SortNewest,
		Rules: []collections.Rule{{Field: collections.FieldAddedWithinDays, Days: 30}}}
	return c
}

type listF func(context.Context, bool, int, int) ([]collections.Collection, int64, error)

func (f listF) Execute(ctx context.Context, p bool, a, b int) ([]collections.Collection, int64, error) {
	return f(ctx, p, a, b)
}

type getF func(context.Context, string) (*appcollections.CollectionDetail, error)

func (f getF) Execute(ctx context.Context, id string) (*appcollections.CollectionDetail, error) {
	return f(ctx, id)
}

type createF func(context.Context, collections.Draft) (*collections.Collection, error)

func (f createF) Execute(ctx context.Context, d collections.Draft) (*collections.Collection, error) {
	return f(ctx, d)
}

type updateF func(context.Context, string, collections.Draft) (*collections.Collection, error)

func (f updateF) Execute(ctx context.Context, id string, d collections.Draft) (*collections.Collection, error) {
	return f(ctx, id, d)
}

type deleteF func(context.Context, string) error

func (f deleteF) Execute(ctx context.Context, id string) error { return f(ctx, id) }

type setF func(context.Context, string, []string) error

func (f setF) Execute(ctx context.Context, id string, ids []string) error { return f(ctx, id, ids) }

func router(s *stub) *gin.Engine {
	uc := handler.UseCases{
		List: listF(func(context.Context, bool, int, int) ([]collections.Collection, int64, error) {
			return []collections.Collection{s.sample()}, 1, s.err
		}),
		Get: getF(func(context.Context, string) (*appcollections.CollectionDetail, error) {
			return &appcollections.CollectionDetail{Collection: s.sample(), ProductIDs: []string{prodID}}, s.err
		}),
		Create: createF(func(_ context.Context, d collections.Draft) (*collections.Collection, error) {
			s.gotDraft = d
			c := s.sample()
			return &c, s.err
		}),
		Update: updateF(func(_ context.Context, _ string, d collections.Draft) (*collections.Collection, error) {
			s.gotDraft = d
			c := s.sample()
			return &c, s.err
		}),
		Delete:      deleteF(func(context.Context, string) error { return s.err }),
		SetProducts: setF(func(_ context.Context, _ string, ids []string) error { s.gotIDs = ids; return s.err }),
	}
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte(secret)))
	handler.Register(g, handler.NewHandler(uc))
	return r
}

func token(t *testing.T, role middleware.Role) string {
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "u", TenantID: "t", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func do(r *gin.Engine, method, path, body, tok string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCollections_authAndRoles(t *testing.T) {
	r := router(&stub{})
	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodGet, "/api/v1/collections", "", "").Code)
	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/api/v1/collections", "", token(t, middleware.RoleCustomerUser)).Code)
	assert.Equal(t, http.StatusForbidden, do(r, http.MethodPost, "/api/v1/collections", `{}`, token(t, middleware.RoleCustomerUser)).Code)
}

func TestCollections_createMapsRules(t *testing.T) {
	s := &stub{}
	body := `{"name":"Sale","slug":"sale","type":"smart","match":"any","sort":"price_asc","published":true,
		"rules":[{"field":"category","value":"c1","include_subcategories":true},{"field":"price","min_minor_units":100}]}`
	w := do(router(s), http.MethodPost, "/api/v1/collections", body, token(t, middleware.RoleCustomerOwner))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, collections.MatchAny, s.gotDraft.Match)
	assert.True(t, s.gotDraft.Published)
	require.Len(t, s.gotDraft.Rules, 2)
	assert.True(t, s.gotDraft.Rules[0].IncludeSubcategories)
	assert.Equal(t, int64(100), *s.gotDraft.Rules[1].MinMinorUnits)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, "sale", out["data"].(map[string]any)["slug"])
}

func TestCollections_errorMapping(t *testing.T) {
	owner := token(t, middleware.RoleCustomerOwner)
	cases := []struct {
		err    error
		method string
		path   string
		body   string
		status int
	}{
		{collections.ErrCollectionNotFound, http.MethodGet, "/api/v1/collections/" + colID, "", http.StatusNotFound},
		{collections.ErrSlugTaken, http.MethodPost, "/api/v1/collections", `{"name":"a","slug":"a","type":"manual","sort":"manual"}`, http.StatusConflict},
		{collections.ErrInvalidCollection, http.MethodPut, "/api/v1/collections/" + colID, `{"name":"a","slug":"a","type":"manual","sort":"manual"}`, http.StatusUnprocessableEntity},
		{collections.ErrNotManual, http.MethodPut, "/api/v1/collections/" + colID + "/products", `{"product_ids":[]}`, http.StatusConflict},
		{assert.AnError, http.MethodDelete, "/api/v1/collections/" + colID, "", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		w := do(router(&stub{err: tc.err}), tc.method, tc.path, tc.body, owner)
		assert.Equal(t, tc.status, w.Code, "%v %s", tc.err, w.Body.String())
	}
}

func TestCollections_setProductsReturnsTheCollection(t *testing.T) {
	s := &stub{}
	w := do(router(s), http.MethodPut, "/api/v1/collections/"+colID+"/products", `{"product_ids":["`+prodID+`"]}`, token(t, middleware.RoleCustomerAdmin))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{prodID}, s.gotIDs)
	assert.Contains(t, w.Body.String(), prodID)
}
