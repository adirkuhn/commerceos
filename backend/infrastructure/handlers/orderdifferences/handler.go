// Package orderdifferences wires the Order Differences API (see
// api/orderdifferences's doc comment) to its use cases.
package orderdifferences

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genorderdifferences "github.com/adirkuhn/commerceos/api/orderdifferences"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

const (
	defaultPage    = 1
	defaultPerPage = 20
)

type listUC interface {
	Execute(ctx context.Context, page, perPage int) ([]omsDomain.OpenLineDiscrepancy, int64, error)
}

type dismissUC interface {
	Execute(ctx context.Context, id string) error
}

type auditUC interface {
	Execute(ctx context.Context, in appchannels.AuditWooOrderDifferencesInput) (*appchannels.AuditWooOrderDifferencesResult, error)
}

type Handler struct {
	list    listUC
	dismiss dismissUC
	audit   auditUC
}

func NewHandler(list listUC, dismiss dismissUC, audit auditUC) *Handler {
	return &Handler{list: list, dismiss: dismiss, audit: audit}
}

var _ genorderdifferences.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository, wooFactory integrationsDomain.WooClientFactory, encryptionSecret string) {
	discrepancies := repositories.NewLineDiscrepancyRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	record := appoms.NewRecordChannelOrderDifferencesUseCase(db, salesOrderRepo, repositories.NewSalesOrderLineItemRepo(db), discrepancies, outboxRepo, time.Now)
	audit := appchannels.NewAuditWooOrderDifferencesUseCase(db, salesOrderRepo,
		appchannels.NewWooConnectionClients(db, repositories.NewConnectionRepo(db), wooFactory, encryptionSecret), record)
	genorderdifferences.RegisterHandlers(rg, NewHandler(
		appoms.NewListOpenLineDiscrepanciesUseCase(db, discrepancies),
		appoms.NewDismissLineDiscrepancyUseCase(db, discrepancies, outboxRepo, time.Now),
		audit,
	))
}

func (h *Handler) ListOrderDifferences(c *gin.Context, params genorderdifferences.ListOrderDifferencesParams) {
	page, perPage := defaultPage, defaultPerPage
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	open, total, err := h.list.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	out := genorderdifferences.OrderDifferencePage{Differences: make([]genorderdifferences.OrderDifference, len(open)), Total: total}
	for i, d := range open {
		out.Differences[i] = toGen(d)
	}
	middleware.OK(c, out)
}

func toGen(d omsDomain.OpenLineDiscrepancy) genorderdifferences.OrderDifference {
	return genorderdifferences.OrderDifference{
		Id: d.ID, SalesOrderId: d.SalesOrderID, OrderNumber: d.Order.Number,
		CustomerName:    strings.TrimSpace(d.Order.Billing.FirstName + " " + d.Order.Billing.LastName),
		OrderOccurredAt: d.Order.OccurredAt, Kind: genorderdifferences.OrderDifferenceKind(d.Kind),
		SkuCode: d.SKUCode, Name: d.Name, ChannelQuantity: d.ChannelQuantity, LocalQuantity: d.LocalQuantity,
		ChannelTotalMinorUnits: d.ChannelTotalMinorUnits, LocalTotalMinorUnits: d.LocalTotalMinorUnits,
		CurrencyCode: d.Order.CurrencyCode, DetectedAt: d.DetectedAt,
	}
}

func (h *Handler) DismissOrderDifference(c *gin.Context, id string) {
	if err := mapError(c, h.dismiss.Execute(c.Request.Context(), id)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) AuditWooOrderDifferences(c *gin.Context, id string) {
	var req genorderdifferences.AuditWooOrderDifferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}
	in := appchannels.AuditWooOrderDifferencesInput{ConnectionID: id}
	if req.Cursor != nil {
		in.Cursor = *req.Cursor
	}
	result, err := h.audit.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	errs := result.Errors
	if errs == nil {
		errs = []string{}
	}
	middleware.OK(c, genorderdifferences.AuditWooOrderDifferencesResult{
		Checked: result.Checked, Errors: errs, NextCursor: result.NextCursor,
	})
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrLineDiscrepancyNotFound), errors.Is(err, integrationsDomain.ErrConnectionNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", rootMessage(err))
	case errors.Is(err, omsDomain.ErrLineDiscrepancyNotOpen), errors.Is(err, integrationsDomain.ErrProviderNotSupported),
		errors.Is(err, appchannels.ErrInvalidBackfillCursor):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", rootMessage(err))
	default:
		log.Printf("orderdifferences: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}

// rootMessage is the domain error's own text, without the use-case
// prefixes it was wrapped in.
func rootMessage(err error) string {
	for _, known := range []error{
		omsDomain.ErrLineDiscrepancyNotFound, omsDomain.ErrLineDiscrepancyNotOpen,
		integrationsDomain.ErrConnectionNotFound, integrationsDomain.ErrProviderNotSupported, appchannels.ErrInvalidBackfillCursor,
	} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return err.Error()
}
