// Pick list handler method lives in its own file, same reasoning as
// packing_handler.go: keeps handler.go from growing further.
package oms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type buildPickListUC interface {
	Execute(ctx context.Context, salesOrderIDs []string) (*omsDomain.PickList, error)
}

func (h *Handler) GetPickList(c *gin.Context, params genoms.GetPickListParams) {
	list, err := h.buildPickList.Execute(c.Request.Context(), params.SalesOrderId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPickList(*list)
	c.JSON(http.StatusOK, genoms.PickListResponse{Data: &resp})
}

func toGenPickList(list omsDomain.PickList) genoms.PickList {
	items := make([]genoms.PickListItem, len(list.Items))
	for i, it := range list.Items {
		orders := make([]genoms.PickListOrderLine, len(it.Orders))
		for j, o := range it.Orders {
			orders[j] = genoms.PickListOrderLine{
				SalesOrderId: o.SalesOrderID, OrderNumber: o.OrderNumber, LineItemId: o.LineItemID, Quantity: o.Quantity,
			}
		}
		items[i] = genoms.PickListItem{
			SkuId: it.SKUID, SkuCode: it.SKUCode, Barcode: it.Barcode, Name: it.Name,
			TotalQuantity: it.TotalQuantity, Orders: orders,
		}
	}
	return genoms.PickList{Items: items}
}
