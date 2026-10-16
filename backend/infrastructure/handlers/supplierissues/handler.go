// Package supplierissues wires the supplier issues API (see
// api/supplierissues' doc comment) to its use cases.
package supplierissues

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	gensupplierissues "github.com/adirkuhn/commerceos/api/supplierissues"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/domain/outbox"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/smtp"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/tenantmail"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listUC interface {
	Execute(ctx context.Context, purchaseOrderID string) (*appprocurement.SupplierIssueList, error)
}

type updateUC interface {
	Execute(ctx context.Context, in appprocurement.UpdateSupplierIssueStatusInput) (*appprocurement.SupplierIssueView, error)
}

type emailUC interface {
	Execute(ctx context.Context, in appprocurement.EmailSupplierIssuesInput) (*appprocurement.EmailSupplierIssuesResult, error)
}

type Handler struct {
	list   listUC
	update updateUC
	email  emailUC
}

func NewHandler(list listUC, update updateUC, email emailUC) *Handler {
	return &Handler{list: list, update: update, email: email}
}

var _ gensupplierissues.ServerInterface = (*Handler)(nil)

// Register wires the use cases on the real adapters. platformEmail is the
// fallback for tenants without their own email connection.
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository, encryptionSecret string, platformEmail messaging.Client) {
	deps := appprocurement.SupplierIssueDeps{
		PurchaseOrders: repositories.NewPurchaseOrderRepo(db), Suppliers: repositories.NewSupplierRepo(db),
		LineItems: repositories.NewPurchaseOrderLineItemRepo(db), Bookings: repositories.NewBookingRepo(db),
		Scans: repositories.NewBookingScanRepo(db), ReceivingRecords: repositories.NewReceivingRecordRepo(db),
		Discrepancies: repositories.NewReceivingDiscrepancyRepo(db), States: repositories.NewSupplierIssueStateRepo(db),
		SKUs: repositories.NewSKURepo(db), Variants: repositories.NewVariantRepo(db), Products: repositories.NewProductRepo(db),
		Outbox: outboxRepo, Now: time.Now,
	}
	mailer := appmc.NewRecordingEmailClient(
		tenantmail.New(db, repositories.NewConnectionRepo(db), encryptionSecret, smtp.NewClientFactory(), platformEmail),
		appmc.NewRecordSentEmailUseCase(db, repositories.NewSentEmailRepo(db), outboxRepo),
		mcDomain.SentEmailSourceSupplierIssues,
	)
	gensupplierissues.RegisterHandlers(rg, NewHandler(
		appprocurement.NewListSupplierIssuesUseCase(db, deps),
		appprocurement.NewUpdateSupplierIssueStatusUseCase(db, deps),
		appprocurement.NewEmailSupplierIssuesUseCase(db, deps, mailer),
	))
}

func (h *Handler) ListSupplierIssues(c *gin.Context, id gensupplierissues.Id) {
	list, err := h.list.Execute(c.Request.Context(), id.String())
	if err := mapError(c, err); err != nil {
		return
	}
	out := gensupplierissues.SupplierIssueList{
		PurchaseOrderId: list.PurchaseOrderID, BookingInProgress: list.BookingInProgress,
		Issues: make([]gensupplierissues.SupplierIssue, len(list.Issues)),
	}
	if list.Supplier != nil {
		out.Supplier = &gensupplierissues.SupplierContact{Id: list.Supplier.ID, Name: list.Supplier.Name, Email: list.Supplier.Email}
	}
	for i, v := range list.Issues {
		out.Issues[i] = toGenIssue(v)
	}
	c.JSON(http.StatusOK, gensupplierissues.SupplierIssueListResponse{Data: out})
}

func (h *Handler) UpdateSupplierIssueStatus(c *gin.Context, id gensupplierissues.Id, source gensupplierissues.SupplierIssueSource, sourceID openapi_types.UUID) {
	var req gensupplierissues.UpdateSupplierIssueStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	in := appprocurement.UpdateSupplierIssueStatusInput{
		PurchaseOrderID: id.String(),
		Ref:             procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSource(source), SourceID: sourceID.String()},
		Status:          procurementDomain.SupplierIssueStatus(req.Status),
	}
	if req.Note != nil {
		in.Note = *req.Note
	}
	view, err := h.update.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gensupplierissues.SupplierIssueResponse{Data: toGenIssue(*view)})
}

func (h *Handler) EmailSupplierIssues(c *gin.Context, id gensupplierissues.Id) {
	var req gensupplierissues.EmailSupplierIssuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	in := appprocurement.EmailSupplierIssuesInput{
		PurchaseOrderID: id.String(), To: string(req.To), Subject: req.Subject, Body: req.Body,
		Refs: make([]procurementDomain.SupplierIssueRef, len(req.Issues)),
	}
	for i, ref := range req.Issues {
		in.Refs[i] = procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSource(ref.Source), SourceID: ref.SourceId.String()}
	}
	res, err := h.email.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gensupplierissues.EmailSupplierIssuesResponse{Data: gensupplierissues.EmailSupplierIssuesResult{SentTo: res.SentTo, Claimed: res.Claimed}})
}

func toGenIssue(v appprocurement.SupplierIssueView) gensupplierissues.SupplierIssue {
	out := gensupplierissues.SupplierIssue{
		Source: gensupplierissues.SupplierIssueSource(v.Ref.Source), SourceId: v.Ref.SourceID,
		Kind: gensupplierissues.SupplierIssueKind(v.Kind), LineItemId: v.LineItemID, SkuId: v.SKUID,
		SkuCode: v.SKUCode, ProductName: v.ProductName,
		SupplierSku: v.SupplierSKU, SupplierDescription: v.SupplierDescription,
		Quantity: v.Quantity, Reason: v.Reason, ReportedBy: v.ReportedBy, OccurredAt: v.OccurredAt,
		Status: gensupplierissues.SupplierIssueStatus(v.Status), StatusNote: v.StatusNote, StatusUpdatedAt: v.StatusUpdatedAt,
	}
	if v.Kind == procurementDomain.SupplierIssueMissing {
		expected, received := v.ExpectedQty, v.ReceivedQty
		out.ExpectedQty, out.ReceivedQty = &expected, &received
	}
	return out
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, procurementDomain.ErrPurchaseOrderNotFound), errors.Is(err, procurementDomain.ErrSupplierIssueNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, procurementDomain.ErrInvalidSupplierIssueStatus), errors.Is(err, procurementDomain.ErrSupplierIssueNoteTooLong),
		errors.Is(err, procurementDomain.ErrInvalidSupplierEmail), errors.Is(err, procurementDomain.ErrSupplierEmailIncomplete),
		errors.Is(err, procurementDomain.ErrNoSupplierIssuesSelected):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, procurementDomain.ErrSupplierEmailNotSent):
		log.Printf("supplierissues: email not sent: %v", err)
		middleware.FailErr(c, http.StatusBadGateway, "SUPPLIER_EMAIL_NOT_SENT", err)
	default:
		log.Printf("supplierissues: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
