// Package quotes wires the saved quotes API (api/quotes) to its use cases.
package quotes

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genquotes "github.com/adirkuhn/commerceos/api/quotes"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type createUC interface {
	Execute(ctx context.Context, in quotes.Input) (*quotes.Quote, error)
}

type updateUC interface {
	Execute(ctx context.Context, id string, in quotes.Input) (*quotes.Quote, error)
}

type getUC interface {
	Execute(ctx context.Context, id string) (*quotes.Quote, error)
}

type listUC interface {
	Execute(ctx context.Context, in appquotes.ListQuotesInput) (*appquotes.ListQuotesResult, error)
}

type deleteUC interface {
	Execute(ctx context.Context, id string) error
}

type UseCases struct {
	Create createUC
	Update updateUC
	Get    getUC
	List   listUC
	Delete deleteUC
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ genquotes.ServerInterface = (*Handler)(nil)

// Register wires the use cases on the real adapters. No RequireRole: every
// authenticated role may work quotes (owner's call, 2026-10-23).
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	repo := repositories.NewQuoteRepo(db)
	customers := appcustomers.NewGetCustomerUseCase(db, repositories.NewCustomerRepo(db))
	genquotes.RegisterHandlers(rg, NewHandler(UseCases{
		Create: appquotes.NewCreateQuoteUseCase(db, repo, customers, outboxRepo),
		Update: appquotes.NewUpdateQuoteUseCase(db, repo, customers, outboxRepo),
		Get:    appquotes.NewGetQuoteUseCase(db, repo),
		List:   appquotes.NewListQuotesUseCase(db, repo),
		Delete: appquotes.NewDeleteQuoteUseCase(db, repo, outboxRepo),
	}))
}

func (h *Handler) ListQuotes(c *gin.Context, params genquotes.ListQuotesParams) {
	in := appquotes.ListQuotesInput{}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	if params.Search != nil {
		in.Search = *params.Search
	}
	result, err := h.uc.List.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	data := make([]genquotes.QuoteSummary, len(result.Quotes))
	for i, s := range result.Quotes {
		data[i] = toGenSummary(s)
	}
	c.JSON(http.StatusOK, genquotes.QuoteListResponse{Data: &data, Meta: &struct {
		Total int64 `json:"total"`
	}{Total: result.Total}})
}

func (h *Handler) CreateQuote(c *gin.Context) {
	in, ok := bindInput(c)
	if !ok {
		return
	}
	q, err := h.uc.Create.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	data := toGenQuote(q)
	c.JSON(http.StatusCreated, genquotes.QuoteResponse{Data: &data})
}

func (h *Handler) GetQuote(c *gin.Context, id genquotes.Id) {
	q, err := h.uc.Get.Execute(c.Request.Context(), id.String())
	if err := mapError(c, err); err != nil {
		return
	}
	data := toGenQuote(q)
	c.JSON(http.StatusOK, genquotes.QuoteResponse{Data: &data})
}

func (h *Handler) UpdateQuote(c *gin.Context, id genquotes.Id) {
	in, ok := bindInput(c)
	if !ok {
		return
	}
	q, err := h.uc.Update.Execute(c.Request.Context(), id.String(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	data := toGenQuote(q)
	c.JSON(http.StatusOK, genquotes.QuoteResponse{Data: &data})
}

func (h *Handler) DeleteQuote(c *gin.Context, id genquotes.Id) {
	if err := mapError(c, h.uc.Delete.Execute(c.Request.Context(), id.String())); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func bindInput(c *gin.Context) (quotes.Input, bool) {
	var req genquotes.QuoteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return quotes.Input{}, false
	}
	in := quotes.Input{Name: req.Name, DiscountBasisPoints: req.DiscountBasisPoints, Lines: make([]quotes.Line, len(req.Lines))}
	if req.CurrencyCode != nil {
		in.CurrencyCode = *req.CurrencyCode
	}
	if req.CustomerId != nil {
		id := req.CustomerId.String()
		in.CustomerID = &id
	}
	for i, l := range req.Lines {
		in.Lines[i] = quotes.Line{ProductID: l.ProductId.String(), SKUID: l.SkuId.String(), Quantity: l.Quantity, UnitPriceMinorUnits: l.UnitPriceMinorUnits}
	}
	return in, true
}

func toGenSummary(s quotes.Summary) genquotes.QuoteSummary {
	return genquotes.QuoteSummary{
		Id: s.ID, Number: s.Number, Reference: quotes.Reference(s.Number), Name: s.Name, Customer: toGenCustomer(s.Customer),
		CurrencyCode:        s.CurrencyCode,
		DiscountBasisPoints: s.DiscountBasisPoints, LineCount: s.LineCount, TotalMinorUnits: s.TotalMinorUnits,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func toGenQuote(q *quotes.Quote) genquotes.Quote {
	lines := make([]genquotes.QuoteLine, len(q.Lines))
	for i, l := range q.Lines {
		lines[i] = genquotes.QuoteLine{
			ProductId: l.ProductID, SkuId: l.SKUID, Quantity: l.Quantity,
			UnitPriceMinorUnits: l.UnitPriceMinorUnits, LineTotalMinorUnits: l.Quantity * l.UnitPriceMinorUnits,
		}
	}
	return genquotes.Quote{
		Id: q.ID, Number: q.Number, Reference: quotes.Reference(q.Number), Name: q.Name, Customer: toGenCustomer(q.Customer),
		CurrencyCode:        q.CurrencyCode,
		DiscountBasisPoints: q.DiscountBasisPoints, LineCount: len(q.Lines), TotalMinorUnits: q.Total(),
		CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt, Lines: lines,
	}
}

func toGenCustomer(c *quotes.CustomerRef) *genquotes.QuoteCustomer {
	if c == nil {
		return nil
	}
	return &genquotes.QuoteCustomer{Id: c.ID, Name: c.Name, Email: c.Email}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, quotes.ErrQuoteNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, customersDomain.ErrCustomerNotFound):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, quotes.ErrNameRequired), errors.Is(err, quotes.ErrNameTooLong),
		errors.Is(err, quotes.ErrInvalidDiscount), errors.Is(err, quotes.ErrInvalidCurrency),
		errors.Is(err, quotes.ErrInvalidLine), errors.Is(err, quotes.ErrDuplicateLine), errors.Is(err, quotes.ErrTooManyLines):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("quotes: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
