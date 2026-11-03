package oms

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// salesOrderOriginLister is application/integrations'
// ListSalesOrderOriginsUseCase: a response's source, external_order_id and
// order_key come from the order's origin (COM-15), the order holds none.
type salesOrderOriginLister interface {
	Execute(ctx context.Context, salesOrderIDs []string) (map[string]integrationsDomain.SalesOrderOrigin, error)
}

var errOriginsNotWired = errors.New("oms handler: sales order origins not wired (WithOrigins)")

// WithOrigins gives the handler the origin lookup every order response
// needs.
func (h *Handler) WithOrigins(origins salesOrderOriginLister) *Handler {
	h.origins = origins
	return h
}

// genSalesOrders maps orders to the API shape with their origins; false
// when the error response was already written.
func (h *Handler) genSalesOrders(c *gin.Context, orders ...omsDomain.SalesOrder) ([]genoms.SalesOrder, bool) {
	if h.origins == nil {
		_ = mapError(c, errOriginsNotWired)
		return nil, false
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	origins, err := h.origins.Execute(c.Request.Context(), ids)
	if err := mapError(c, err); err != nil {
		return nil, false
	}
	out := make([]genoms.SalesOrder, len(orders))
	for i, o := range orders {
		out[i] = toGenSalesOrder(o, origins[o.ID])
	}
	return out, true
}

// genSalesOrder is genSalesOrders for one order.
func (h *Handler) genSalesOrder(c *gin.Context, o omsDomain.SalesOrder) (genoms.SalesOrder, bool) {
	out, ok := h.genSalesOrders(c, o)
	if !ok {
		return genoms.SalesOrder{}, false
	}
	return out[0], true
}
