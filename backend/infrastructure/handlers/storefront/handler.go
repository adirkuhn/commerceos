// Package storefront wires CommerceOS' authenticated public-storefront API
// (see api/storefront's doc comment) to its use cases. Every route here
// authenticates via a storefront API key resolved to a tenant (and, for
// the cart/checkout routes, a specific internal sales channel) at request
// time -- there is no JWT, so this handler (not middleware.JWT) owns
// establishing tenant context, then behaves like any other handler package
// from there.
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
	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/revolut"
	"github.com/adirkuhn/commerceos/infrastructure/stripe"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type resolveAPIKeyUC interface {
	Execute(ctx context.Context, plaintextKey string) (apikeysDomain.ResolvedAPIKey, error)
}

type getCartRecommendationsUC interface {
	Execute(ctx context.Context, lines []appreports.CartLine) (*appreports.CartRecommendationsReport, error)
}

type createCartUC interface {
	Execute(ctx context.Context, salesChannelID string) (*storefrontDomain.Cart, error)
}

type getCartUC interface {
	Execute(ctx context.Context, token string) (*storefrontDomain.Cart, error)
}

type updateCartItemsUC interface {
	Execute(ctx context.Context, token string, lines []appstorefront.UpdateCartItemLineInput) (*storefrontDomain.Cart, error)
}

type checkoutCartUC interface {
	Execute(ctx context.Context, token string, in appstorefront.CheckoutInput) (*appstorefront.CheckoutResult, error)
}

type Handler struct {
	resolveAPIKey          resolveAPIKeyUC
	getCartRecommendations getCartRecommendationsUC
	createCart             createCartUC
	getCart                getCartUC
	updateCartItems        updateCartItemsUC
	checkoutCart           checkoutCartUC
}

func NewHandler(
	resolveAPIKey resolveAPIKeyUC,
	getCartRecommendations getCartRecommendationsUC,
	createCart createCartUC,
	getCart getCartUC,
	updateCartItems updateCartItemsUC,
	checkoutCart checkoutCartUC,
) *Handler {
	return &Handler{
		resolveAPIKey: resolveAPIKey, getCartRecommendations: getCartRecommendations,
		createCart: createCart, getCart: getCart, updateCartItems: updateCartItems, checkoutCart: checkoutCart,
	}
}

var _ genstorefront.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, encryptionSecret, publicBaseURL string) {
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	apiKeyRepo := repositories.NewAPIKeyRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	productRepo := repositories.NewProductRepo(db)
	channelRepo := repositories.NewSalesChannelRepo(db)
	overrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)
	cartRepo := repositories.NewStorefrontCartRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)
	taxClassRepo := repositories.NewTaxClassRepo(db)
	taxRateRepo := repositories.NewTaxRateRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	connRepo := repositories.NewConnectionRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	stockMovementLinkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	stockMovementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)

	stockLevel := appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, batchRepo, settingsRepo)
	recordMovement := appinventory.NewAllocateSaleStockUseCase(db, stockMovementRepo, batchRepo, outboxRepo)
	createSalesOrder := appoms.NewCreateSalesOrderUseCase(db, salesRepo, lineItemRepo, stockMovementLinkRepo, skuRepo, recordMovement)
	createCustomer := appcustomers.NewCreateCustomerUseCase(db, customerRepo)
	generatePaymentLink := appstorefront.NewGeneratePaymentLinkUseCase(
		db, channelRepo, connRepo, stripe.NewClientFactory(), revolut.NewClientFactory(),
		encryptionSecret, publicBaseURL,
	)

	h := NewHandler(
		appapikeys.NewResolveAPIKeyUseCase(db, apiKeyRepo),
		appreports.NewGetCartRecommendationsUseCase(db, salesRepo, settingsRepo, skuRepo, skuRepo, variantRepo, productRepo),
		appstorefront.NewCreateCartUseCase(db, channelRepo, cartRepo),
		appstorefront.NewGetCartUseCase(db, cartRepo),
		appstorefront.NewUpdateCartItemsUseCase(db, cartRepo, channelRepo, skuRepo, variantRepo, productRepo, overrideRepo, stockLevel),
		appstorefront.NewCheckoutCartUseCase(db, cartRepo, customerRepo, createCustomer, createSalesOrder, taxClassRepo, taxRateRepo, outboxRepo, generatePaymentLink),
	)
	genstorefront.RegisterHandlers(rg, h)
}

func (h *Handler) GetCartRecommendations(c *gin.Context) {
	resolved, ok := h.authenticate(c)
	if !ok {
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

	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
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

func (h *Handler) CreateCart(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.createCart.Execute(ctx, *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, genstorefront.CartResponse{Data: toGenCart(*cart)})
}

func (h *Handler) GetCart(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.getCart.Execute(ctx, token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*cart))
}

func (h *Handler) UpdateCartItems(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.UpdateCartItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	lines := make([]appstorefront.UpdateCartItemLineInput, len(req.Items))
	for i, item := range req.Items {
		lines[i] = appstorefront.UpdateCartItemLineInput{SKUCode: item.SkuCode, Quantity: int64(item.Quantity)}
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	cart, err := h.updateCartItems.Execute(ctx, token, lines)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*cart))
}

func (h *Handler) CheckoutCart(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.checkoutCart.Execute(ctx, token, appstorefront.CheckoutInput{
		Email:    string(req.Email),
		Billing:  toAddressInput(req.Billing),
		Shipping: toAddressInput(req.Shipping),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.Checkout{
		OrderNumber: result.OrderNumber, TotalMinorUnits: result.TotalMinorUnits,
		CurrencyCode: result.CurrencyCode, PaymentUrl: result.PaymentURL,
	})
}

// authenticate resolves the bearer key to a tenant only -- used by routes
// (GetCartRecommendations) that don't need a specific sales channel.
func (h *Handler) authenticate(c *gin.Context) (apikeysDomain.ResolvedAPIKey, bool) {
	key := bearerToken(c.GetHeader("Authorization"))
	if key == "" {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	resolved, err := h.resolveAPIKey.Execute(c.Request.Context(), key)
	if err != nil {
		// Deliberately the same message as the missing-header case above --
		// never lets a caller distinguish "no key" from "wrong/revoked key".
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	return resolved, true
}

// authenticateChannel additionally requires the resolved key to be scoped
// to a sales channel -- every cart/checkout route needs to know which
// internal storefront it's for (see domain/storefront's package doc
// comment), so a tenant-wide key (e.g. the WooCommerce plugin's) is
// rejected here rather than silently operating on an arbitrary channel.
func (h *Handler) authenticateChannel(c *gin.Context) (apikeysDomain.ResolvedAPIKey, bool) {
	resolved, ok := h.authenticate(c)
	if !ok {
		return resolved, false
	}
	if resolved.SalesChannelID == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "this key is not scoped to a sales channel")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	return resolved, true
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}

func toGenCart(cart storefrontDomain.Cart) *genstorefront.Cart {
	items := make([]genstorefront.CartItem, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = genstorefront.CartItem{
			SkuCode: item.SKUCode, ProductName: item.ProductName, Quantity: int(item.Quantity),
			PriceMinorUnits: item.PriceMinorUnits, CurrencyCode: item.CurrencyCode,
		}
	}
	return &genstorefront.Cart{Token: cart.Token, Items: items, SubtotalMinorUnits: cart.SubtotalMinorUnits()}
}

func toAddressInput(a genstorefront.CheckoutAddress) appstorefront.CheckoutAddressInput {
	return appstorefront.CheckoutAddressInput{
		FirstName: a.FirstName, LastName: a.LastName, Company: strPtrOr(a.Company),
		Address1: a.Address1, Address2: strPtrOr(a.Address2), City: a.City, State: strPtrOr(a.State),
		Postcode: a.Postcode, Country: a.Country, Phone: strPtrOr(a.Phone),
	}
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appreports.ErrCartEmpty),
		errors.Is(err, storefrontDomain.ErrCartEmpty),
		errors.Is(err, storefrontDomain.ErrInsufficientStock),
		errors.Is(err, storefrontDomain.ErrProductNotAvailable),
		errors.Is(err, storefrontDomain.ErrNoInternalChannel):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartAlreadyCheckedOut):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("storefront: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
