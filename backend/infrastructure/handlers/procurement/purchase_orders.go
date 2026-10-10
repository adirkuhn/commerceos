package procurement

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Purchase order route handlers ────────────────────────────────────────────

func (h *Handler) ListPurchaseOrders(c *gin.Context) {
	pos, err := h.listPurchaseOrders.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genprocurement.PurchaseOrder, len(pos))
	for i, po := range pos {
		resp[i] = toGenPurchaseOrder(po)
	}
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderListResponse{Data: &resp})
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	var req genprocurement.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	po, err := h.createPurchaseOrder.Execute(c.Request.Context(), appprocurement.CreatePurchaseOrderInput{
		SupplierID: &req.SupplierId, WarehouseID: req.WarehouseId, CurrencyCode: req.CurrencyCode,
		ExpectedDate: genDateToTimePtr(req.ExpectedDate), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusCreated, genprocurement.PurchaseOrderResponse{Data: &resp})
}

func (h *Handler) GetPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getPurchaseOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genprocurement.PurchaseOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenLineItem(li)
	}
	documents := make([]genprocurement.PurchaseOrderDocument, len(detail.Documents))
	for i, d := range detail.Documents {
		documents[i] = toGenDocument(d)
	}
	resp := genprocurement.PurchaseOrderDetail{
		PurchaseOrder: toGenPurchaseOrder(detail.PurchaseOrder),
		LineItems:     lineItems,
		Documents:     documents,
	}
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderDetailResponse{Data: &resp})
}

func (h *Handler) UpdatePurchaseOrder(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.UpdatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	po, err := h.updatePurchaseOrder.Execute(c.Request.Context(), appprocurement.UpdatePurchaseOrderInput{
		ID: id, SupplierID: &req.SupplierId, WarehouseID: req.WarehouseId, Status: procurementDomain.PurchaseOrderStatus(req.Status),
		CurrencyCode: req.CurrencyCode, ExpectedDate: genDateToTimePtr(req.ExpectedDate), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderResponse{Data: &resp})
}

func (h *Handler) DeletePurchaseOrder(c *gin.Context, id genprocurement.Id) {
	err := h.deletePurchaseOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

// ── Line item route handlers ─────────────────────────────────────────────────

func (h *Handler) AddPurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.AddLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	li, err := h.addLineItem.Execute(c.Request.Context(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: id, SKUID: req.SkuId, RawSKU: strPtrOr(req.RawSku), RawDescription: req.RawDescription,
		Quantity: req.Quantity, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
		UnitCostMinorUnits: req.UnitCostMinorUnits, LineType: lineTypeOr(req.LineType),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusCreated, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) UpdatePurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.UpdateLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	li, err := h.updateLineItem.Execute(c.Request.Context(), appprocurement.UpdatePurchaseOrderLineItemInput{
		ID: id, SKUID: req.SkuId, RawSKU: strPtrOr(req.RawSku), RawDescription: req.RawDescription,
		Quantity: req.Quantity, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
		UnitCostMinorUnits: req.UnitCostMinorUnits, LineType: lineTypeOr(req.LineType),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) RemovePurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	err := h.removeLineItem.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

// ── Review UI route handlers ─────────────────────────────────────────────────

func (h *Handler) LinkLineItemToSku(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.LinkLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	li, err := h.linkLineItemToSKU.Execute(c.Request.Context(), appprocurement.LinkLineItemToSKUInput{
		LineItemID: id, SKUID: req.SkuId, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) GetLandedUnitCost(c *gin.Context, id genprocurement.Id) {
	cost, err := h.getLandedUnitCost.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.LandedUnitCost{UnitCostMinorUnits: cost}
	c.JSON(http.StatusOK, genprocurement.LandedUnitCostResponse{Data: &resp})
}

func (h *Handler) CreateProductAndLinkLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.CreateProductAndLinkLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	var priceMinorUnits int64
	if req.PriceMinorUnits != nil {
		priceMinorUnits = *req.PriceMinorUnits
	}
	li, err := h.createProductAndLink.Execute(c.Request.Context(), appprocurement.CreateProductAndLinkLineItemInput{
		LineItemID: id, ProductName: req.ProductName, SKUCode: req.SkuCode, Barcode: strPtrOr(req.Barcode),
		PriceMinorUnits: priceMinorUnits, CurrencyCode: req.CurrencyCode,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) CreateSupplierAndAssignToPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.CreateSupplierAndAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	po, err := h.createSupplierAndAssign.Execute(c.Request.Context(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{
		PurchaseOrderID: id, Name: req.Name,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderResponse{Data: &resp})
}

func (h *Handler) FlagDamagedLineItem(c *gin.Context, id genprocurement.Id, lineItemId genprocurement.LineItemId) {
	var req genprocurement.FlagDamagedLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	d, err := h.flagDamagedLineItem.Execute(c.Request.Context(), id, lineItemId, req.DamagedQty, stringPtrOr(req.Note))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingDiscrepancy(*d)
	c.JSON(http.StatusCreated, genprocurement.ReceivingDiscrepancyResponse{Data: &resp})
}
