package shipping

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) CreateShipment(c *gin.Context) {
	var req genshipping.CreateShipmentJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	in := appshipping.CreateShipmentInput{
		SalesOrderID: req.SalesOrderId.String(), ConnectionID: req.ConnectionId.String(), Carrier: shippingDomain.Carrier(req.Carrier),
		Recipient: appshipping.RecipientInput{
			Person: req.Recipient.Person, Street: req.Recipient.Street, City: req.Recipient.City, CC: req.Recipient.Cc,
		},
		UserID:            userID,
		SignatureRequired: req.SignatureRequired, OnlyRecipient: req.OnlyRecipient,
	}
	if req.Recipient.Company != nil {
		in.Recipient.Company = *req.Recipient.Company
	}
	if req.Recipient.Number != nil {
		in.Recipient.Number = *req.Recipient.Number
	}
	if req.Recipient.NumberSuffix != nil {
		in.Recipient.NumberSuffix = *req.Recipient.NumberSuffix
	}
	if req.Recipient.PostalCode != nil {
		in.Recipient.PostalCode = *req.Recipient.PostalCode
	}
	if req.Recipient.Email != nil {
		in.Recipient.Email = *req.Recipient.Email
	}
	if req.Recipient.Phone != nil {
		in.Recipient.Phone = *req.Recipient.Phone
	}
	if req.PackageType != nil {
		pt := shippingDomain.PackageType(*req.PackageType)
		in.PackageType = &pt
	}
	if req.LabelFormat != nil {
		in.LabelFormat = string(*req.LabelFormat)
	}

	shipment, err := h.createShipment.Execute(c.Request.Context(), in)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": shipmentToResponse(*shipment), "error": nil})
}

func (h *Handler) ExportMyParcelOrder(c *gin.Context, connectionId openapi_types.UUID, salesOrderId openapi_types.UUID) {
	var req genshipping.ExportMyParcelOrderJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	in := appshipping.ExportMyParcelOrderInput{
		SalesOrderID: salesOrderId.String(), ConnectionID: connectionId.String(), Carrier: shippingDomain.Carrier(req.Carrier),
		Recipient: appshipping.RecipientInput{
			Person: req.Recipient.Person, Street: req.Recipient.Street, City: req.Recipient.City, CC: req.Recipient.Cc,
		},
		UserID: userID,
	}
	if req.Recipient.Company != nil {
		in.Recipient.Company = *req.Recipient.Company
	}
	if req.Recipient.Number != nil {
		in.Recipient.Number = *req.Recipient.Number
	}
	if req.Recipient.NumberSuffix != nil {
		in.Recipient.NumberSuffix = *req.Recipient.NumberSuffix
	}
	if req.Recipient.PostalCode != nil {
		in.Recipient.PostalCode = *req.Recipient.PostalCode
	}
	if req.Recipient.Email != nil {
		in.Recipient.Email = *req.Recipient.Email
	}
	if req.Recipient.Phone != nil {
		in.Recipient.Phone = *req.Recipient.Phone
	}

	order, err := h.exportMyParcelOrder.Execute(c.Request.Context(), in)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": genshipping.MyParcelFulfilmentOrder{
		Id: &order.ID, ConnectionId: &order.ConnectionID, SalesOrderId: &order.SalesOrderID,
		MyparcelUuid: &order.MyParcelUUID, Status: &order.Status,
	}, "error": nil})
}

func (h *Handler) GetShipmentLabel(c *gin.Context, shipmentId openapi_types.UUID, params genshipping.GetShipmentLabelParams) {
	size := ""
	if params.Size != nil {
		size = string(*params.Size)
	}
	label, err := h.getLabel.Execute(c.Request.Context(), shipmentId.String(), size)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.Data(http.StatusOK, label.ContentType, label.Data)
}

func (h *Handler) ListShipmentsForOrder(c *gin.Context, salesOrderId openapi_types.UUID) {
	shipments, err := h.listShipments.Execute(c.Request.Context(), salesOrderId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.Shipment, len(shipments))
	for i, s := range shipments {
		out[i] = shipmentToResponse(s)
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}

func (h *Handler) CancelShipment(c *gin.Context, shipmentId openapi_types.UUID) {
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	result, err := h.cancelShipment.Execute(c.Request.Context(), appshipping.CancelShipmentInput{
		ShipmentID: shipmentId.String(), UserID: userID,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": genshipping.CancelShipmentResult{
		DeletedOnMyparcel: &result.DeletedOnMyParcel, CarrierError: &result.CarrierError, LabelKept: &result.LabelKept, Message: &result.Message,
	}, "error": nil})
}

func shipmentToResponse(s shippingDomain.Shipment) genshipping.Shipment {
	carrier, status := int(s.Carrier), s.Status
	resp := genshipping.Shipment{
		Id: &s.ID, SalesOrderId: &s.SalesOrderID, ConnectionId: &s.ConnectionID, Carrier: &carrier,
		Barcode: &s.Barcode, Status: &status, StatusMain: &s.StatusMain, TrackingUrl: &s.TrackingURL, LabelFormat: &s.LabelFormat,
		CreatedAt: &s.CreatedAt, CancelledAt: s.CancelledAt, CancelNote: &s.CancelNote,
		CancelDeletedOnCarrier: cancelDeletedOnCarrier(s), CancelCarrierError: cancelCarrierError(s), CancelLabelKept: cancelLabelKept(s),
		Recipient: &genshipping.RecipientAddress{
			Person: s.RecipientPerson, Company: &s.RecipientCompany, Street: s.RecipientStreet,
			Number: &s.RecipientNumber, NumberSuffix: &s.RecipientNumberSuffix, PostalCode: &s.RecipientPostalCode,
			City: s.RecipientCity, Cc: s.RecipientCC, Email: &s.RecipientEmail, Phone: &s.RecipientPhone,
		},
	}
	return resp
}

func cancelDeletedOnCarrier(s shippingDomain.Shipment) *bool {
	if s.CancelOutcome == nil {
		return nil
	}
	return &s.CancelOutcome.DeletedOnCarrier
}

func cancelLabelKept(s shippingDomain.Shipment) *bool {
	kept := s.CancelOutcome != nil && s.CancelOutcome.LabelKept
	return &kept
}

func cancelCarrierError(s shippingDomain.Shipment) *string {
	e := ""
	if s.CancelOutcome != nil {
		e = s.CancelOutcome.CarrierError
	}
	return &e
}
