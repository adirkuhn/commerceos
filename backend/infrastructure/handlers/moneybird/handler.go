// Package moneybird wires the Moneybird API (see api/moneybird) to its
// use cases.
package moneybird

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genmoneybird "github.com/adirkuhn/commerceos/api/moneybird"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/domain/quotes"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	mbclient "github.com/adirkuhn/commerceos/infrastructure/moneybird"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type connectUC interface {
	Execute(ctx context.Context, in appmb.ConnectInput) (*integrationsDomain.Connection, error)
}
type setupUC interface {
	Execute(ctx context.Context) (*appmb.Setup, error)
}
type updateSettingsUC interface {
	Execute(ctx context.Context, in appmb.UpdateSettingsInput) (*mb.Settings, error)
}
type sendUC interface {
	Execute(ctx context.Context, sourceID string) (*mb.Document, error)
}
type deleteUC interface {
	Execute(ctx context.Context, sourceType mb.SourceType, sourceID string) error
}
type statusUC interface {
	Execute(ctx context.Context, sourceType mb.SourceType, sourceID string) (*appmb.DocumentStatus, error)
}

type UseCases struct {
	Connect   connectUC
	Setup     setupUC
	Update    updateSettingsUC
	SendOrder sendUC
	SendQuote sendUC
	Status    statusUC
	Delete    deleteUC
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ genmoneybird.ServerInterface = (*Handler)(nil)

// RegisterRoutes mounts the routes: every role may read; connecting,
// changing settings and sending need an owner or admin.
func RegisterRoutes(rg gin.IRouter, h *Handler) {
	g := rg.Group("")
	g.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	genmoneybird.RegisterHandlers(g, h)
}

// Register builds the use cases on the real repositories and Moneybird.
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository, secret string) {
	w := newWiring(db, outboxRepo, secret)
	RegisterRoutes(rg, NewHandler(UseCases{
		Connect:   appmb.NewConnectUseCase(db, w.conns, w.settings, w.factory, outboxRepo, secret),
		Setup:     appmb.NewGetSetupUseCase(db, w.conns, w.settings, w.factory, secret),
		Update:    appmb.NewUpdateSettingsUseCase(db, w.conns, w.settings, w.factory, outboxRepo, secret),
		SendOrder: w.sendOrder,
		SendQuote: w.sendQuote,
		Status:    appmb.NewGetDocumentUseCase(db, w.conns, w.settings, w.docs, secret),
		Delete:    appmb.NewDeleteDocumentUseCase(db, w.conns, w.settings, w.docs, w.factory, outboxRepo, secret),
	}))
}

// NewAutoSync is the auto-sync use case for cmd/scheduler's consumers, on
// the same wiring as the HTTP send buttons.
func NewAutoSync(db *gorm.DB, outboxRepo outbox.Repository, secret string) *appmb.AutoSyncUseCase {
	w := newWiring(db, outboxRepo, secret)
	return appmb.NewAutoSyncUseCase(db, w.conns, w.settings, w.sendOrder, w.sendQuote, secret)
}

type wiring struct {
	conns     *repositories.ConnectionRepo
	settings  *repositories.MoneybirdSettingsRepo
	docs      *repositories.MoneybirdDocumentRepo
	factory   mb.ClientFactory
	sendOrder *appmb.SendSalesOrderUseCase
	sendQuote *appmb.SendQuoteUseCase
}

func newWiring(db *gorm.DB, outboxRepo outbox.Repository, secret string) wiring {
	w := wiring{
		conns: repositories.NewConnectionRepo(db), settings: repositories.NewMoneybirdSettingsRepo(db),
		docs: repositories.NewMoneybirdDocumentRepo(db), factory: mbclient.NewFactory(),
	}
	contacts := repositories.NewMoneybirdContactRepo(db)
	customers := repositories.NewCustomerRepo(db)
	w.sendOrder = appmb.NewSendSalesOrderUseCase(db, appmb.SendSalesOrderDeps{
		Connections: w.conns, Settings: w.settings, Documents: w.docs, Contacts: contacts,
		Orders: repositories.NewSalesOrderRepo(db), Lines: repositories.NewSalesOrderLineItemRepo(db),
		Payments: repositories.NewSalesOrderPaymentRepo(db), Customers: customers,
		Factory: w.factory, Outbox: outboxRepo, Secret: secret,
	})
	w.sendQuote = appmb.NewSendQuoteUseCase(db, appmb.SendQuoteDeps{
		Connections: w.conns, Settings: w.settings, Documents: w.docs, Contacts: contacts,
		Quotes: repositories.NewQuoteRepo(db), Catalog: pmshandler.NewListPriceListUseCase(db), Customers: customers,
		Factory: w.factory, Outbox: outboxRepo, Secret: secret,
	})
	return w
}

func (h *Handler) ConnectMoneybird(c *gin.Context) {
	var req genmoneybird.ConnectMoneybirdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	conn, err := h.uc.Connect.Execute(c.Request.Context(), appmb.ConnectInput{APIToken: req.ApiToken})
	if failed(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: gin.H{"connection_id": conn.ID}})
}

func (h *Handler) GetMoneybirdSetup(c *gin.Context) {
	setup, err := h.uc.Setup.Execute(c.Request.Context())
	if failed(c, err) {
		return
	}
	out := genmoneybird.MoneybirdSetup{
		Settings:        toSettings(setup.Settings),
		Administrations: make([]genmoneybird.MoneybirdAdministration, len(setup.Administrations)),
		TaxRates:        make([]genmoneybird.MoneybirdTaxRate, len(setup.TaxRates)),
		LedgerAccounts:  make([]genmoneybird.MoneybirdLedgerAccount, len(setup.LedgerAccounts)),
		PaymentMethods:  setup.PaymentMethods,
	}
	for i, a := range setup.Administrations {
		out.Administrations[i] = genmoneybird.MoneybirdAdministration{Id: a.ID, Name: a.Name, Currency: a.Currency}
	}
	for i, r := range setup.TaxRates {
		out.TaxRates[i] = genmoneybird.MoneybirdTaxRate{Id: r.ID, Name: r.Name, Percentage: r.Percentage}
	}
	for i, l := range setup.LedgerAccounts {
		out.LedgerAccounts[i] = genmoneybird.MoneybirdLedgerAccount{Id: l.ID, Name: l.Name}
	}
	if out.PaymentMethods == nil {
		out.PaymentMethods = []string{}
	}
	middleware.OK(c, out)
}

func (h *Handler) UpdateMoneybirdSettings(c *gin.Context) {
	var req genmoneybird.UpdateMoneybirdSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	in := appmb.UpdateSettingsInput{
		AdministrationID: req.AdministrationId, DefaultLedgerAccountID: req.DefaultLedgerAccountId, AutoSync: req.AutoSync,
	}
	if req.TaxRates != nil {
		rates := make([]mb.TaxRateMapping, len(*req.TaxRates))
		for i, r := range *req.TaxRates {
			rates[i] = mb.TaxRateMapping{RateBasisPoints: int32(r.RateBasisPoints), MoneybirdTaxRateID: r.MoneybirdTaxRateId}
		}
		in.TaxRates = &rates
	}
	if req.PaymentMethods != nil {
		methods := make([]mb.PaymentMethodMapping, len(*req.PaymentMethods))
		for i, m := range *req.PaymentMethods {
			methods[i] = mb.PaymentMethodMapping{Method: m.Method, MoneybirdPaymentMethod: string(m.MoneybirdPaymentMethod)}
		}
		in.PaymentMethods = &methods
	}
	s, err := h.uc.Update.Execute(c.Request.Context(), in)
	if failed(c, err) {
		return
	}
	middleware.OK(c, toSettings(*s))
}

func (h *Handler) SendSalesOrderToMoneybird(c *gin.Context, id genmoneybird.Id) {
	doc, err := h.uc.SendOrder.Execute(c.Request.Context(), id.String())
	if failed(c, err) {
		return
	}
	middleware.OK(c, toDocument(*doc))
}

func (h *Handler) SendQuoteToMoneybird(c *gin.Context, id genmoneybird.Id) {
	doc, err := h.uc.SendQuote.Execute(c.Request.Context(), id.String())
	if failed(c, err) {
		return
	}
	middleware.OK(c, toDocument(*doc))
}

func (h *Handler) GetMoneybirdSalesOrderDocument(c *gin.Context, id genmoneybird.Id) {
	h.documentStatus(c, mb.SourceSalesOrder, id.String())
}

func (h *Handler) GetMoneybirdQuoteDocument(c *gin.Context, id genmoneybird.Id) {
	h.documentStatus(c, mb.SourceQuote, id.String())
}

func (h *Handler) DeleteMoneybirdSalesOrderDocument(c *gin.Context, id genmoneybird.Id) {
	h.deleteDocument(c, mb.SourceSalesOrder, id.String())
}

func (h *Handler) DeleteMoneybirdQuoteDocument(c *gin.Context, id genmoneybird.Id) {
	h.deleteDocument(c, mb.SourceQuote, id.String())
}

func (h *Handler) deleteDocument(c *gin.Context, sourceType mb.SourceType, id string) {
	if failed(c, h.uc.Delete.Execute(c.Request.Context(), sourceType, id)) {
		return
	}
	middleware.OK(c, gin.H{"source_id": id})
}

// documentState is genmoneybird.MoneybirdDocumentState with document
// always present -- null until first sent, as the contract says.
type documentState struct {
	Document         *genmoneybird.MoneybirdDocument `json:"document"`
	AdministrationID string                          `json:"administration_id"`
	AutoSync         bool                            `json:"auto_sync"`
}

func (h *Handler) documentStatus(c *gin.Context, sourceType mb.SourceType, id string) {
	status, err := h.uc.Status.Execute(c.Request.Context(), sourceType, id)
	if failed(c, err) {
		return
	}
	out := documentState{AdministrationID: status.AdministrationID, AutoSync: status.AutoSync}
	if status.Document != nil {
		d := toDocument(*status.Document)
		out.Document = &d
	}
	middleware.OK(c, out)
}

func toSettings(s mb.Settings) genmoneybird.MoneybirdSettings {
	out := genmoneybird.MoneybirdSettings{
		ConnectionId: s.ConnectionID, AdministrationId: s.AdministrationID, AdministrationName: s.AdministrationName,
		DefaultLedgerAccountId: s.DefaultLedgerAccountID, AutoSync: s.AutoSync,
		TaxRates:       make([]genmoneybird.TaxRateMapping, len(s.TaxRates)),
		PaymentMethods: make([]genmoneybird.PaymentMethodMapping, len(s.PaymentMethods)),
	}
	for i, r := range s.TaxRates {
		out.TaxRates[i] = genmoneybird.TaxRateMapping{RateBasisPoints: int(r.RateBasisPoints), MoneybirdTaxRateId: r.MoneybirdTaxRateID}
	}
	for i, m := range s.PaymentMethods {
		out.PaymentMethods[i] = genmoneybird.PaymentMethodMapping{
			Method: m.Method, MoneybirdPaymentMethod: genmoneybird.PaymentMethodMappingMoneybirdPaymentMethod(m.MoneybirdPaymentMethod),
		}
	}
	return out
}

func toDocument(d mb.Document) genmoneybird.MoneybirdDocument {
	return genmoneybird.MoneybirdDocument{
		SourceType: genmoneybird.MoneybirdDocumentSourceType(d.SourceType), SourceId: d.SourceID,
		MoneybirdId: d.MoneybirdID, Reference: d.Reference, Status: genmoneybird.MoneybirdDocumentStatus(d.Status),
		PaymentRegistered: d.PaymentRegistered, Error: d.Error, SyncedAt: d.SyncedAt,
	}
}

// failed maps the use cases' domain errors to HTTP in one place.
func failed(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, mb.ErrNotConnected), errors.Is(err, mb.ErrDocumentNotFound),
		errors.Is(err, oms.ErrSalesOrderNotFound), errors.Is(err, quotes.ErrQuoteNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, mb.ErrDuplicateReference), errors.Is(err, mb.ErrDocumentLocked):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case errors.Is(err, mb.ErrNotConfigured), errors.Is(err, mb.ErrTokenInvalid), errors.Is(err, mb.ErrTaxRateNotMapped),
		errors.Is(err, mb.ErrPaymentMethodNotMapped), errors.Is(err, mb.ErrTotalsDontAddUp), errors.Is(err, mb.ErrNothingToBook),
		errors.Is(err, appmb.ErrUnknownMoneybirdID), errors.Is(err, appmb.ErrDuplicateMapping), errors.Is(err, quotes.ErrLineUnavailable):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
	case errors.Is(err, mb.ErrRateLimited):
		middleware.FailErr(c, http.StatusServiceUnavailable, "UNAVAILABLE", err)
	case errors.Is(err, mb.ErrRequestFailed):
		middleware.FailErr(c, http.StatusBadGateway, "BAD_GATEWAY", err)
	default:
		log.Printf("moneybird: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return true
}
