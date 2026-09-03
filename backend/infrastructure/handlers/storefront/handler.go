// Package storefront wires CommerceOS' authenticated public-storefront API
// (see api/storefront's doc comment) to its use cases. Every route here
// authenticates via a per-connection storefront key resolved to a tenant at
// request time -- there is no JWT, so this handler (not middleware.JWT)
// owns establishing tenant context, then behaves like any other handler
// package from there.
package storefront

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type resolveStorefrontKeyUC interface {
	Execute(ctx context.Context, plaintextKey string) (string, error)
}

type getCartRecommendationsUC interface {
	Execute(ctx context.Context, lines []appreports.CartLine) (*appreports.CartRecommendationsReport, error)
}

type Handler struct {
	resolveStorefrontKey   resolveStorefrontKeyUC
	getCartRecommendations getCartRecommendationsUC
}

func NewHandler(resolveStorefrontKey resolveStorefrontKeyUC, getCartRecommendations getCartRecommendationsUC) *Handler {
	return &Handler{resolveStorefrontKey: resolveStorefrontKey, getCartRecommendations: getCartRecommendations}
}

var _ genstorefront.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	connRepo := repositories.NewConnectionRepo(db)
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	productRepo := repositories.NewProductRepo(db)

	h := NewHandler(
		appintegrations.NewResolveStorefrontKeyUseCase(db, connRepo),
		appreports.NewGetCartRecommendationsUseCase(db, salesRepo, settingsRepo, skuRepo, skuRepo, variantRepo, productRepo),
	)
	genstorefront.RegisterHandlers(rg, h)
}

func (h *Handler) GetCartRecommendations(c *gin.Context) {
	key := bearerToken(c.GetHeader("Authorization"))
	if key == "" {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}

	tenantID, err := h.resolveStorefrontKey.Execute(c.Request.Context(), key)
	if err != nil {
		// Deliberately the same message as the missing-header case above --
		// never lets a caller distinguish "no key" from "wrong key".
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid storefront key")
		return
	}

	var req genstorefront.CartRecommendationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}

	lines := make([]appreports.CartLine, len(req.Items))
	for i, item := range req.Items {
		lines[i] = appreports.CartLine{SKUCode: item.SkuCode, Quantity: item.Quantity}
	}

	ctx := tenant.WithTenantID(c.Request.Context(), tenantID)
	report, err := h.getCartRecommendations.Execute(ctx, lines)
	if err := mapError(c, err); err != nil {
		return
	}

	recs := make([]genstorefront.RecommendedProduct, len(report.Recommendations))
	for i, r := range report.Recommendations {
		recs[i] = genstorefront.RecommendedProduct{
			SkuCode:           r.SKUCode,
			ProductName:       r.ProductName,
			Confidence:        r.Confidence,
			IsBundleCandidate: r.IsBundleCandidate,
		}
	}
	middleware.OK(c, genstorefront.CartRecommendationsReport{Recommendations: recs, TotalOrders: report.TotalOrders})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appreports.ErrCartEmpty):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("storefront: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
