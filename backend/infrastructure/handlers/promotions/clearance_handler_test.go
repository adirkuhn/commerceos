package promotions_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func clearanceBody() map[string]any {
	return map[string]any{
		"name": "Clearance yoghurt", "type": "percentage", "starts_at": time.Now().Format(time.RFC3339),
		"lines": []map[string]any{{"sku_id": "sku-1", "batch_id": "batch-1", "discount_type": "percentage", "discount_value": 3000}},
	}
}

func TestCreatePromotionHandler_passesClearanceBatch(t *testing.T) {
	create := &stubCreatePromotion{}
	h := newTestHandler(create, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/promotions", clearanceBody())

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, create.gotInput.Lines, 1)
	require.NotNil(t, create.gotInput.Lines[0].BatchID)
	assert.Equal(t, "batch-1", *create.gotInput.Lines[0].BatchID)
}

func TestUpdatePromotionHandler_passesClearanceBatch(t *testing.T) {
	update := &stubUpdatePromotion{}
	h := newTestHandler(&stubCreatePromotion{}, update, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})
	body := clearanceBody()
	body["status"] = "active"

	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/promotions/promo-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, update.gotInput.Lines[0].BatchID)
	assert.Equal(t, "batch-1", *update.gotInput.Lines[0].BatchID)
}

func TestCreatePromotionHandler_clearanceBatchErrorsAre422(t *testing.T) {
	for _, domainErr := range []error{promotionsDomain.ErrClearanceBatchNotFound, promotionsDomain.ErrClearanceBatchFinished} {
		t.Run(domainErr.Error(), func(t *testing.T) {
			create := &stubCreatePromotion{err: domainErr}
			h := newTestHandler(create, &stubUpdatePromotion{}, &stubDeletePromotion{}, &stubGetPromotion{}, &stubListPromotions{})

			w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/promotions", clearanceBody())

			require.Equal(t, http.StatusUnprocessableEntity, w.Code)
			assert.Contains(t, w.Body.String(), domainErr.Error())
		})
	}
}

func TestGetPromotionHandler_returnsClearanceBatchAndFinishedAt(t *testing.T) {
	batchID := "batch-1"
	finishedAt := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	get := &stubGetPromotion{out: &apppromotions.GetPromotionResult{
		Promotion: promotionsDomain.Promotion{ID: "promo-1", Name: "Clearance"},
		Lines: []apppromotions.PromotionLineView{{
			PromotionSKU: promotionsDomain.PromotionSKU{ID: "line-1", SKUID: "sku-1", BatchID: &batchID, FinishedAt: &finishedAt, DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 3000},
			SKUCode:      "YOG-1",
		}},
	}}
	h := newTestHandler(&stubCreatePromotion{}, &stubUpdatePromotion{}, &stubDeletePromotion{}, get, &stubListPromotions{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/promotions/promo-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"batch_id":"batch-1"`)
	assert.Contains(t, w.Body.String(), `"finished_at":"2026-09-24T10:00:00Z"`)
}
