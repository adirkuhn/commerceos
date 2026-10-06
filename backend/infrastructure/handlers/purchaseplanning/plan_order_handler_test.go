package purchaseplanning_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	purchaseplanninghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/purchaseplanning"
)

type stubPlanOrder struct {
	err    error
	got    appprocurement.PlanOrderInput
	called bool
}

func (s *stubPlanOrder) Execute(_ context.Context, in appprocurement.PlanOrderInput) (*procurementDomain.PurchaseOrder, error) {
	s.got, s.called = in, true
	if s.err != nil {
		return nil, s.err
	}
	return &procurementDomain.PurchaseOrder{ID: "po-9"}, nil
}

const (
	supplierUUID = "6f1c0a6e-0d4b-4b55-9a51-1c3f1d0f7a11"
	skuUUID      = "0b8a1f3e-7c2d-4e6f-9a1b-2c3d4e5f6a7b"
)

func planOrderRouter(t *testing.T, uc *stubPlanOrder) *gin.Engine {
	return newRouter(t, purchaseplanninghandler.UseCases{
		Recommendations: &stubRecommendations{}, SeasonPlan: &stubSeasonPlan{}, SeasonEvents: &stubSeasonEvents{}, PlanOrder: uc,
	})
}

func postPlanOrder(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/procurement/recommendations/purchase-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreatePurchaseOrderFromPlan_Created(t *testing.T) {
	uc := &stubPlanOrder{}

	w := postPlanOrder(planOrderRouter(t, uc), fmt.Sprintf(`{"supplier_id":%q,"lines":[{"sku_id":%q,"quantity":172}],"notes":"Xmas"}`, supplierUUID, skuUUID))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{"purchase_order_id":"po-9"},"error":null}`, w.Body.String())
	assert.Equal(t, appprocurement.PlanOrderInput{
		SupplierID: supplierUUID, Notes: "Xmas", Lines: []appprocurement.PlanOrderLine{{SKUID: skuUUID, Quantity: 172}},
	}, uc.got)
}

func TestCreatePurchaseOrderFromPlan_ContractRejectsBadBodies(t *testing.T) {
	for body, status := range map[string]int{
		`{}`: http.StatusBadRequest,
		fmt.Sprintf(`{"supplier_id":%q,"lines":[]}`, supplierUUID):                                    http.StatusBadRequest,
		fmt.Sprintf(`{"supplier_id":%q,"lines":[{"sku_id":%q,"quantity":0}]}`, supplierUUID, skuUUID): http.StatusBadRequest,
		// Not a UUID: fails binding into the contract's UUID type.
		fmt.Sprintf(`{"supplier_id":"nope","lines":[{"sku_id":%q,"quantity":1}]}`, skuUUID): http.StatusUnprocessableEntity,
	} {
		uc := &stubPlanOrder{}
		w := postPlanOrder(planOrderRouter(t, uc), body)
		assert.Equal(t, status, w.Code, body)
		assert.False(t, uc.called, body)
	}
}

func TestCreatePurchaseOrderFromPlan_MapsErrors(t *testing.T) {
	body := fmt.Sprintf(`{"supplier_id":%q,"lines":[{"sku_id":%q,"quantity":1}]}`, supplierUUID, skuUUID)
	for err, status := range map[error]int{
		suppliersDomain.ErrSupplierNotFound: http.StatusNotFound,
		appprocurement.ErrSKUNotPlannable:   http.StatusUnprocessableEntity,
		appprocurement.ErrInvalidPlanOrder:  http.StatusUnprocessableEntity,
		fmt.Errorf("db down"):               http.StatusInternalServerError,
	} {
		w := postPlanOrder(planOrderRouter(t, &stubPlanOrder{err: fmt.Errorf("x: %w", err)}), body)
		assert.Equal(t, status, w.Code, err.Error())
	}
}
