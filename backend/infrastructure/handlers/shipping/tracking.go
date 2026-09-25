package shipping

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func (h *Handler) ListShipmentTrackingEvents(c *gin.Context, shipmentId openapi_types.UUID) {
	events, err := h.listTrackingEvents.Execute(c.Request.Context(), shipmentId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.ShipmentTrackingEvent, len(events))
	for i, e := range events {
		out[i] = trackingEventToResponse(e)
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}

func (h *Handler) RefreshShipmentTracking(c *gin.Context, shipmentId openapi_types.UUID) {
	event, err := h.refreshTracking.Execute(c.Request.Context(), shipmentId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trackingEventToResponse(event), "error": nil})
}

func trackingEventToResponse(e shippingDomain.ShipmentStatusEvent) genshipping.ShipmentTrackingEvent {
	source := string(e.Source)
	label := shippingDomain.MyParcelStatusLabel(e.Status)
	return genshipping.ShipmentTrackingEvent{
		Id: &e.ID, ShipmentId: &e.ShipmentID, Status: &e.Status, StatusMain: &e.StatusMain,
		StatusLabel: &label, Source: (*genshipping.ShipmentTrackingEventSource)(&source), OccurredAt: &e.OccurredAt,
	}
}
