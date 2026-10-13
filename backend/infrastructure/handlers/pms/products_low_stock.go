package pms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type setProductLowStockThresholdUC interface {
	Execute(ctx context.Context, productID string, threshold *int) (*pmsDomain.Product, error)
}

func (h *Handler) SetProductLowStockThreshold(c *gin.Context, id genpms.Id) {
	var req genpms.ProductLowStockThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	product, err := h.setLowStockThreshold.Execute(c.Request.Context(), id, req.LowStockThreshold)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusOK, genpms.ProductResponse{Data: &resp})
}
