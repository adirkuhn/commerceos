// Package fx wires the admin-only (JWT) Exchange Rates API (see
// api/fx's doc comment) to its use cases.
package fx

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genfx "github.com/adirkuhn/commerceos/api/fx"
	appfx "github.com/adirkuhn/commerceos/application/fx"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listExchangeRatesUC interface {
	Execute(ctx context.Context) ([]fxDomain.ExchangeRate, error)
}

type upsertExchangeRateUC interface {
	Execute(ctx context.Context, in appfx.UpsertExchangeRateInput) (*fxDomain.ExchangeRate, error)
}

type deleteExchangeRateUC interface {
	Execute(ctx context.Context, currencyCode string) error
}

type Handler struct {
	listExchangeRates  listExchangeRatesUC
	upsertExchangeRate upsertExchangeRateUC
	deleteExchangeRate deleteExchangeRateUC
}

func NewHandler(listExchangeRates listExchangeRatesUC, upsertExchangeRate upsertExchangeRateUC, deleteExchangeRate deleteExchangeRateUC) *Handler {
	return &Handler{listExchangeRates: listExchangeRates, upsertExchangeRate: upsertExchangeRate, deleteExchangeRate: deleteExchangeRate}
}

var _ genfx.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewExchangeRateRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	h := NewHandler(
		appfx.NewListExchangeRatesUseCase(db, repo),
		appfx.NewUpsertExchangeRateUseCase(db, repo, outboxRepo),
		appfx.NewDeleteExchangeRateUseCase(db, repo, outboxRepo),
	)
	genfx.RegisterHandlers(rg, h)
}

func (h *Handler) ListExchangeRates(c *gin.Context) {
	rates, err := h.listExchangeRates.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genfx.ExchangeRate, len(rates))
	for i, r := range rates {
		out[i] = toGenExchangeRate(r)
	}
	middleware.OK(c, out)
}

func (h *Handler) UpsertExchangeRate(c *gin.Context) {
	var req genfx.UpsertExchangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	rate, err := h.upsertExchangeRate.Execute(c.Request.Context(), appfx.UpsertExchangeRateInput{
		CurrencyCode: req.CurrencyCode, RateBasisPoints: req.RateBasisPoints,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenExchangeRate(*rate))
}

func (h *Handler) DeleteExchangeRate(c *gin.Context, currencyCode string) {
	err := h.deleteExchangeRate.Execute(c.Request.Context(), currencyCode)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenExchangeRate(r fxDomain.ExchangeRate) genfx.ExchangeRate {
	return genfx.ExchangeRate{CurrencyCode: r.CurrencyCode, RateBasisPoints: r.RateBasisPoints, UpdatedAt: r.UpdatedAt}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, fxDomain.ErrInvalidCurrencyCode), errors.Is(err, fxDomain.ErrInvalidRate):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("fx: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
