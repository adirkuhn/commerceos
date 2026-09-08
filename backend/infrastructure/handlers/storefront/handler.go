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
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	appTheme "github.com/adirkuhn/commerceos/application/theme"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
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
	Execute(ctx context.Context, token string) (*appstorefront.CartResult, error)
}

type updateCartItemsUC interface {
	Execute(ctx context.Context, token string, lines []appstorefront.UpdateCartItemLineInput) (*appstorefront.CartResult, error)
}

type applyCouponUC interface {
	Execute(ctx context.Context, token string, code string) (*appstorefront.CartResult, error)
}

type checkoutCartUC interface {
	Execute(ctx context.Context, token string, in appstorefront.CheckoutInput) (*appstorefront.CheckoutResult, error)
}

type getShippingOptionsUC interface {
	Execute(ctx context.Context, shipping appstorefront.CheckoutAddressInput) []appstorefront.ShippingOption
}

type listProductsUC interface {
	Execute(ctx context.Context, salesChannelID string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error)
}

type getProductUC interface {
	Execute(ctx context.Context, salesChannelID, productID string) (*appstorefront.ProductDetail, error)
}

type getProductRecommendationsUC interface {
	Execute(ctx context.Context, salesChannelID, productID string) ([]appstorefront.ProductSummary, error)
}

type listCategoriesUC interface {
	Execute(ctx context.Context) ([]pmsDomain.Category, error)
}

type lookupOrderUC interface {
	Execute(ctx context.Context, salesChannelID, orderNumber, email string) (*appstorefront.OrderStatus, error)
}

type getStorefrontThemeUC interface {
	Execute(ctx context.Context, salesChannelID, slug string) (*themeDomain.ThemePage, error)
}

type submitReviewUC interface {
	Execute(ctx context.Context, in appstorefront.SubmitReviewInput) (*reviewsDomain.Review, error)
}

type listReviewsUC interface {
	Execute(ctx context.Context, in appstorefront.ListReviewsInput) (*appstorefront.ReviewsResult, error)
}

type createWishlistUC interface {
	Execute(ctx context.Context, salesChannelID string) (*storefrontDomain.Wishlist, error)
}

type getWishlistUC interface {
	Execute(ctx context.Context, token string) (*storefrontDomain.Wishlist, error)
}

type toggleWishlistItemUC interface {
	Execute(ctx context.Context, token, productID string) (*storefrontDomain.Wishlist, error)
}

type Handler struct {
	resolveAPIKey             resolveAPIKeyUC
	getCartRecommendations    getCartRecommendationsUC
	createCart                createCartUC
	getCart                   getCartUC
	updateCartItems           updateCartItemsUC
	checkoutCart              checkoutCartUC
	listProducts              listProductsUC
	getProduct                getProductUC
	listCategories            listCategoriesUC
	lookupOrder               lookupOrderUC
	getStorefrontTheme        getStorefrontThemeUC
	getShippingOptions        getShippingOptionsUC
	applyCoupon               applyCouponUC
	submitReview              submitReviewUC
	listReviews               listReviewsUC
	createWishlist            createWishlistUC
	getWishlist               getWishlistUC
	toggleWishlistItem        toggleWishlistItemUC
	getProductRecommendations getProductRecommendationsUC
}

func NewHandler(
	resolveAPIKey resolveAPIKeyUC,
	getCartRecommendations getCartRecommendationsUC,
	createCart createCartUC,
	getCart getCartUC,
	updateCartItems updateCartItemsUC,
	checkoutCart checkoutCartUC,
	listProducts listProductsUC,
	getProduct getProductUC,
	listCategories listCategoriesUC,
	lookupOrder lookupOrderUC,
	getStorefrontTheme getStorefrontThemeUC,
	getShippingOptions getShippingOptionsUC,
	applyCoupon applyCouponUC,
	submitReview submitReviewUC,
	listReviews listReviewsUC,
	createWishlist createWishlistUC,
	getWishlist getWishlistUC,
	toggleWishlistItem toggleWishlistItemUC,
	getProductRecommendations getProductRecommendationsUC,
) *Handler {
	return &Handler{
		resolveAPIKey: resolveAPIKey, getCartRecommendations: getCartRecommendations,
		createCart: createCart, getCart: getCart, updateCartItems: updateCartItems, checkoutCart: checkoutCart,
		listProducts: listProducts, getProduct: getProduct, listCategories: listCategories, lookupOrder: lookupOrder,
		getStorefrontTheme: getStorefrontTheme, getShippingOptions: getShippingOptions, applyCoupon: applyCoupon,
		submitReview: submitReview, listReviews: listReviews,
		createWishlist: createWishlist, getWishlist: getWishlist, toggleWishlistItem: toggleWishlistItem,
		getProductRecommendations: getProductRecommendations,
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
	categoryRepo := repositories.NewCategoryRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	themeRepo := repositories.NewThemePageRepo(db)
	shippingSettingsRepo := repositories.NewShippingConnectionSettingsRepo(db)
	couponRepo := repositories.NewCouponRepo(db)
	reviewRepo := repositories.NewReviewRepo(db)
	wishlistRepo := repositories.NewStorefrontWishlistRepo(db)

	stockLevel := appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, batchRepo, settingsRepo)
	recordMovement := appinventory.NewAllocateSaleStockUseCase(db, stockMovementRepo, batchRepo, outboxRepo)
	createSalesOrder := appoms.NewCreateSalesOrderUseCase(db, salesRepo, lineItemRepo, stockMovementLinkRepo, skuRepo, recordMovement)
	createCustomer := appcustomers.NewCreateCustomerUseCase(db, customerRepo)
	generatePaymentLink := appstorefront.NewGeneratePaymentLinkUseCase(
		db, channelRepo, connRepo, stripe.NewClientFactory(), revolut.NewClientFactory(),
		encryptionSecret, publicBaseURL,
	)
	estimateShipping := appshipping.NewGetDeliveryPriceEstimateUseCase(db, connRepo, myparcel.NewClientFactory(), encryptionSecret)
	listPOSProductCatalog := apppms.NewListPOSProductCatalogUseCase(db, apppms.NewListProductsUseCase(db, productRepo), variantRepo, skuRepo, mediaRepo)
	getCartRecommendations := appreports.NewGetCartRecommendationsUseCase(db, salesRepo, settingsRepo, skuRepo, skuRepo, variantRepo, productRepo)

	h := NewHandler(
		appapikeys.NewResolveAPIKeyUseCase(db, apiKeyRepo),
		getCartRecommendations,
		appstorefront.NewCreateCartUseCase(db, channelRepo, cartRepo),
		appstorefront.NewGetCartUseCase(db, cartRepo, couponRepo),
		appstorefront.NewUpdateCartItemsUseCase(db, cartRepo, channelRepo, skuRepo, variantRepo, productRepo, overrideRepo, stockLevel, couponRepo),
		appstorefront.NewCheckoutCartUseCase(
			db, cartRepo, customerRepo, createCustomer, createSalesOrder, taxClassRepo, taxRateRepo, outboxRepo, generatePaymentLink,
			connRepo, shippingSettingsRepo, estimateShipping, couponRepo,
		),
		appstorefront.NewListProductsUseCase(db, channelRepo, overrideRepo, listPOSProductCatalog, productRepo, categoryRepo),
		appstorefront.NewGetProductUseCase(db, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, stockLevel),
		appstorefront.NewListCategoriesUseCase(db, categoryRepo),
		appstorefront.NewLookupOrderUseCase(db, salesRepo),
		appstorefront.NewGetStorefrontThemeUseCase(appTheme.NewGetThemePageUseCase(db, themeRepo)),
		appstorefront.NewGetShippingOptionsUseCase(db, connRepo, shippingSettingsRepo, estimateShipping),
		appstorefront.NewApplyCouponUseCase(db, cartRepo, couponRepo),
		appstorefront.NewSubmitReviewUseCase(db, channelRepo, overrideRepo, productRepo, reviewRepo),
		appstorefront.NewListReviewsUseCase(db, reviewRepo),
		appstorefront.NewCreateWishlistUseCase(db, channelRepo, wishlistRepo),
		appstorefront.NewGetWishlistUseCase(db, wishlistRepo),
		appstorefront.NewToggleWishlistItemUseCase(db, wishlistRepo, channelRepo, overrideRepo, productRepo),
		appstorefront.NewGetProductRecommendationsUseCase(db, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, getCartRecommendations),
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
	c.JSON(http.StatusCreated, genstorefront.CartResponse{Data: toGenCart(appstorefront.CartResult{Cart: cart})})
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

func (h *Handler) ApplyCoupon(c *gin.Context, token genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.applyCoupon.Execute(ctx, token, strPtrOr(req.Code))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCart(*result))
}

func (h *Handler) GetShippingOptions(c *gin.Context, _ genstorefront.CartToken) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.CheckoutAddress
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	options := h.getShippingOptions.Execute(ctx, toAddressInput(req))
	genOptions := make([]genstorefront.ShippingOption, len(options))
	for i, o := range options {
		genOptions[i] = genstorefront.ShippingOption{Title: o.Title, AmountMinorUnits: o.AmountMinorUnits, CurrencyCode: o.CurrencyCode}
	}
	middleware.OK(c, genOptions)
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
		Email: string(req.Email), Billing: toAddressInput(req.Billing), Shipping: toAddressInput(req.Shipping),
		ShippingMethodTitle: strPtrOr(req.ShippingMethodTitle),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.Checkout{
		OrderNumber: result.OrderNumber, TotalMinorUnits: result.TotalMinorUnits,
		CurrencyCode: result.CurrencyCode, PaymentUrl: result.PaymentURL,
	})
}

func (h *Handler) ListStorefrontProducts(c *gin.Context, params genstorefront.ListStorefrontProductsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	in := appstorefront.ListProductsInput{CategoryID: params.CategoryId}
	if params.Search != nil {
		in.Search = *params.Search
	}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.listProducts.Execute(ctx, *resolved.SalesChannelID, in)
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genstorefront.StorefrontProductSummary, len(result.Products))
	for i, p := range result.Products {
		items[i] = toGenProductSummary(p)
	}
	facets := make([]genstorefront.StorefrontCategoryFacet, len(result.CategoryFacets))
	for i, f := range result.CategoryFacets {
		facets[i] = genstorefront.StorefrontCategoryFacet{CategoryId: f.CategoryID, CategoryName: f.CategoryName, Count: f.Count}
	}
	total := result.Total
	middleware.OK(c, struct {
		Items          []genstorefront.StorefrontProductSummary `json:"items"`
		Total          int64                                    `json:"total"`
		CategoryFacets []genstorefront.StorefrontCategoryFacet  `json:"category_facets"`
	}{Items: items, Total: total, CategoryFacets: facets})
}

func (h *Handler) GetStorefrontProduct(c *gin.Context, id string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	detail, err := h.getProduct.Execute(ctx, *resolved.SalesChannelID, id)
	if errors.Is(err, storefrontDomain.ErrProductNotAvailable) {
		// A GET-by-id 404s for "doesn't exist" -- unlike a cart line's SKU
		// code (a request-body field, so the same error is a 422 there),
		// this is a plain resource lookup.
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenProductDetail(*detail))
}

func (h *Handler) GetStorefrontProductRecommendations(c *gin.Context, id string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	summaries, err := h.getProductRecommendations.Execute(ctx, *resolved.SalesChannelID, id)
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genstorefront.StorefrontProductSummary, len(summaries))
	for i, p := range summaries {
		items[i] = toGenProductSummary(p)
	}
	middleware.OK(c, items)
}

func (h *Handler) ListStorefrontReviews(c *gin.Context, id string, params genstorefront.ListStorefrontReviewsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	in := appstorefront.ListReviewsInput{SalesChannelID: *resolved.SalesChannelID, ProductID: id}
	if params.PerPage != nil {
		in.Limit = *params.PerPage
	}
	if params.Page != nil && in.Limit > 0 {
		in.Offset = (*params.Page - 1) * in.Limit
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	result, err := h.listReviews.Execute(ctx, in)
	if err := mapError(c, err); err != nil {
		return
	}
	reviews := make([]genstorefront.Review, len(result.Reviews))
	for i, rv := range result.Reviews {
		reviews[i] = toGenReview(rv)
	}
	middleware.OK(c, genstorefront.ReviewList{
		Reviews: reviews,
		Summary: genstorefront.ReviewSummary{Average: result.Summary.Average, Count: result.Summary.Count},
	})
}

func (h *Handler) SubmitStorefrontReview(c *gin.Context, id string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	review, err := h.submitReview.Execute(ctx, appstorefront.SubmitReviewInput{
		SalesChannelID: *resolved.SalesChannelID, ProductID: id,
		CustomerName: req.CustomerName, CustomerEmail: string(req.CustomerEmail), Rating: req.Rating,
		Title: strPtrOr(req.Title), Body: req.Body,
	})
	if errors.Is(err, storefrontDomain.ErrProductNotAvailable) {
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenReview(*review)})
}

func toGenReview(rv reviewsDomain.Review) genstorefront.Review {
	return genstorefront.Review{
		Id: rv.ID, CustomerName: rv.CustomerName, Rating: rv.Rating, Title: rv.Title, Body: rv.Body,
		Verified: rv.Verified, CreatedAt: rv.CreatedAt,
	}
}

func (h *Handler) CreateWishlist(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.createWishlist.Execute(ctx, *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func (h *Handler) GetWishlist(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.getWishlist.Execute(ctx, token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func (h *Handler) ToggleWishlistItem(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.ToggleWishlistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.toggleWishlistItem.Execute(ctx, token, req.ProductId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func toGenWishlist(w storefrontDomain.Wishlist) genstorefront.Wishlist {
	productIDs := w.ProductIDs
	if productIDs == nil {
		productIDs = []string{}
	}
	return genstorefront.Wishlist{Token: w.Token, ProductIds: productIDs}
}

func (h *Handler) ListStorefrontCategories(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	categories, err := h.listCategories.Execute(ctx)
	if err := mapError(c, err); err != nil {
		return
	}
	genCategories := make([]genstorefront.StorefrontCategory, len(categories))
	for i, cat := range categories {
		genCategories[i] = genstorefront.StorefrontCategory{
			Id: cat.ID, ParentId: cat.ParentID, Name: cat.Name, Slug: cat.Slug, Position: cat.Position,
		}
	}
	middleware.OK(c, genCategories)
}

func (h *Handler) LookupStorefrontOrder(c *gin.Context, params genstorefront.LookupStorefrontOrderParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	status, err := h.lookupOrder.Execute(ctx, *resolved.SalesChannelID, params.OrderNumber, string(params.Email))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.StorefrontOrderStatus{
		OrderNumber: status.OrderNumber, Status: string(status.Status), Fulfilled: status.Fulfilled,
		TotalMinorUnits: status.TotalMinorUnits, CurrencyCode: status.CurrencyCode,
	})
}

func (h *Handler) GetStorefrontTheme(c *gin.Context, slug genstorefront.StorefrontPageSlug) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	page, err := h.getStorefrontTheme.Execute(ctx, *resolved.SalesChannelID, string(slug))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenStorefrontThemePage(*page))
}

func toGenStorefrontThemePage(p themeDomain.ThemePage) genstorefront.StorefrontThemePage {
	blocks := make([]genstorefront.StorefrontBlock, len(p.Blocks))
	for i, b := range p.Blocks {
		blocks[i] = toGenStorefrontBlock(b)
	}
	return genstorefront.StorefrontThemePage{Slug: genstorefront.StorefrontPageSlug(p.Slug), CanvasWidth: p.CanvasWidth, Blocks: blocks, CustomCss: p.CustomCSS}
}

func toGenStorefrontBlock(b themeDomain.Block) genstorefront.StorefrontBlock {
	layout := genstorefront.StorefrontBlockLayout{Desktop: genstorefront.StorefrontRect{
		X: b.Layout.Desktop.X, Y: b.Layout.Desktop.Y, Width: b.Layout.Desktop.Width, Height: b.Layout.Desktop.Height,
	}}
	if b.Layout.Mobile != nil {
		layout.Mobile = &genstorefront.StorefrontRect{X: b.Layout.Mobile.X, Y: b.Layout.Mobile.Y, Width: b.Layout.Mobile.Width, Height: b.Layout.Mobile.Height}
	}
	props := b.Props
	if props == nil {
		props = map[string]any{}
	}
	return genstorefront.StorefrontBlock{Id: b.ID, Type: genstorefront.StorefrontBlockType(b.Type), Layout: layout, Props: props}
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

func toGenCart(result appstorefront.CartResult) *genstorefront.Cart {
	cart := result.Cart
	items := make([]genstorefront.CartItem, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = genstorefront.CartItem{
			SkuCode: item.SKUCode, ProductName: item.ProductName, Quantity: int(item.Quantity),
			PriceMinorUnits: item.PriceMinorUnits, CurrencyCode: item.CurrencyCode,
		}
	}
	return &genstorefront.Cart{
		Token: cart.Token, Items: items, SubtotalMinorUnits: cart.SubtotalMinorUnits(),
		CouponCode: cart.CouponCode, DiscountMinorUnits: result.DiscountMinorUnits,
	}
}

func toAddressInput(a genstorefront.CheckoutAddress) appstorefront.CheckoutAddressInput {
	return appstorefront.CheckoutAddressInput{
		FirstName: a.FirstName, LastName: a.LastName, Company: strPtrOr(a.Company),
		Address1: a.Address1, Address2: strPtrOr(a.Address2), City: a.City, State: strPtrOr(a.State),
		Postcode: a.Postcode, Country: a.Country, Phone: strPtrOr(a.Phone),
	}
}

func toGenProductSummary(p appstorefront.ProductSummary) genstorefront.StorefrontProductSummary {
	return genstorefront.StorefrontProductSummary{
		Id: p.ID, Name: p.Name, Slug: p.Slug, CategoryId: p.CategoryID, Tags: p.Tags,
		ImageUrl: p.ImageURL, PriceMinorUnits: p.PriceMinorUnits, SalePriceMinorUnits: p.SalePriceMinorUnits,
		CurrencyCode: p.CurrencyCode,
	}
}

func toGenProductDetail(d appstorefront.ProductDetail) genstorefront.StorefrontProduct {
	variants := make([]genstorefront.StorefrontVariant, len(d.Variants))
	for i, v := range d.Variants {
		skus := make([]genstorefront.StorefrontSKU, len(v.SKUs))
		for j, s := range v.SKUs {
			skus[j] = genstorefront.StorefrontSKU{
				Id: s.ID, SkuCode: s.SKUCode, PriceMinorUnits: s.PriceMinorUnits,
				SalePriceMinorUnits: s.SalePriceMinorUnits, CurrencyCode: s.CurrencyCode, StockQuantity: s.StockQuantity,
			}
		}
		variants[i] = genstorefront.StorefrontVariant{Id: v.ID, Name: v.Name, Skus: skus}
	}
	return genstorefront.StorefrontProduct{
		Id: d.ID, Name: d.Name, Slug: d.Slug, Description: d.Description, CategoryId: d.CategoryID,
		Tags: d.Tags, Images: d.Images, Variants: variants,
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
		errors.Is(err, storefrontDomain.ErrNoInternalChannel),
		errors.Is(err, couponsDomain.ErrCouponNotRedeemable),
		errors.Is(err, couponsDomain.ErrCouponMinSubtotalNotMet),
		errors.Is(err, reviewsDomain.ErrCustomerNameRequired),
		errors.Is(err, reviewsDomain.ErrCustomerEmailRequired),
		errors.Is(err, reviewsDomain.ErrInvalidRating),
		errors.Is(err, reviewsDomain.ErrReviewBodyRequired),
		errors.Is(err, themeDomain.ErrInvalidPageSlug):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartNotFound), errors.Is(err, storefrontDomain.ErrOrderNotFound),
		errors.Is(err, storefrontDomain.ErrWishlistNotFound), errors.Is(err, couponsDomain.ErrCouponNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartAlreadyCheckedOut):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("storefront: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
