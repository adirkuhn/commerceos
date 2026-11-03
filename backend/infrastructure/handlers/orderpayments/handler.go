// Package orderpayments wires the admin-only (JWT) Order Payments Backfill
// API (see api/orderpayments's doc comment) to its use case.
package orderpayments

import (
	"context"
	"errors"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genorderpayments "github.com/adirkuhn/commerceos/api/orderpayments"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type backfillUC interface {
	Execute(ctx context.Context, in appchannels.BackfillWooOrderPaymentsInput) (*appchannels.BackfillWooOrderPaymentsResult, error)
}

type Handler struct {
	backfill backfillUC
}

func NewHandler(backfill backfillUC) *Handler {
	return &Handler{backfill: backfill}
}

var _ genorderpayments.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, wooFactory integrationsDomain.WooClientFactory, encryptionSecret string) {
	uc := appchannels.NewBackfillWooOrderPaymentsUseCase(
		db, repositories.NewSalesOrderRepo(db), appintegrations.NewSalesOrderOrigins(repositories.NewSalesOrderOriginRepo(db)),
		appchannels.NewWooConnectionClients(db, repositories.NewConnectionRepo(db), wooFactory, encryptionSecret),
		omshandler.NewUpdateSalesOrderUseCase(db),
	)
	genorderpayments.RegisterHandlers(rg, NewHandler(uc))
}

func (h *Handler) BackfillWooOrderPayments(c *gin.Context, id string) {
	var req genorderpayments.BackfillWooOrderPaymentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	in := appchannels.BackfillWooOrderPaymentsInput{ConnectionID: id}
	if req.Cursor != nil {
		in.Cursor = *req.Cursor
	}
	result, err := h.backfill.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	var next *string
	if result.NextCursor != "" {
		next = &result.NextCursor
	}
	errs := result.Errors
	if errs == nil {
		errs = []string{}
	}
	middleware.OK(c, genorderpayments.BackfillWooOrderPaymentsResult{
		Checked: result.Checked, Updated: result.Updated, Errors: errs, NextCursor: next,
	})
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, integrationsDomain.ErrConnectionNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, integrationsDomain.ErrProviderNotSupported), errors.Is(err, appchannels.ErrInvalidBackfillCursor):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
	case errors.Is(err, integrationsDomain.ErrWooRequestFailed):
		middleware.FailErr(c, http.StatusBadGateway, "UPSTREAM_ERROR", err)
	default:
		log.Printf("orderpayments: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
