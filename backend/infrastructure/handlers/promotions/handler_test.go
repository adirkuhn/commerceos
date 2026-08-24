package promotions_test

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

	genpromotions "github.com/adirkuhn/commerceos/api/promotions"
	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	promotionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/promotions"
)

func init() { gin.SetMode(gin.TestMode) }

type stubCreatePromotion struct {
	out      *apppromotions.CreatePromotionResult
	err      error
	gotInput apppromotions.CreatePromotionInput
}

func (s *stubCreatePromotion) Execute(_ context.Context, in apppromotions.CreatePromotionInput) (*apppromotions.CreatePromotionResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &apppromotions.CreatePromotionResult{
		Promotion: promotionsDomain.Promotion{ID: "promo-1", Name: in.Name, Type: in.Type, StartsAt: in.StartsAt, Status: in.Status},
	}, nil
}

type stubUpdatePromotion struct {
	out      *promotionsDomain.Promotion
	err      error
	gotInput apppromotions.UpdatePromotionInput
}

func (s *stubUpdatePromotion) Execute(_ context.Context, in apppromotions.UpdatePromotionInput) (*promotionsDomain.Promotion, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &promotionsDomain.Promotion{ID: in.ID, Name: in.Name, Type: in.Type, StartsAt: in.StartsAt, Status: in.Status}, nil
}

type stubDeletePromotion struct {
	err   error
	gotID string
}

func (s *stubDeletePromotion) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubGetPromotion struct {
	out *apppromotions.GetPromotionResult
	err error
}

func (s *stubGetPromotion) Execute(_ context.Context, _ string) (*apppromotions.GetPromotionResult, error) {
	return s.out, s.err
}

type stubListPromotions struct {
	out []promotionsDomain.Promotion
	err error
}

func (s *stubListPromotions) Execute(_ context.Context) ([]promotionsDomain.Promotion, error) {
	return s.out, s.err
}

func newTestHandler(
	create *stubCreatePromotion, update *stubUpdatePromotion, deletePromo *stubDeletePromotion,
	get *stubGetPromotion, list *stubListPromotions,
) *promotionshandler.Handler {
	return promotionshandler.NewHandler(create, update, deletePromo, get, list)
}

func newTestRouter(h *promotionshandler.Handler) *gin.Engine {
	r := gin.New()
	genpromotions.RegisterHandlers(r.Group("/api/v1"), h)
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

func TestListPromotionsHandler_returnsTheList(t *testing.T) {
	list := &stubListPromotions{out: []promotionsDomain.Promotion{{ID: "promo-1", Name: "Summer", UpdatedAt: time.Now()}}}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/promotions", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Summer")
}

func TestCreatePromotionHandler_success(t *testing.T) {
	create := &stubCreatePromotion{}
	h := newTestHandler(create, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/promotions", map[string]any{
		"name": "Summer Acai", "type": "percentage", "starts_at": time.Now().Format(time.RFC3339),
		"lines": []map[string]any{{"sku_id": "sku-1", "discount_type": "percentage", "discount_value": 1000}},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Summer Acai", create.gotInput.Name)
	assert.Len(t, create.gotInput.Lines, 1)
	assert.Contains(t, w.Body.String(), "Summer Acai")
}

func TestCreatePromotionHandler_surfacesWarnings(t *testing.T) {
	create := &stubCreatePromotion{out: &apppromotions.CreatePromotionResult{
		Promotion: promotionsDomain.Promotion{ID: "promo-1", Name: "Summer Acai"},
		Warnings:  []string{"SKU ACAI-1L already has a manual sale price -- this promotion will override it while active"},
	}}
	h := newTestHandler(create, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/promotions", map[string]any{
		"name": "Summer Acai", "type": "percentage", "starts_at": time.Now().Format(time.RFC3339),
		"lines": []map[string]any{{"sku_id": "sku-1", "discount_type": "percentage", "discount_value": 1000}},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "already has a manual sale price")
}

func TestCreatePromotionHandler_validationError(t *testing.T) {
	create := &stubCreatePromotion{err: promotionsDomain.ErrPromotionRequiresAtLeastOneSKU}
	h := newTestHandler(create, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/promotions", map[string]any{
		"name": "Summer Acai", "type": "percentage", "starts_at": time.Now().Format(time.RFC3339), "lines": []map[string]any{},
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetPromotionHandler_success(t *testing.T) {
	get := &stubGetPromotion{out: &apppromotions.GetPromotionResult{
		Promotion: promotionsDomain.Promotion{ID: "promo-1", Name: "Summer"},
		Lines: []apppromotions.PromotionLineView{{
			PromotionSKU: promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
			SKUCode:      "ACAI-1L",
		}},
	}}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, &stubDeletePromotion{}, get, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/promotions/promo-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "sku-1")
	assert.Contains(t, w.Body.String(), "ACAI-1L")
}

func TestGetPromotionHandler_notFound(t *testing.T) {
	get := &stubGetPromotion{err: promotionsDomain.ErrPromotionNotFound}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, &stubDeletePromotion{}, get, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/promotions/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdatePromotionHandler_success(t *testing.T) {
	update := &stubUpdatePromotion{}
	h := newTestHandler(&stubCreatePromotion{}, update, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/promotions/promo-1", map[string]any{
		"name": "New Name", "type": "fixed_amount", "starts_at": time.Now().Format(time.RFC3339), "status": "active",
		"lines": []map[string]any{{"sku_id": "sku-2", "discount_type": "fixed_amount", "discount_value": 200}},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "promo-1", update.gotInput.ID)
	assert.Equal(t, "New Name", update.gotInput.Name)
}

func TestDeletePromotionHandler_success(t *testing.T) {
	deletePromo := &stubDeletePromotion{}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, deletePromo, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/promotions/promo-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "promo-1", deletePromo.gotID)
}

func TestDeletePromotionHandler_notFound(t *testing.T) {
	deletePromo := &stubDeletePromotion{err: promotionsDomain.ErrPromotionNotFound}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, deletePromo, &stubGetPromotion{}, &stubListPromotions{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/promotions/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}
