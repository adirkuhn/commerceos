package pms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// listSKUSupplierCostsUC is application/procurement.ListSKUSupplierCostsUseCase (ISP).
type listSKUSupplierCostsUC interface {
	Execute(ctx context.Context, in appprocurement.ListSKUSupplierCostsInput) (appprocurement.SKUSupplierCosts, error)
}

const defaultSupplierCostsLimit = 50

// ListSKUSupplierCosts serves GET /pms/skus/{id}/supplier-costs (COM-11).
func (h *Handler) ListSKUSupplierCosts(c *gin.Context, id genpms.Id, params genpms.ListSKUSupplierCostsParams) {
	in := appprocurement.ListSKUSupplierCostsInput{SKUID: id, Limit: defaultSupplierCostsLimit}
	if params.SupplierId != nil {
		in.SupplierID = *params.SupplierId
	}
	if params.Cursor != nil {
		in.Cursor = *params.Cursor
	}
	if params.Limit != nil {
		in.Limit = *params.Limit
	}
	out, err := h.listSKUSupplierCosts.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.SKUSupplierCostsResponse{Data: toSKUSupplierCosts(out)})
}

func toSKUSupplierCosts(out appprocurement.SKUSupplierCosts) genpms.SKUSupplierCosts {
	name := func(id *string) *string {
		if id == nil {
			return nil
		}
		if n, ok := out.SupplierNames[*id]; ok {
			return &n
		}
		return nil
	}
	resp := genpms.SKUSupplierCosts{
		CurrencyCode: out.Currency, AverageCostMinorUnits: out.Summary.AverageMinorUnits,
		ExcludedOtherCurrency: out.Summary.ExcludedOtherCurrency,
		BySupplier:            make([]genpms.SupplierCostBySupplier, len(out.Summary.BySupplier)),
		Items:                 make([]genpms.SupplierCostEntry, len(out.Page.Items)),
		NextCursor:            out.Page.NextCursor, Total: out.Page.Total,
	}
	if l := out.Summary.Last; l != nil {
		resp.Last = &genpms.SupplierCostLast{
			LandedUnitCostMinorUnits: l.LandedUnitCostMinorUnits, ReceivedAt: l.ReceivedAt,
			PurchaseOrderId: l.PurchaseOrderID, PurchaseOrderLineItemId: l.PurchaseOrderLineItemID,
			SupplierId: l.SupplierID, SupplierName: name(l.SupplierID),
		}
	}
	for i, s := range out.Summary.BySupplier {
		resp.BySupplier[i] = genpms.SupplierCostBySupplier{
			SupplierId: s.SupplierID, SupplierName: name(s.SupplierID),
			LastLandedUnitCostMinorUnits: s.LastLandedUnitCostMinorUnits, LastReceivedAt: s.LastReceivedAt,
			AverageCostMinorUnits: s.AverageMinorUnits, Quantity: s.Quantity, Receipts: s.Receipts,
		}
	}
	for i, e := range out.Page.Items {
		resp.Items[i] = toSupplierCostEntry(e, name(e.SupplierID))
	}
	return resp
}

func toSupplierCostEntry(e procurementDomain.SupplierCostEntry, supplierName *string) genpms.SupplierCostEntry {
	return genpms.SupplierCostEntry{
		Kind: genpms.SupplierCostKind(e.Kind), Quantity: e.Quantity,
		UnitCostMinorUnits: e.UnitCostMinorUnits, LandedUnitCostMinorUnits: e.LandedUnitCostMinorUnits,
		CurrencyCode: e.CurrencyCode, ReceivedAt: e.ReceivedAt,
		PurchaseOrderId: e.PurchaseOrderID, PurchaseOrderLineItemId: e.PurchaseOrderLineItemID,
		SupplierId: e.SupplierID, SupplierName: supplierName,
	}
}
