// Package bolconfirmations wires the read-only Bol Shipment Confirmations
// API (see api/bolconfirmations's doc comment) to its use case.
package bolconfirmations

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genbolconfirmations "github.com/adirkuhn/commerceos/api/bolconfirmations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listBolShipmentConfirmationsUC interface {
	Execute(ctx context.Context, salesOrderID string) ([]channelsDomain.BolShipmentConfirmation, error)
}

type Handler struct {
	list listBolShipmentConfirmationsUC
}

func NewHandler(list listBolShipmentConfirmationsUC) *Handler {
	return &Handler{list: list}
}

var _ genbolconfirmations.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	list := appchannels.NewListBolShipmentConfirmationsUseCase(db,
		repositories.NewSalesOrderRepo(db), repositories.NewBolShipmentConfirmationRepo(db))
	genbolconfirmations.RegisterHandlers(rg, NewHandler(list))
}

func (h *Handler) ListBolShipmentConfirmations(c *gin.Context, id string) {
	confirmations, err := h.list.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genbolconfirmations.BolShipmentConfirmation, len(confirmations))
	for i, conf := range confirmations {
		out[i] = toGen(conf)
	}
	middleware.OK(c, out)
}

func toGen(c channelsDomain.BolShipmentConfirmation) genbolconfirmations.BolShipmentConfirmation {
	return genbolconfirmations.BolShipmentConfirmation{
		Id: c.ID, ShipmentId: c.ShipmentID, ExternalOrderId: c.ExternalOrderID,
		Action: genbolconfirmations.BolShipmentConfirmationAction(c.Action), TransportId: c.TransportID,
		TransporterCode: c.TransporterCode, TrackAndTrace: c.TrackAndTrace, ProcessStatusId: c.ProcessStatusID,
		Status:       genbolconfirmations.BolShipmentConfirmationStatus(c.Status),
		FailureReason: genbolconfirmations.BolShipmentConfirmationFailureReason(c.FailureReason),
		ErrorMessage: c.ErrorMessage, NeedsManualAction: c.NeedsManualAction(),
		CreatedAt: c.CreatedAt, ResolvedAt: c.ResolvedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrSalesOrderNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", "sales order not found")
	default:
		log.Printf("bolconfirmations: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
