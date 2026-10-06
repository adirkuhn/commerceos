package purchaseplanning

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) CreatePurchaseOrderFromPlan(c *gin.Context) {
	var req genpurchaseplanning.CreatePurchaseOrderFromPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}
	in := appprocurement.PlanOrderInput{SupplierID: req.SupplierId.String(), Lines: make([]appprocurement.PlanOrderLine, len(req.Lines))}
	if req.Notes != nil {
		in.Notes = *req.Notes
	}
	for i, l := range req.Lines {
		in.Lines[i] = appprocurement.PlanOrderLine{SKUID: l.SkuId.String(), Quantity: l.Quantity}
	}
	po, err := h.planOrder.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: genpurchaseplanning.CreatedPurchaseOrder{PurchaseOrderId: po.ID}})
}
