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
	"github.com/adirkuhn/commerceos/domain/messaging"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/revolut"
	"github.com/adirkuhn/commerceos/infrastructure/stripe"
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

// resolveEditSessionUC turns a theme-edit-session bearer token into the
// tenant/channel it authorizes -- a distinct, higher-privilege credential
// from resolveAPIKeyUC's storefront key (see
// application/theme.ResolveEditSessionUseCase's own doc comment).
type resolveEditSessionUC interface {
	Execute(ctx context.Context, token string) (themeDomain.EditSession, error)
}

// getThemeDraftUC/saveThemeDraftUC are the admin Theme Builder's own
// GetThemePageUseCase/SaveThemePageUseCase, redeclared here (interface at
// point of use, per this codebase's ISP convention) -- unlike
// getStorefrontThemeUC's read-only, editor-fields-stripped view, the
// embedded Puck editor needs the full document (editor_mode, puck_data)
// and the ability to write it back.
type getThemeDraftUC interface {
	Execute(ctx context.Context, salesChannelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error)
}

type saveThemeDraftUC interface {
	Execute(ctx context.Context, in appTheme.SaveThemePageInput) (*themeDomain.ThemePage, error)
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

type requestMagicLinkUC interface {
	Execute(ctx context.Context, in appstorefront.RequestMagicLinkInput) error
}

type verifyMagicLinkUC interface {
	Execute(ctx context.Context, salesChannelID, token string) (*storefrontDomain.CustomerSession, error)
}

type getAccountUC interface {
	Execute(ctx context.Context, salesChannelID, sessionToken string) (*appstorefront.Account, error)
}

type listAccountOrdersUC interface {
	Execute(ctx context.Context, salesChannelID, sessionToken string) ([]appstorefront.AccountOrder, error)
}

type logoutUC interface {
	Execute(ctx context.Context, sessionToken string) error
}

type createPaymentIntentUC interface {
	Execute(ctx context.Context, salesChannelID, cartToken string, shipping appstorefront.CheckoutAddressInput, shippingMethodTitle string) (*appstorefront.PaymentIntentResult, error)
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
	requestMagicLink          requestMagicLinkUC
	verifyMagicLink           verifyMagicLinkUC
	getAccount                getAccountUC
	listAccountOrders         listAccountOrdersUC
	logout                    logoutUC
	createPaymentIntent       createPaymentIntentUC
	resolveEditSession        resolveEditSessionUC
	getThemeDraft             getThemeDraftUC
	saveThemeDraft            saveThemeDraftUC
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
	requestMagicLink requestMagicLinkUC,
	verifyMagicLink verifyMagicLinkUC,
	getAccount getAccountUC,
	listAccountOrders listAccountOrdersUC,
	logout logoutUC,
	createPaymentIntent createPaymentIntentUC,
	resolveEditSession resolveEditSessionUC,
	getThemeDraft getThemeDraftUC,
	saveThemeDraft saveThemeDraftUC,
) *Handler {
	return &Handler{
		resolveAPIKey: resolveAPIKey, getCartRecommendations: getCartRecommendations,
		createCart: createCart, getCart: getCart, updateCartItems: updateCartItems, checkoutCart: checkoutCart,
		listProducts: listProducts, getProduct: getProduct, listCategories: listCategories, lookupOrder: lookupOrder,
		getStorefrontTheme: getStorefrontTheme, getShippingOptions: getShippingOptions, applyCoupon: applyCoupon,
		submitReview: submitReview, listReviews: listReviews,
		createWishlist: createWishlist, getWishlist: getWishlist, toggleWishlistItem: toggleWishlistItem,
		getProductRecommendations: getProductRecommendations,
		requestMagicLink:          requestMagicLink, verifyMagicLink: verifyMagicLink, getAccount: getAccount,
		listAccountOrders: listAccountOrders, logout: logout,
		createPaymentIntent: createPaymentIntent,
		resolveEditSession:  resolveEditSession, getThemeDraft: getThemeDraft, saveThemeDraft: saveThemeDraft,
	}
}

var _ genstorefront.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, encryptionSecret, publicBaseURL string, emailClient messaging.Client) {
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	apiKeyRepo := repositories.NewAPIKeyRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	barcodeAliasRepo := repositories.NewBarcodeAliasRepo(db)
	scaleCodeAliasRepo := repositories.NewScaleCodeAliasRepo(db)
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
	magicLinkTokenRepo := repositories.NewStorefrontMagicLinkTokenRepo(db)
	customerSessionRepo := repositories.NewStorefrontCustomerSessionRepo(db)
	skuCurrencyPriceRepo := repositories.NewSKUCurrencyPriceRepo(db)
	fxRepo := repositories.NewExchangeRateRepo(db)
	couponCurrencyRepo := repositories.NewCouponCurrencyAmountRepo(db)

	stockLevel := appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, batchRepo, settingsRepo)
	recordMovement := appinventory.NewAllocateSaleStockUseCase(db, stockMovementRepo, batchRepo, outboxRepo)
	createSalesOrder := appoms.NewCreateSalesOrderUseCase(db, salesRepo, lineItemRepo, stockMovementLinkRepo, skuRepo, recordMovement)
	createCustomer := appcustomers.NewCreateCustomerUseCase(db, customerRepo)
	generatePaymentLink := appstorefront.NewGeneratePaymentLinkUseCase(
		db, channelRepo, connRepo, stripe.NewClientFactory(), revolut.NewClientFactory(),
		encryptionSecret, publicBaseURL,
	)
	estimateShipping := appshipping.NewGetDeliveryPriceEstimateUseCase(db, connRepo, myparcel.NewCarrierClientFactory(), encryptionSecret)
	listPOSProductCatalog := apppms.NewListPOSProductCatalogUseCase(db, apppms.NewListProductsUseCase(db, productRepo), variantRepo, skuRepo, mediaRepo, barcodeAliasRepo, scaleCodeAliasRepo)
	getCartRecommendations := appreports.NewGetCartRecommendationsUseCase(db, salesRepo, settingsRepo, skuRepo, skuRepo, variantRepo, productRepo)
	verifyCardPayment := appstorefront.NewVerifyCardPaymentUseCase(db, channelRepo, connRepo, stripe.NewClientFactory(), encryptionSecret)
	checkoutCart := appstorefront.NewCheckoutCartUseCase(
		db, cartRepo, channelRepo, customerRepo, createCustomer, createSalesOrder, taxClassRepo, taxRateRepo, outboxRepo, generatePaymentLink,
		connRepo, shippingSettingsRepo, estimateShipping, couponRepo, couponCurrencyRepo, fxRepo, verifyCardPayment,
	)
	createPaymentIntent := appstorefront.NewCreatePaymentIntentUseCase(db, channelRepo, connRepo, stripe.NewClientFactory(), encryptionSecret, checkoutCart)

	h := NewHandler(
		appapikeys.NewResolveAPIKeyUseCase(db, apiKeyRepo, channelRepo),
		getCartRecommendations,
		appstorefront.NewCreateCartUseCase(db, channelRepo, cartRepo),
		appstorefront.NewGetCartUseCase(db, cartRepo, couponRepo, couponCurrencyRepo, fxRepo),
		appstorefront.NewUpdateCartItemsUseCase(db, cartRepo, channelRepo, skuRepo, variantRepo, productRepo, overrideRepo, skuCurrencyPriceRepo, fxRepo, stockLevel, couponRepo, couponCurrencyRepo),
		checkoutCart,
		appstorefront.NewListProductsUseCase(db, channelRepo, overrideRepo, listPOSProductCatalog, productRepo, categoryRepo, skuCurrencyPriceRepo, fxRepo),
		appstorefront.NewGetProductUseCase(db, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, skuCurrencyPriceRepo, fxRepo, stockLevel),
		appstorefront.NewListCategoriesUseCase(db, categoryRepo),
		appstorefront.NewLookupOrderUseCase(db, salesRepo),
		appstorefront.NewGetStorefrontThemeUseCase(appTheme.NewGetThemePageUseCase(db, themeRepo)),
		appstorefront.NewGetShippingOptionsUseCase(db, connRepo, shippingSettingsRepo, estimateShipping),
		appstorefront.NewApplyCouponUseCase(db, cartRepo, couponRepo, couponCurrencyRepo, fxRepo),
		appstorefront.NewSubmitReviewUseCase(db, channelRepo, overrideRepo, productRepo, reviewRepo),
		appstorefront.NewListReviewsUseCase(db, reviewRepo),
		appstorefront.NewCreateWishlistUseCase(db, channelRepo, wishlistRepo),
		appstorefront.NewGetWishlistUseCase(db, wishlistRepo),
		appstorefront.NewToggleWishlistItemUseCase(db, wishlistRepo, channelRepo, overrideRepo, productRepo),
		appstorefront.NewGetProductRecommendationsUseCase(db, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, skuCurrencyPriceRepo, fxRepo, getCartRecommendations),
		appstorefront.NewRequestMagicLinkUseCase(db, channelRepo, customerRepo, createCustomer, magicLinkTokenRepo, emailClient),
		appstorefront.NewVerifyMagicLinkUseCase(db, magicLinkTokenRepo, customerSessionRepo),
		appstorefront.NewGetAccountUseCase(db, customerSessionRepo, customerRepo),
		appstorefront.NewListAccountOrdersUseCase(db, customerSessionRepo, salesRepo),
		appstorefront.NewLogoutUseCase(db, customerSessionRepo),
		createPaymentIntent,
		appTheme.NewResolveEditSessionUseCase(db, repositories.NewThemeEditSessionRepo(db)),
		appTheme.NewGetThemePageUseCase(db, themeRepo),
		appTheme.NewSaveThemePageUseCase(db, themeRepo, channelRepo),
	)
	genstorefront.RegisterHandlers(rg, h)
}
