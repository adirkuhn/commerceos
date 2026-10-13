package storefront

import (
	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Mirror the defaults declared on listSlowMovingProducts' query params in
// api/storefront/openapi.yaml -- the request validator checks ranges but
// doesn't inject defaults, so an omitted param arrives here as nil.
const (
	defaultSlowMovingDays  = 60
	defaultSlowMovingLimit = 12
)

func (h *Handler) ListSlowMovingProducts(c *gin.Context, params genstorefront.ListSlowMovingProductsParams) {
	resolved, ok := h.authenticate(c)
	if !ok {
		return
	}

	days, limit := defaultSlowMovingDays, defaultSlowMovingLimit
	if params.Days != nil {
		days = *params.Days
	}
	if params.Limit != nil {
		limit = *params.Limit
	}

	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.catalog.ListSlowMovingProducts.Execute(ctx, days, limit)
	if err := mapError(c, err); err != nil {
		return
	}

	products := make([]genstorefront.SlowMovingProduct, len(result.Products))
	for i, p := range result.Products {
		products[i] = genstorefront.SlowMovingProduct{SkuCode: p.SKUCode, ProductName: p.ProductName, StockOnHand: p.StockOnHand}
	}
	middleware.OK(c, genstorefront.SlowMovingProductsList{WindowDays: result.WindowDays, Products: products})
}
