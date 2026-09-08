package coupons_test

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

	gencoupons "github.com/adirkuhn/commerceos/api/coupons"
	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	couponshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/coupons"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListCoupons struct {
	out []couponsDomain.Coupon
	err error
}

func (s *stubListCoupons) Execute(_ context.Context) ([]couponsDomain.Coupon, error) {
	return s.out, s.err
}

type stubCreateCoupon struct {
	out      *couponsDomain.Coupon
	err      error
	gotInput appcoupons.CreateCouponInput
}

func (s *stubCreateCoupon) Execute(_ context.Context, in appcoupons.CreateCouponInput) (*couponsDomain.Coupon, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &couponsDomain.Coupon{ID: "coupon-1", Code: in.Code, DiscountType: in.DiscountType, DiscountValue: in.DiscountValue, Active: true}, nil
}

type stubUpdateCoupon struct {
	out      *couponsDomain.Coupon
	err      error
	gotInput appcoupons.UpdateCouponInput
}

func (s *stubUpdateCoupon) Execute(_ context.Context, in appcoupons.UpdateCouponInput) (*couponsDomain.Coupon, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &couponsDomain.Coupon{ID: in.ID, Code: in.Code, DiscountType: in.DiscountType, DiscountValue: in.DiscountValue, Active: in.Active}, nil
}

type stubDeleteCoupon struct {
	err   error
	gotID string
}

func (s *stubDeleteCoupon) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newTestHandler(list *stubListCoupons, create *stubCreateCoupon, update *stubUpdateCoupon, deleteCoupon *stubDeleteCoupon) *couponshandler.Handler {
	return couponshandler.NewHandler(list, create, update, deleteCoupon)
}

func newTestRouter(h *couponshandler.Handler) *gin.Engine {
	r := gin.New()
	gencoupons.RegisterHandlers(r.Group("/api/v1"), h)
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

func TestListCouponsHandler_returnsTheList(t *testing.T) {
	list := &stubListCoupons{out: []couponsDomain.Coupon{{ID: "coupon-1", Code: "SUMMER10", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000, UpdatedAt: time.Now()}}}
	h := newTestHandler(list, &stubCreateCoupon{}, &stubUpdateCoupon{}, &stubDeleteCoupon{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/coupons", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SUMMER10")
}

func TestCreateCouponHandler_success(t *testing.T) {
	create := &stubCreateCoupon{}
	h := newTestHandler(&stubListCoupons{}, create, &stubUpdateCoupon{}, &stubDeleteCoupon{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/coupons", map[string]any{
		"code": "SUMMER10", "discount_type": "percent", "discount_value": 1000,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "SUMMER10", create.gotInput.Code)
	assert.Contains(t, w.Body.String(), "SUMMER10")
}

func TestCreateCouponHandler_validationError(t *testing.T) {
	create := &stubCreateCoupon{err: couponsDomain.ErrCouponCodeTaken}
	h := newTestHandler(&stubListCoupons{}, create, &stubUpdateCoupon{}, &stubDeleteCoupon{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/coupons", map[string]any{
		"code": "SUMMER10", "discount_type": "percent", "discount_value": 1000,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateCouponHandler_success(t *testing.T) {
	update := &stubUpdateCoupon{}
	h := newTestHandler(&stubListCoupons{}, &stubCreateCoupon{}, update, &stubDeleteCoupon{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/coupons/coupon-1", map[string]any{
		"code": "SUMMER20", "discount_type": "percent", "discount_value": 2000, "active": true,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "coupon-1", update.gotInput.ID)
	assert.Equal(t, "SUMMER20", update.gotInput.Code)
}

func TestUpdateCouponHandler_notFound(t *testing.T) {
	update := &stubUpdateCoupon{err: couponsDomain.ErrCouponNotFound}
	h := newTestHandler(&stubListCoupons{}, &stubCreateCoupon{}, update, &stubDeleteCoupon{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/coupons/missing", map[string]any{
		"code": "SUMMER20", "discount_type": "percent", "discount_value": 2000, "active": true,
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteCouponHandler_success(t *testing.T) {
	deleteCoupon := &stubDeleteCoupon{}
	h := newTestHandler(&stubListCoupons{}, &stubCreateCoupon{}, &stubUpdateCoupon{}, deleteCoupon)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/coupons/coupon-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "coupon-1", deleteCoupon.gotID)
}

func TestDeleteCouponHandler_notFound(t *testing.T) {
	deleteCoupon := &stubDeleteCoupon{err: couponsDomain.ErrCouponNotFound}
	h := newTestHandler(&stubListCoupons{}, &stubCreateCoupon{}, &stubUpdateCoupon{}, deleteCoupon)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/coupons/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}
